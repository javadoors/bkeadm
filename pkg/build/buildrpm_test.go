/*
 *
 * Copyright (c) 2025 Bocloud Technologies Co., Ltd.
 * installer is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          <http://license.coscl.org.cn/MulanPSL2>
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 *
 */

package build

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

const (
	testOneValue   = 1
	testThreeValue = 3
	testEightValue = 8
)

func TestRpmOptionsStruct(t *testing.T) {
	// Test that the RpmOptions struct has the expected fields
	ro := &RpmOptions{}

	// Check that it embeds root.Options
	_ = &ro.Options

	// Test that the fields exist
	ro.Source = "source-path"
	ro.Add = "centos/7/amd64"
	ro.Registry = "registry.example.com"
	ro.Package = "package-name"

	assert.Equal(t, "source-path", ro.Source)
	assert.Equal(t, "centos/7/amd64", ro.Add)
	assert.Equal(t, "registry.example.com", ro.Registry)
	assert.Equal(t, "package-name", ro.Package)
}

func TestAddsMap(t *testing.T) {
	// Test that the adds map has the expected values
	expected := map[string]string{
		"centos/7/amd64":  "CentOS/7/amd64",
		"centos/7/arm64":  "CentOS/7/arm64",
		"centos/8/amd64":  "CentOS/8/amd64",
		"centos/8/arm64":  "CentOS/8/arm64",
		"ubuntu/22/amd64": "Ubuntu/22/amd64",
		"ubuntu/22/arm64": "Ubuntu/22/arm64",
		"kylin/v10/arm64": "Kylin/V10/arm64",
		"kylin/v10/amd64": "Kylin/V10/amd64",
	}

	assert.Equal(t, expected, adds)
}

func TestValidateAddOption(t *testing.T) {
	tests := []struct {
		name     string
		add      string
		expected bool
	}{
		{
			name:     "valid add option",
			add:      "centos/7/amd64",
			expected: true,
		},
		{
			name:     "invalid add option",
			add:      "invalid/option",
			expected: false,
		},
		{
			name:     "case insensitive",
			add:      "CENTOS/7/AMD64",
			expected: false, // because the function is case sensitive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateAddOption(tt.add)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidatePackageDirectory(t *testing.T) {
	tests := []struct {
		name        string
		pack        string
		add         string
		mockIsDir   func(string) bool
		mockReadDir func(string) ([]os.DirEntry, error)
		mockExists  func(string) bool
		expectError bool
	}{
		{
			name:      "valid directory",
			pack:      "/valid/dir",
			add:       "centos/7/amd64",
			mockIsDir: func(path string) bool { return true },
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "subdir", isDir: true},
				}, nil
			},
			expectError: false,
		},
		{
			name:        "not a directory",
			pack:        "/invalid/path",
			add:         "centos/7/amd64",
			mockIsDir:   func(path string) bool { return false },
			expectError: true,
		},
		{
			name:      "read dir error",
			pack:      "/valid/dir",
			add:       "centos/7/amd64",
			mockIsDir: func(path string) bool { return true },
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, fmt.Errorf("read error")
			},
			expectError: true,
		},
		{
			name:      "contains non-directory file",
			pack:      "/valid/dir",
			add:       "centos/7/amd64",
			mockIsDir: func(path string) bool { return true },
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "file.txt", isDir: false},
				}, nil
			},
			expectError: true,
		},
		{
			name:      "centos with modules.yaml (allowed)",
			pack:      "/valid/dir",
			add:       "centos/7/amd64",
			mockIsDir: func(path string) bool { return true },
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "modules.yaml", isDir: false},
					&mockDirEntry{name: "subdir", isDir: true},
				}, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.IsDir, tt.mockIsDir)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			err := validatePackageDirectory(tt.pack, tt.add)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetAbsolutePath(t *testing.T) {
	tests := []struct {
		name      string
		inputPath string
		expected  string
	}{
		{
			name:      "absolute path error",
			inputPath: "relative/path",
			expected:  `relative/path`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getAbsolutePath(tt.inputPath)

			assert.NoError(t, err)
			assert.NotEqual(t, tt.expected, result)
		})
	}
}

func TestBuildRpm(t *testing.T) {
	tests := []struct {
		name                 string
		source               string
		add                  string
		pkg                  string
		isDockerEnv          bool
		validateAddResult    bool
		validatePackageError error
		getAbsSourceError    error
		getAbsPackageError   error
		expectConsoleOutput  bool
	}{
		{
			name:                "no arguments shows console output",
			source:              "",
			add:                 "",
			pkg:                 "",
			isDockerEnv:         true,
			expectConsoleOutput: true,
		},
		{
			name:                 "valid build with docker env",
			source:               "/source/path",
			add:                  "centos/7/amd64",
			pkg:                  "package-name",
			isDockerEnv:          true,
			validateAddResult:    true,
			validatePackageError: nil,
			getAbsSourceError:    nil,
			getAbsPackageError:   nil,
		},
		{
			name:              "invalid add option",
			source:            "/source/path",
			add:               "",
			pkg:               "",
			isDockerEnv:       true,
			validateAddResult: false,
		},
		{
			name:        "not in docker environment",
			source:      "/source/path",
			add:         "centos/7/amd64",
			pkg:         "package-name",
			isDockerEnv: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ro := &RpmOptions{
				Source:  tt.source,
				Add:     tt.add,
				Package: tt.pkg,
			}

			// Apply patches
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
				return tt.isDockerEnv
			})
			defer patches.Reset()

			if tt.source != "" || tt.add != "" || tt.pkg != "" {
				patches = gomonkey.ApplyFunc(validateAddOption, func(add string) bool {
					return tt.validateAddResult
				})
				defer patches.Reset()

				if tt.validatePackageError != nil {
					patches = gomonkey.ApplyFunc(validatePackageDirectory, func(pack string, add string) error {
						return tt.validatePackageError
					})
					defer patches.Reset()
				}

				if tt.getAbsSourceError != nil {
					patches = gomonkey.ApplyFunc(getAbsolutePath, func(path string) (string, error) {
						if path == tt.source {
							return "", tt.getAbsSourceError
						}
						return path, nil
					})
					defer patches.Reset()
				}

				if tt.getAbsPackageError != nil {
					patches = gomonkey.ApplyFunc(getAbsolutePath, func(path string) (string, error) {
						if path == tt.pkg {
							return "", tt.getAbsPackageError
						}
						return path, nil
					})
					defer patches.Reset()
				}
			}

			if tt.expectConsoleOutput {
				patches = gomonkey.ApplyFunc(consoleOutputStruct, func() {})
				defer patches.Reset()
			} else {
				patches = gomonkey.ApplyFunc((*RpmOptions).executeBuild, func(ro *RpmOptions) {})
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			ro.Build()

			// The function should complete without panic
			assert.True(t, true)
		})
	}
}

func TestExecuteBuild(t *testing.T) {
	tests := []struct {
		name                 string
		source               string
		add                  string
		packagePath          string
		registry             string
		mockRmpBuild         func(string, string, string) error
		mockRpmBuildPackage  func(string, string)
		mockRpmPackageAddOne func(string, string, string, string)
	}{
		{
			name:        "build with empty source - calls rmpBuild",
			source:      "",
			add:         "centos/7/amd64",
			packagePath: "/package/path",
			registry:    "registry.example.com",
			mockRmpBuild: func(registry, add, absPath string) error {
				assert.Equal(t, "registry.example.com", registry)
				assert.Equal(t, "centos/7/amd64", add)
				assert.Equal(t, "/package/path", absPath)
				return nil
			},
			mockRpmBuildPackage:  func(source, registry string) {},
			mockRpmPackageAddOne: func(source, registry, add, pack string) {},
		},
		{
			name:        "build with source but no add and package - calls rpmBuildPackage",
			source:      "/source/path",
			add:         "",
			packagePath: "",
			registry:    "registry.example.com",
			mockRmpBuild: func(registry, add, absPath string) error {
				return nil
			},
			mockRpmBuildPackage: func(source, registry string) {
				assert.Equal(t, "/source/path", source)
				assert.Equal(t, "registry.example.com", registry)
			},
			mockRpmPackageAddOne: func(source, registry, add, pack string) {},
		},
		{
			name:        "build with source, add and package - calls rpmPackageAddOne",
			source:      "/source/path",
			add:         "centos/7/amd64",
			packagePath: "/package/path",
			registry:    "registry.example.com",
			mockRmpBuild: func(registry, add, absPath string) error {
				return nil
			},
			mockRpmBuildPackage: func(source, registry string) {},
			mockRpmPackageAddOne: func(source, registry, add, pack string) {
				assert.Equal(t, "/source/path", source)
				assert.Equal(t, "registry.example.com", registry)
				assert.Equal(t, "centos/7/amd64", add)
				assert.Equal(t, "/package/path", pack)
			},
		},
		{
			name:        "rmpBuild returns error",
			source:      "",
			add:         "centos/7/amd64",
			packagePath: "/package/path",
			registry:    "registry.example.com",
			mockRmpBuild: func(registry, add, absPath string) error {
				return fmt.Errorf("build error")
			},
			mockRpmBuildPackage:  func(source, registry string) {},
			mockRpmPackageAddOne: func(source, registry, add, pack string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ro := &RpmOptions{
				Source:   tt.source,
				Add:      tt.add,
				Package:  tt.packagePath,
				Registry: tt.registry,
			}

			// Apply patches
			patches := gomonkey.ApplyFunc(rmpBuild, tt.mockRmpBuild)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(rpmBuildPackage, tt.mockRpmBuildPackage)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(rpmPackageAddOne, tt.mockRpmPackageAddOne)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			ro.executeBuild()

			// The function should complete without panic
			assert.True(t, true)
		})
	}
}

func TestPrepareWorkspace(t *testing.T) {
	tests := []struct {
		name          string
		mockRemoveAll func(string) error
		mockMkdirAll  func(string, os.FileMode) error
		expectError   bool
	}{
		{
			name:          "successful workspace preparation",
			mockRemoveAll: func(path string) error { return nil },
			mockMkdirAll:  func(path string, perm os.FileMode) error { return nil },
			expectError:   false,
		},
		{
			name:          "remove all fails",
			mockRemoveAll: func(path string) error { return fmt.Errorf("remove error") },
			mockMkdirAll:  func(path string, perm os.FileMode) error { return nil },
			expectError:   true,
		},
		{
			name:          "mkdir all fails",
			mockRemoveAll: func(path string) error { return nil },
			mockMkdirAll:  func(path string, perm os.FileMode) error { return fmt.Errorf("mkdir error") },
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			patches := gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.MkdirAll, tt.mockMkdirAll)
			defer patches.Reset()

			err := prepareWorkspace()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConsoleOutputStruct(t *testing.T) {
	// Capture the output of consoleOutputStruct
	patches := gomonkey.ApplyFunc(fmt.Print, func(a ...interface{}) (n int, err error) {
		output := a[0].(string)
		// Check that the output contains expected elements
		assert.Contains(t, output, "rpm")
		assert.Contains(t, output, "CentOS")
		assert.Contains(t, output, "Ubuntu")
		assert.Contains(t, output, "Kylin")
		assert.Contains(t, output, "files")
		return len(output), nil
	})
	defer patches.Reset()

	consoleOutputStruct()
}

func TestRmpBuild(t *testing.T) {
	tests := []struct {
		name          string
		add           string
		mockBuildFunc func(string, string) error
		expectError   bool
	}{
		{
			name:          "build for centos/7/amd64",
			add:           "centos/7/amd64",
			mockBuildFunc: func(registry string, mnt string) error { return nil },
			expectError:   false,
		},
		{
			name:          "build for centos/8/amd64",
			add:           "centos/8/amd64",
			mockBuildFunc: func(registry string, mnt string) error { return nil },
			expectError:   false,
		},
		{
			name:          "build for ubuntu/22/amd64",
			add:           "ubuntu/22/amd64",
			mockBuildFunc: func(registry string, mnt string) error { return nil },
			expectError:   false,
		},
		{
			name:          "build for kylin/v10/amd64",
			add:           "kylin/v10/amd64",
			mockBuildFunc: func(registry string, mnt string) error { return nil },
			expectError:   false,
		},
		{
			name:        "unsupported add option",
			add:         "unsupported/option",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockBuildFunc != nil {
				var patches *gomonkey.Patches
				switch tt.add {
				case "centos/7/amd64", "centos/7/arm64":
					patches = gomonkey.ApplyFunc(rpmCentos7Build, tt.mockBuildFunc)
				case "centos/8/amd64", "centos/8/arm64":
					patches = gomonkey.ApplyFunc(rpmCentos8Build, tt.mockBuildFunc)
				case "ubuntu/22/amd64", "ubuntu/22/arm64":
					patches = gomonkey.ApplyFunc(rpmUbuntu22Build, tt.mockBuildFunc)
				case "kylin/v10/amd64", "kylin/v10/arm64":
					patches = gomonkey.ApplyFunc(rpmKylinV10Build, tt.mockBuildFunc)
				}
				if patches != nil {
					defer patches.Reset()
				}
			}

			patches := gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := rmpBuild("registry.example.com", tt.add, "/mnt/path")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetTargets(t *testing.T) {
	targets := getTargets()

	// Check that we have the expected number of targets
	assert.Len(t, targets, testEightValue)

	// Check that each target has the expected properties
	expectedTargets := []target{
		{"Centos", "7", "amd64", rpmCentos7Build},
		{"Centos", "7", "arm64", rpmCentos7Build},
		{"Centos", "8", "amd64", rpmCentos8Build},
		{"Centos", "8", "arm64", rpmCentos8Build},
		{"Ubuntu", "22", "amd64", rpmUbuntu22Build},
		{"Ubuntu", "22", "arm64", rpmUbuntu22Build},
		{"Kylin", "V10", "amd64", rpmKylinV10Build},
		{"Kylin", "V10", "arm64", rpmKylinV10Build},
	}

	for i, expected := range expectedTargets {
		assert.Equal(t, expected.osName, targets[i].osName)
		assert.Equal(t, expected.version, targets[i].version)
		assert.Equal(t, expected.arch, targets[i].arch)
		// We can't directly compare function pointers, so we'll just check they're not nil
		assert.NotNil(t, targets[i].builder)
	}
}

func TestExecuteSingleTarget(t *testing.T) {
	// Test with a mock builder function
	mockBuilder := func(registry string, targetPath string) error {
		// Verify the parameters are passed correctly
		return nil
	}

	tgt := target{
		osName:  "Centos",
		version: "7",
		arch:    "amd64",
		builder: mockBuilder,
	}

	err := executeSingleTarget("test-registry", tgt)
	assert.NoError(t, err)
}

func TestRpmBuildAllArchitectures(t *testing.T) {
	// Test with a mock executeSingleTarget that tracks calls
	callCount := 0

	// Apply patch to override executeSingleTarget to increment callCount
	patches := gomonkey.ApplyFunc(executeSingleTarget, func(registry string, tgt target) error {
		callCount++
		return nil
	})
	defer patches.Reset()

	err := rpmBuildAllArchitectures("test-registry")
	assert.NoError(t, err)
	assert.Equal(t, testEightValue, callCount)
}

func TestCompressAndCleanupRpm(t *testing.T) {
	tests := []struct {
		name          string
		mockRemoveAll func(string) error
		mockTarGZ     func(string, string) error
		mockChmod     func(string, os.FileMode) error
		targetFile    string
		expectError   bool
	}{
		{
			name:          "successful compression and cleanup",
			mockRemoveAll: func(path string) error { return nil },
			mockTarGZ:     func(src, dst string) error { return nil },
			mockChmod:     func(name string, perm os.FileMode) error { return nil },
			targetFile:    "test.tar.gz",
			expectError:   false,
		},
		{
			name:          "remove all fails",
			mockRemoveAll: func(path string) error { return fmt.Errorf("remove error") },
			mockTarGZ:     func(src, dst string) error { return nil },
			mockChmod:     func(name string, perm os.FileMode) error { return nil },
			targetFile:    "test.tar.gz",
			expectError:   true,
		},
		{
			name:          "tar gz fails",
			mockRemoveAll: func(path string) error { return nil },
			mockTarGZ:     func(src, dst string) error { return fmt.Errorf("tar error") },
			mockChmod:     func(name string, perm os.FileMode) error { return nil },
			targetFile:    "test.tar.gz",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)

			if tt.mockTarGZ != nil {
				patches.ApplyFunc(global.TarGZ, tt.mockTarGZ)
			}

			if tt.mockChmod != nil {
				patches.ApplyFunc(os.Chmod, tt.mockChmod)
			}

			err := compressAndCleanupRpm(tt.targetFile, "success message")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCleanRepodata(t *testing.T) {
	tests := []struct {
		name          string
		mockReadDir   func(string) ([]os.DirEntry, error)
		mockRemoveAll func(string) error
		mnt           string
		expectError   bool
	}{
		{
			name: "successful cleanup",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "subdir", isDir: true},
				}, nil
			},
			mockRemoveAll: func(path string) error { return nil },
			mnt:           "/mnt/path",
			expectError:   false,
		},
		{
			name: "read dir fails",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, fmt.Errorf("read error")
			},
			mnt:         "/mnt/path",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(path.Join, func(elem ...string) string {
				return strings.Join(elem, "/")
			})
			defer patches.Reset()

			err := cleanRepodata(tt.mnt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// MockDirEntry implements os.DirEntry for testing
type mockDirEntry struct {
	name  string
	isDir bool
}

func (m *mockDirEntry) Name() string {
	return m.name
}

func (m *mockDirEntry) IsDir() bool {
	return m.isDir
}

func (m *mockDirEntry) Type() os.FileMode {
	if m.isDir {
		return os.ModeDir
	}
	return 0
}

func (m *mockDirEntry) Info() (os.FileInfo, error) {
	return nil, nil
}

func TestAddListMinPartsConstant(t *testing.T) {
	// Test that the constant is defined correctly
	assert.Equal(t, testThreeValue, addListMinParts)
}

func TestRunBuildContainer(t *testing.T) {
	tests := []struct {
		name          string
		image         string
		mnt           string
		containerName string
		cmd           string
		mockRun       func(*container.Config, *container.HostConfig, *network.NetworkingConfig, *specs.Platform, string) error
		expectError   bool
	}{
		{
			name:          "successful run build container",
			image:         "test-image:latest",
			mnt:           "/mnt/path",
			containerName: "test-container",
			cmd:           "echo hello",
			mockRun: func(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, name string) error {
				// Verify that the config is set up correctly
				assert.Equal(t, "test-image:latest", config.Image)
				assert.Equal(t, "/opt/mnt", config.WorkingDir)
				assert.Equal(t, []string{"sh", "-c", "echo hello"}, []string(config.Cmd))

				assert.Equal(t, "/mnt/path", hostConfig.Mounts[0].Source)
				assert.Equal(t, "/opt/mnt", hostConfig.Mounts[0].Target)
				assert.Equal(t, container.RestartPolicyMode("no"), hostConfig.RestartPolicy.Name)
				return nil
			},
			expectError: false,
		},
		{
			name:          "run container fails",
			image:         "test-image:latest",
			mnt:           "/mnt/path",
			containerName: "test-container",
			cmd:           "echo hello",
			mockRun: func(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, name string) error {
				return fmt.Errorf("run failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock docker client
			mockDockerClient := &docker.Client{}

			// Save original global.Docker
			originalDocker := global.Docker
			defer func() {
				// Restore original global.Docker after test
				global.Docker = originalDocker
			}()

			// Set global.Docker to our mock
			global.Docker = mockDockerClient

			// Apply patches
			patches := gomonkey.ApplyMethod((*docker.Client)(nil), "Run",
				func(client *docker.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) error {
					return tt.mockRun(config, hostConfig, networkingConfig, platform, containerName)
				})
			defer patches.Reset()

			err := runBuildContainer(tt.image, tt.mnt, tt.containerName, tt.cmd)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWaitForContainerComplete(t *testing.T) {
	// This function has a loop that waits, so we need to mock the time.Sleep
	// and the ContainerInspect function to avoid infinite loops
	containerRunning := true
	callCount := 0

	// Create a mock docker client
	mockDockerClient := &docker.Client{}
	clientInstance := &client.Client{}

	// Save original global.Docker
	originalDocker := global.Docker
	defer func() {
		// Restore original global.Docker after test
		global.Docker = originalDocker
	}()

	// Set global.Docker to our mock
	global.Docker = mockDockerClient

	patches := gomonkey.ApplyFunc(time.Sleep, func(d time.Duration) {
		// After a few calls, make the container stop running to exit the loop
		callCount++
		if callCount >= 3 {
			containerRunning = false
		}
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*client.Client).ContainerInspect,
		func(_ *client.Client, ctx context.Context, containerID string) (types.ContainerJSON, error) {
			var containerInfo types.ContainerJSON
			if containerRunning {
				containerInfo.ContainerJSONBase = &types.ContainerJSONBase{
					State: &types.ContainerState{
						Running: true,
					},
				}
			} else {
				containerInfo.ContainerJSONBase = &types.ContainerJSONBase{
					State: &types.ContainerState{
						Running: false,
					},
				}
			}
			return containerInfo, nil
		})
	defer patches.Reset()

	patches = gomonkey.ApplyMethod((*docker.Client)(nil), "GetClient", func(_ *docker.Client) *client.Client {
		return clientInstance
	})
	defer patches.Reset()

	// Call the function - it should complete without hanging
	waitForContainerComplete("test-container")

	// The function should complete without errors
	assert.True(t, true)
}

func TestEnsureRpmBuildImage(t *testing.T) {
	tests := []struct {
		name                  string
		registry              string
		imageTag              string
		mockEnsureImageExists func(docker.ImageRef, utils.RetryOptions) error
		expectError           bool
		expectedImage         string
	}{
		{
			name:     "successful image pull",
			registry: "registry.example.com",
			imageTag: "centos:7-amd64-build",
			mockEnsureImageExists: func(ref docker.ImageRef, opts utils.RetryOptions) error {
				assert.Equal(t, "registry.example.com/centos:7-amd64-build", ref.Image)
				assert.Equal(t, testThreeValue, opts.MaxRetry)
				assert.Equal(t, time.Duration(testOneValue), opts.Delay)
				return nil
			},
			expectError:   false,
			expectedImage: "registry.example.com/centos:7-amd64-build",
		},
		{
			name:     "image pull fails",
			registry: "registry.example.com",
			imageTag: "centos:7-amd64-build",
			mockEnsureImageExists: func(ref docker.ImageRef, opts utils.RetryOptions) error {
				return fmt.Errorf("pull failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock docker client
			mockDockerClient := &docker.Client{}

			// Save original global.Docker
			originalDocker := global.Docker
			defer func() {
				// Restore original global.Docker after test
				global.Docker = originalDocker
			}()

			// Set global.Docker to our mock
			global.Docker = mockDockerClient

			patches := gomonkey.ApplyMethod((*docker.Client)(nil), "EnsureImageExists",
				func(client *docker.Client, ref docker.ImageRef, opts utils.RetryOptions) error {
					return tt.mockEnsureImageExists(ref, opts)
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			image, err := ensureRpmBuildImage(tt.registry, tt.imageTag)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, image)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedImage, image)
			}
		})
	}
}

func TestExecuteRpmBuildContainer(t *testing.T) {
	tests := []struct {
		name                  string
		image                 string
		mnt                   string
		containerName         string
		cmd                   string
		mockRunBuildContainer func(string, string, string, string) error
		mockContainerRemove   func(string) error
		expectError           bool
	}{
		{
			name:          "successful execution",
			image:         "test-image:latest",
			mnt:           "/mnt/path",
			containerName: "test-container",
			cmd:           "echo hello",
			mockRunBuildContainer: func(img, mnt, name, cmd string) error {
				return nil
			},
			mockContainerRemove: func(name string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "run build container fails",
			image:         "test-image:latest",
			mnt:           "/mnt/path",
			containerName: "test-container",
			cmd:           "echo hello",
			mockRunBuildContainer: func(img, mnt, name, cmd string) error {
				return fmt.Errorf("build failed")
			},
			mockContainerRemove: func(name string) error {
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock docker client
			mockDockerClient := &docker.Client{}

			// Save original global.Docker
			originalDocker := global.Docker
			defer func() {
				// Restore original global.Docker after test
				global.Docker = originalDocker
			}()

			// Set global.Docker to our mock
			global.Docker = mockDockerClient

			patches := gomonkey.ApplyFunc(runBuildContainer, tt.mockRunBuildContainer)
			defer patches.Reset()

			patches = gomonkey.ApplyMethod((*docker.Client)(nil), "ContainerRemove",
				func(client *docker.Client, name string) error {
					return tt.mockContainerRemove(name)
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(waitForContainerComplete, func(name string) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := executeRpmBuildContainer(tt.image, tt.mnt, tt.containerName, tt.cmd)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVerifyRpmBuildResult(t *testing.T) {
	tests := []struct {
		name          string
		mnt           string
		osInfo        string
		requiredFiles []string
		mockExists    func(string) bool
		expectError   bool
	}{
		{
			name:          "all required files exist",
			mnt:           "/mnt/path",
			osInfo:        "centos/7/amd64",
			requiredFiles: []string{"file1", "file2"},
			mockExists: func(filePath string) bool {
				return true // All files exist
			},
			expectError: false,
		},
		{
			name:          "some required files missing",
			mnt:           "/mnt/path",
			osInfo:        "centos/7/amd64",
			requiredFiles: []string{"existing_file", "missing_file"},
			mockExists: func(filePath string) bool {
				return !strings.Contains(filePath, "missing_file") // missing_file does not exist
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.Exists, tt.mockExists)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(path.Join, func(elem ...string) string {
				return strings.Join(elem, "/")
			})
			defer patches.Reset()

			err := verifyRpmBuildResult(tt.mnt, tt.osInfo, tt.requiredFiles...)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRpmBuildPackage(t *testing.T) {
	tests := []struct {
		name                         string
		source                       string
		registry                     string
		mockIsDir                    func(string) bool
		mockPrepareWorkspace         func() error
		mockCopyDir                  func(string, string) error
		mockRpmBuildAllArchitectures func(string) error
		mockCompressAndCleanupRpm    func(string, string) error
	}{
		{
			name:     "successful rpm build package",
			source:   "/source/dir",
			registry: "registry.example.com",
			mockIsDir: func(path string) bool {
				return true // Directory exists
			},
			mockPrepareWorkspace: func() error {
				return nil
			},
			mockCopyDir: func(src, dst string) error {
				return nil
			},
			mockRpmBuildAllArchitectures: func(registry string) error {
				return nil
			},
			mockCompressAndCleanupRpm: func(targetFile, successMsg string) error {
				return nil
			},
		},
		{
			name:     "source is not a directory",
			source:   "/non/existent/dir",
			registry: "registry.example.com",
			mockIsDir: func(path string) bool {
				return false // Directory does not exist
			},
		},
		{
			name:     "prepare workspace fails",
			source:   "/source/dir",
			registry: "registry.example.com",
			mockIsDir: func(path string) bool {
				return true // Directory exists
			},
			mockPrepareWorkspace: func() error {
				return fmt.Errorf("workspace preparation failed")
			},
		},
		{
			name:     "copy dir fails",
			source:   "/source/dir",
			registry: "registry.example.com",
			mockIsDir: func(path string) bool {
				return true // Directory exists
			},
			mockPrepareWorkspace: func() error {
				return nil
			},
			mockCopyDir: func(src, dst string) error {
				return fmt.Errorf("copying directory failed")
			},
		},
		{
			name:     "rpm build all architectures fails",
			source:   "/source/dir",
			registry: "registry.example.com",
			mockIsDir: func(path string) bool {
				return true // Directory exists
			},
			mockPrepareWorkspace: func() error {
				return nil
			},
			mockCopyDir: func(src, dst string) error {
				return nil
			},
			mockRpmBuildAllArchitectures: func(registry string) error {
				return fmt.Errorf("rpm build all architectures failed")
			},
		},
		{
			name:     "compress and cleanup rpm fails",
			source:   "/source/dir",
			registry: "registry.example.com",
			mockIsDir: func(path string) bool {
				return true // Directory exists
			},
			mockPrepareWorkspace: func() error {
				return nil
			},
			mockCopyDir: func(src, dst string) error {
				return nil
			},
			mockRpmBuildAllArchitectures: func(registry string) error {
				return nil
			},
			mockCompressAndCleanupRpm: func(targetFile, successMsg string) error {
				return fmt.Errorf("compress and cleanup rpm failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.IsDir, tt.mockIsDir)
			defer patches.Reset()

			if tt.mockPrepareWorkspace != nil {
				patches = gomonkey.ApplyFunc(prepareWorkspace, tt.mockPrepareWorkspace)
				defer patches.Reset()
			}

			if tt.mockCopyDir != nil {
				patches = gomonkey.ApplyFunc(utils.CopyDir, tt.mockCopyDir)
				defer patches.Reset()
			}

			if tt.mockRpmBuildAllArchitectures != nil {
				patches = gomonkey.ApplyFunc(rpmBuildAllArchitectures, tt.mockRpmBuildAllArchitectures)
				defer patches.Reset()
			}

			if tt.mockCompressAndCleanupRpm != nil {
				patches = gomonkey.ApplyFunc(compressAndCleanupRpm, tt.mockCompressAndCleanupRpm)
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			// Since rpmBuildPackage is void function, we just call it and make sure it doesn't panic
			rpmBuildPackage(tt.source, tt.registry)

			// The function should complete without panicking
			assert.True(t, true)
		})
	}
}

func TestRpmPackageAddOne(t *testing.T) {
	tests := []struct {
		name                      string
		source                    string
		registry                  string
		add                       string
		pack                      string
		mockIsFile                func(string) bool
		mockRemoveAll             func(string) error
		mockMkdirAll              func(string, os.FileMode) error
		mockUnTar                 func(string, string) error
		mockCopyDir               func(string, string) error
		mockRmpBuild              func(string, string, string) error
		mockCompressAndCleanupRpm func(string, string) error
	}{
		{
			name:     "successful rpm package add one",
			source:   "/source/file.tar.gz",
			registry: "registry.example.com",
			add:      "centos/7/amd64",
			pack:     "/package/dir",
			mockIsFile: func(path string) bool {
				return true // File exists
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockMkdirAll: func(path string, perm os.FileMode) error {
				return nil
			},
			mockUnTar: func(src, dst string) error {
				return nil
			},
			mockCopyDir: func(src, dst string) error {
				return nil
			},
			mockRmpBuild: func(registry, add, absPath string) error {
				return nil
			},
			mockCompressAndCleanupRpm: func(targetFile, successMsg string) error {
				return nil
			},
		},
		{
			name:     "source is not a file",
			source:   "/non/existent/file.tar.gz",
			registry: "registry.example.com",
			add:      "centos/7/amd64",
			pack:     "/package/dir",
			mockIsFile: func(path string) bool {
				return false // File does not exist
			},
		},
		{
			name:     "remove all fails",
			source:   "/source/file.tar.gz",
			registry: "registry.example.com",
			add:      "centos/7/amd64",
			pack:     "/package/dir",
			mockIsFile: func(path string) bool {
				return true // File exists
			},
			mockRemoveAll: func(path string) error {
				return fmt.Errorf("remove all failed")
			},
		},
		{
			name:     "mkdir all fails",
			source:   "/source/file.tar.gz",
			registry: "registry.example.com",
			add:      "centos/7/amd64",
			pack:     "/package/dir",
			mockIsFile: func(path string) bool {
				return true // File exists
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockMkdirAll: func(path string, perm os.FileMode) error {
				return fmt.Errorf("mkdir all failed")
			},
		},
		{
			name:     "untar fails",
			source:   "/source/file.tar.gz",
			registry: "registry.example.com",
			add:      "centos/7/amd64",
			pack:     "/package/dir",
			mockIsFile: func(path string) bool {
				return true // File exists
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockMkdirAll: func(path string, perm os.FileMode) error {
				return nil
			},
			mockUnTar: func(src, dst string) error {
				return fmt.Errorf("untar failed")
			},
		},
		{
			name:     "copy dir fails",
			source:   "/source/file.tar.gz",
			registry: "registry.example.com",
			add:      "centos/7/amd64",
			pack:     "/package/dir",
			mockIsFile: func(path string) bool {
				return true // File exists
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockMkdirAll: func(path string, perm os.FileMode) error {
				return nil
			},
			mockUnTar: func(src, dst string) error {
				return nil
			},
			mockCopyDir: func(src, dst string) error {
				return fmt.Errorf("copy dir failed")
			},
		},
		{
			name:     "rmp build fails",
			source:   "/source/file.tar.gz",
			registry: "registry.example.com",
			add:      "centos/7/amd64",
			pack:     "/package/dir",
			mockIsFile: func(path string) bool {
				return true // File exists
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockMkdirAll: func(path string, perm os.FileMode) error {
				return nil
			},
			mockUnTar: func(src, dst string) error {
				return nil
			},
			mockCopyDir: func(src, dst string) error {
				return nil
			},
			mockRmpBuild: func(registry, add, absPath string) error {
				return fmt.Errorf("rmp build failed")
			},
		},
		{
			name:     "compress and cleanup rpm fails",
			source:   "/source/file.tar.gz",
			registry: "registry.example.com",
			add:      "centos/7/amd64",
			pack:     "/package/dir",
			mockIsFile: func(path string) bool {
				return true // File exists
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockMkdirAll: func(path string, perm os.FileMode) error {
				return nil
			},
			mockUnTar: func(src, dst string) error {
				return nil
			},
			mockCopyDir: func(src, dst string) error {
				return nil
			},
			mockRmpBuild: func(registry, add, absPath string) error {
				return nil
			},
			mockCompressAndCleanupRpm: func(targetFile, successMsg string) error {
				return fmt.Errorf("compress and cleanup rpm failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.IsFile, tt.mockIsFile)
			defer patches.Reset()

			if tt.mockRemoveAll != nil {
				patches = gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
				defer patches.Reset()
			}

			if tt.mockMkdirAll != nil {
				patches = gomonkey.ApplyFunc(os.MkdirAll, tt.mockMkdirAll)
				defer patches.Reset()
			}

			if tt.mockUnTar != nil {
				patches = gomonkey.ApplyFunc(utils.UnTar, tt.mockUnTar)
				defer patches.Reset()
			}

			if tt.mockCopyDir != nil {
				patches = gomonkey.ApplyFunc(utils.CopyDir, tt.mockCopyDir)
				defer patches.Reset()
			}

			if tt.mockRmpBuild != nil {
				patches = gomonkey.ApplyFunc(rmpBuild, tt.mockRmpBuild)
				defer patches.Reset()
			}

			if tt.mockCompressAndCleanupRpm != nil {
				patches = gomonkey.ApplyFunc(compressAndCleanupRpm, tt.mockCompressAndCleanupRpm)
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			// Since rpmPackageAddOne is void function, we just call it and make sure it doesn't panic
			rpmPackageAddOne(tt.source, tt.registry, tt.add, tt.pack)

			// The function should complete without panicking
			assert.True(t, true)
		})
	}
}

func TestRpmCentos8Build(t *testing.T) {
	tests := []struct {
		name                         string
		registry                     string
		mnt                          string
		mockDirectoryIsEmpty         func(string) bool
		mockEnsureRpmBuildImage      func(string, string) (string, error)
		mockCleanCentos8Modules      func(string) error
		mockExecuteRpmBuildContainer func(string, string, string, string) error
		mockVerifyRpmBuildResult     func(string, string, ...string) error
		expectError                  bool
	}{
		{
			name:     "directory is empty",
			registry: "registry.example.com",
			mnt:      "/empty/dir",
			mockDirectoryIsEmpty: func(path string) bool {
				return true
			},
			expectError: false,
		},
		{
			name:     "successful build",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				assert.Equal(t, "registry.example.com", registry)
				assert.Equal(t, "centos:8-amd64-build", imageTag)
				return "registry.example.com/centos:8-amd64-build", nil
			},
			mockCleanCentos8Modules: func(mnt string) error {
				assert.Equal(t, "/mnt/path", mnt)
				return nil
			},
			mockExecuteRpmBuildContainer: func(image, mnt, containerName, cmd string) error {
				assert.Equal(t, "registry.example.com/centos:8-amd64-build", image)
				assert.Equal(t, "/mnt/path", mnt)
				assert.Equal(t, "build-centos8-rpm", containerName)
				assert.Contains(t, cmd, "createrepo")
				return nil
			},
			mockVerifyRpmBuildResult: func(mnt, osInfo string, files ...string) error {
				assert.Equal(t, "/mnt/path", mnt)
				assert.Equal(t, "centos/8/amd64", osInfo)
				assert.Contains(t, files, "modules.yaml")
				assert.Contains(t, files, "repodata")
				return nil
			},
			expectError: false,
		},
		{
			name:     "ensureRpmBuildImage fails",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				return "", fmt.Errorf("image pull failed")
			},
			expectError: true,
		},
		{
			name:     "cleanCentos8Modules fails",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				return "registry.example.com/centos:8-amd64-build", nil
			},
			mockCleanCentos8Modules: func(mnt string) error {
				return fmt.Errorf("clean modules failed")
			},
			expectError: true,
		},
		{
			name:     "executeRpmBuildContainer fails",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				return "registry.example.com/centos:8-amd64-build", nil
			},
			mockCleanCentos8Modules: func(mnt string) error {
				return nil
			},
			mockExecuteRpmBuildContainer: func(image, mnt, containerName, cmd string) error {
				return fmt.Errorf("container execution failed")
			},
			expectError: true,
		},
		{
			name:     "verifyRpmBuildResult fails",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				return "registry.example.com/centos:8-amd64-build", nil
			},
			mockCleanCentos8Modules: func(mnt string) error {
				return nil
			},
			mockExecuteRpmBuildContainer: func(image, mnt, containerName, cmd string) error {
				return nil
			},
			mockVerifyRpmBuildResult: func(mnt, osInfo string, files ...string) error {
				return fmt.Errorf("verification failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(utils.DirectoryIsEmpty, tt.mockDirectoryIsEmpty)
			defer patches.Reset()

			if tt.mockEnsureRpmBuildImage != nil {
				patches = gomonkey.ApplyFunc(ensureRpmBuildImage, tt.mockEnsureRpmBuildImage)
				defer patches.Reset()
			}

			if tt.mockCleanCentos8Modules != nil {
				patches = gomonkey.ApplyFunc(cleanCentos8Modules, tt.mockCleanCentos8Modules)
				defer patches.Reset()
			}

			if tt.mockExecuteRpmBuildContainer != nil {
				patches = gomonkey.ApplyFunc(executeRpmBuildContainer, tt.mockExecuteRpmBuildContainer)
				defer patches.Reset()
			}

			if tt.mockVerifyRpmBuildResult != nil {
				patches = gomonkey.ApplyFunc(verifyRpmBuildResult, tt.mockVerifyRpmBuildResult)
				defer patches.Reset()
			}

			err := rpmCentos8Build(tt.registry, tt.mnt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCleanCentos8Modules(t *testing.T) {
	tests := []struct {
		name          string
		mnt           string
		mockReadDir   func(string) ([]os.DirEntry, error)
		mockRemoveAll func(string) error
		expectError   bool
	}{
		{
			name: "successful cleaning",
			mnt:  "/mnt/path",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				// Mock directory entries
				return []os.DirEntry{
					&mockDirEntry{name: "subdir", isDir: true},
				}, nil
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "read dir fails",
			mnt:  "/mnt/path",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, fmt.Errorf("read error")
			},
			expectError: true,
		},
		{
			name: "remove all fails",
			mnt:  "/mnt/path",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "subdir", isDir: true},
				}, nil
			},
			mockRemoveAll: func(path string) error {
				return fmt.Errorf("remove error")
			},
			expectError: false, // Should not return error even if RemoveAll fails
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(path.Join, func(elem ...string) string {
				return strings.Join(elem, "/")
			})
			defer patches.Reset()

			err := cleanCentos8Modules(tt.mnt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExecuteGenericRpmBuild(t *testing.T) {
	tests := []struct {
		name                         string
		config                       rpmBuildConfig
		mockDirectoryIsEmpty         func(string) bool
		mockEnsureRpmBuildImage      func(string, string) (string, error)
		mockCleanRepodata            func(string) error
		mockExecuteRpmBuildContainer func(string, string, string, string) error
		mockVerifyRpmBuildResult     func(string, string, ...string) error
		expectError                  bool
	}{
		{
			name: "directory is empty",
			config: rpmBuildConfig{
				registry:      "registry.example.com",
				mnt:           "/empty/dir",
				image:         "test-image",
				containerName: "test-container",
				cmd:           "test-cmd",
				osInfo:        "test-os",
				checkFile:     "test-file",
			},
			mockDirectoryIsEmpty: func(path string) bool {
				return true
			},
			expectError: false,
		},
		{
			name: "successful build",
			config: rpmBuildConfig{
				registry:      "registry.example.com",
				mnt:           "/mnt/path",
				image:         "test-image",
				containerName: "test-container",
				cmd:           "test-cmd",
				osInfo:        "test-os",
				checkFile:     "test-file",
			},
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				assert.Equal(t, "registry.example.com", registry)
				assert.Equal(t, "test-image", imageTag)
				return "registry.example.com/test-image", nil
			},
			mockCleanRepodata: func(mnt string) error {
				assert.Equal(t, "/mnt/path", mnt)
				return nil
			},
			mockExecuteRpmBuildContainer: func(image, mnt, containerName, cmd string) error {
				assert.Equal(t, "registry.example.com/test-image", image)
				assert.Equal(t, "/mnt/path", mnt)
				assert.Equal(t, "test-container", containerName)
				assert.Equal(t, "test-cmd", cmd)
				return nil
			},
			mockVerifyRpmBuildResult: func(mnt, osInfo string, files ...string) error {
				assert.Equal(t, "/mnt/path", mnt)
				assert.Equal(t, "test-os", osInfo)
				assert.Contains(t, files, "test-file")
				return nil
			},
			expectError: false,
		},
		{
			name: "ensureRpmBuildImage fails",
			config: rpmBuildConfig{
				registry:      "registry.example.com",
				mnt:           "/mnt/path",
				image:         "test-image",
				containerName: "test-container",
				cmd:           "test-cmd",
				osInfo:        "test-os",
				checkFile:     "test-file",
			},
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				return "", fmt.Errorf("image pull failed")
			},
			expectError: true,
		},
		{
			name: "cleanRepodata fails",
			config: rpmBuildConfig{
				registry:      "registry.example.com",
				mnt:           "/mnt/path",
				image:         "test-image",
				containerName: "test-container",
				cmd:           "test-cmd",
				osInfo:        "test-os",
				checkFile:     "test-file",
			},
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				return "registry.example.com/test-image", nil
			},
			mockCleanRepodata: func(mnt string) error {
				return fmt.Errorf("clean repodata failed")
			},
			expectError: true,
		},
		{
			name: "executeRpmBuildContainer fails",
			config: rpmBuildConfig{
				registry:      "registry.example.com",
				mnt:           "/mnt/path",
				image:         "test-image",
				containerName: "test-container",
				cmd:           "test-cmd",
				osInfo:        "test-os",
				checkFile:     "test-file",
			},
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				return "registry.example.com/test-image", nil
			},
			mockCleanRepodata: func(mnt string) error {
				return nil
			},
			mockExecuteRpmBuildContainer: func(image, mnt, containerName, cmd string) error {
				return fmt.Errorf("container execution failed")
			},
			expectError: true,
		},
		{
			name: "verifyRpmBuildResult fails",
			config: rpmBuildConfig{
				registry:      "registry.example.com",
				mnt:           "/mnt/path",
				image:         "test-image",
				containerName: "test-container",
				cmd:           "test-cmd",
				osInfo:        "test-os",
				checkFile:     "test-file",
			},
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureRpmBuildImage: func(registry, imageTag string) (string, error) {
				return "registry.example.com/test-image", nil
			},
			mockCleanRepodata: func(mnt string) error {
				return nil
			},
			mockExecuteRpmBuildContainer: func(image, mnt, containerName, cmd string) error {
				return nil
			},
			mockVerifyRpmBuildResult: func(mnt, osInfo string, files ...string) error {
				return fmt.Errorf("verification failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(utils.DirectoryIsEmpty, tt.mockDirectoryIsEmpty)
			defer patches.Reset()

			if tt.mockEnsureRpmBuildImage != nil {
				patches = gomonkey.ApplyFunc(ensureRpmBuildImage, tt.mockEnsureRpmBuildImage)
				defer patches.Reset()
			}

			if tt.mockCleanRepodata != nil {
				patches = gomonkey.ApplyFunc(cleanRepodata, tt.mockCleanRepodata)
				defer patches.Reset()
			}

			if tt.mockExecuteRpmBuildContainer != nil {
				patches = gomonkey.ApplyFunc(executeRpmBuildContainer, tt.mockExecuteRpmBuildContainer)
				defer patches.Reset()
			}

			if tt.mockVerifyRpmBuildResult != nil {
				patches = gomonkey.ApplyFunc(verifyRpmBuildResult, tt.mockVerifyRpmBuildResult)
				defer patches.Reset()
			}

			err := executeGenericRpmBuild(tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRpmUbuntu22Build(t *testing.T) {
	tests := []struct {
		name                         string
		registry                     string
		mnt                          string
		mockDirectoryIsEmpty         func(string) bool
		mockEnsureImageExists        func(docker.ImageRef, utils.RetryOptions) error
		mockRemoveAll                func(string) error
		mockRunBuildContainer        func(string, string, string, string) error
		mockWaitForContainerComplete func(string)
		mockExists                   func(string) bool
		expectError                  bool
	}{
		{
			name:     "directory is empty",
			registry: "registry.example.com",
			mnt:      "/empty/dir",
			mockDirectoryIsEmpty: func(path string) bool {
				return true
			},
			expectError: false,
		},
		{
			name:     "successful build",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureImageExists: func(ref docker.ImageRef, opts utils.RetryOptions) error {
				assert.Equal(t, "registry.example.com/ubuntu:22-amd64-build", ref.Image)
				assert.Equal(t, testThreeValue, opts.MaxRetry)
				assert.Equal(t, time.Duration(testOneValue), opts.Delay)
				return nil
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockRunBuildContainer: func(image, mnt, containerName, cmd string) error {
				assert.Equal(t, "registry.example.com/ubuntu:22-amd64-build", image)
				assert.Equal(t, "/mnt/path", mnt)
				assert.Equal(t, "build-ubuntu22-rpm", containerName)
				assert.Contains(t, cmd, "dpkg-scanpackages")
				return nil
			},
			mockWaitForContainerComplete: func(name string) {},
			mockExists: func(path string) bool {
				return true
			},
			expectError: false,
		},
		{
			name:     "ensureImageExists fails",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureImageExists: func(ref docker.ImageRef, opts utils.RetryOptions) error {
				return fmt.Errorf("image pull failed")
			},
			expectError: true,
		},
		{
			name:     "runBuildContainer fails",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureImageExists: func(ref docker.ImageRef, opts utils.RetryOptions) error {
				return nil
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockRunBuildContainer: func(image, mnt, containerName, cmd string) error {
				return fmt.Errorf("container run failed")
			},
			expectError: true,
		},
		{
			name:     "packages.gz not found",
			registry: "registry.example.com",
			mnt:      "/mnt/path",
			mockDirectoryIsEmpty: func(path string) bool {
				return false
			},
			mockEnsureImageExists: func(ref docker.ImageRef, opts utils.RetryOptions) error {
				return nil
			},
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockRunBuildContainer: func(image, mnt, containerName, cmd string) error {
				return nil
			},
			mockWaitForContainerComplete: func(name string) {},
			mockExists: func(path string) bool {
				return false // Packages.gz not found
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock docker client
			mockDockerClient := &docker.Client{}

			// Save original global.Docker
			originalDocker := global.Docker
			defer func() {
				// Restore original global.Docker after test
				global.Docker = originalDocker
			}()

			// Set global.Docker to our mock
			global.Docker = mockDockerClient

			// Apply patches
			patches := gomonkey.ApplyFunc(utils.DirectoryIsEmpty, tt.mockDirectoryIsEmpty)
			defer patches.Reset()

			if tt.mockEnsureImageExists != nil {
				patches = gomonkey.ApplyMethod((*docker.Client)(nil), "EnsureImageExists",
					func(client *docker.Client, ref docker.ImageRef, opts utils.RetryOptions) error {
						return tt.mockEnsureImageExists(ref, opts)
					})
				defer patches.Reset()
			}

			patches = gomonkey.ApplyMethod((*docker.Client)(nil), "ContainerRemove",
				func(client *docker.Client, name string) error {
					return nil
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
			defer patches.Reset()

			if tt.mockRunBuildContainer != nil {
				patches = gomonkey.ApplyFunc(runBuildContainer, tt.mockRunBuildContainer)
				defer patches.Reset()
			}

			if tt.mockWaitForContainerComplete != nil {
				patches = gomonkey.ApplyFunc(waitForContainerComplete, tt.mockWaitForContainerComplete)
				defer patches.Reset()
			}

			if tt.mockExists != nil {
				patches = gomonkey.ApplyFunc(utils.Exists, tt.mockExists)
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(path.Join, func(elem ...string) string {
				return strings.Join(elem, "/")
			})
			defer patches.Reset()

			err := rpmUbuntu22Build(tt.registry, tt.mnt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
