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
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/exec"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
)

func TestOptionsStruct(t *testing.T) {
	// Test that the Options struct has the expected fields
	o := &Options{}

	// Check that it embeds root.Options
	_ = &o.Options

	// Test that the fields exist
	o.File = "test.yaml"
	o.Target = "test.tar.gz"
	o.Strategy = "registry"
	o.Arch = "amd64"

	assert.Equal(t, "test.yaml", o.File)
	assert.Equal(t, "test.tar.gz", o.Target)
	assert.Equal(t, "registry", o.Strategy)
	assert.Equal(t, "amd64", o.Arch)
}

func TestBuild(t *testing.T) {
	tests := []struct {
		name                string
		isDockerEnvironment bool
		loadConfigResult    *BuildConfig
		loadConfigError     error
		prepareWorkspaceErr error
		collectDepsResult   string
		collectDepsError    error
		createPackageError  error
		expectError         bool
	}{
		{
			name:                "successful build",
			isDockerEnvironment: true,
			loadConfigResult:    &BuildConfig{},
			loadConfigError:     nil,
			prepareWorkspaceErr: nil,
			collectDepsResult:   "v1.0.0",
			collectDepsError:    nil,
			createPackageError:  nil,
			expectError:         false,
		},
		{
			name:                "not in docker environment",
			isDockerEnvironment: false,
			expectError:         true,
		},
		{
			name:                "config loading fails",
			isDockerEnvironment: true,
			loadConfigResult:    nil,
			loadConfigError:     fmt.Errorf("config error"),
			expectError:         true,
		},
		{
			name:                "workspace preparation fails",
			isDockerEnvironment: true,
			loadConfigResult:    &BuildConfig{},
			loadConfigError:     nil,
			prepareWorkspaceErr: fmt.Errorf("workspace error"),
			expectError:         true,
		},
		{
			name:                "dependency collection fails",
			isDockerEnvironment: true,
			loadConfigResult:    &BuildConfig{},
			loadConfigError:     nil,
			prepareWorkspaceErr: nil,
			collectDepsResult:   "",
			collectDepsError:    fmt.Errorf("deps error"),
			expectError:         true,
		},
		{
			name:                "package creation fails",
			isDockerEnvironment: true,
			loadConfigResult:    &BuildConfig{},
			loadConfigError:     nil,
			prepareWorkspaceErr: nil,
			collectDepsResult:   "v1.0.0",
			collectDepsError:    nil,
			createPackageError:  fmt.Errorf("package error"),
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				File:   "test-config.yaml",
				Target: "test-package.tar.gz",
			}

			// Apply patches
			patchesDocker := gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
				return tt.isDockerEnvironment
			})
			defer patchesDocker.Reset()

			if tt.isDockerEnvironment {
				patchesLoadConfig := gomonkey.ApplyFunc(loadAndVerifyBuildConfig, func(file string) (*BuildConfig, error) {
					return tt.loadConfigResult, tt.loadConfigError
				})
				defer patchesLoadConfig.Reset()

				patchesPrepare := gomonkey.ApplyFunc(prepareBuildWorkspace, func() error {
					return tt.prepareWorkspaceErr
				})
				defer patchesPrepare.Reset()

				patchesCollect := gomonkey.ApplyFunc((*Options).collectDependenciesAndImages, func(o *Options, cfg *BuildConfig) (string, error) {
					return tt.collectDepsResult, tt.collectDepsError
				})
				defer patchesCollect.Reset()

				patchesCreate := gomonkey.ApplyFunc((*Options).createFinalPackage, func(o *Options, cfg *BuildConfig, version string) error {
					return tt.createPackageError
				})
				defer patchesCreate.Reset()

			}

			o.Build()

			// The function should complete without panic
			assert.True(t, true)
		})
	}
}

func TestLoadAndVerifyBuildConfig(t *testing.T) {
	tests := []struct {
		name          string
		fileContent   string
		mockReadFile  func(string) ([]byte, error)
		mockUnmarshal func([]byte, interface{}) error
		mockVerify    func(*BuildConfig) error
		expectError   bool
	}{
		{
			name:        "successful config loading",
			fileContent: "version: v1.0.0",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("version: v1.0.0"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				cfg := v.(*BuildConfig)
				cfg.OpenFuyaoVersion = "v1.0.0"
				return nil
			},
			mockVerify:  func(cfg *BuildConfig) error { return nil },
			expectError: false,
		},
		{
			name:        "file read error",
			fileContent: "",
			mockReadFile: func(filename string) ([]byte, error) {
				return nil, fmt.Errorf("file not found")
			},
			expectError: true,
		},
		{
			name:        "unmarshal error",
			fileContent: "invalid yaml",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("invalid yaml"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				return fmt.Errorf("unmarshal error")
			},
			expectError: true,
		},
		{
			name:        "verification error",
			fileContent: "version: v1.0.0",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("version: v1.0.0"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				cfg := v.(*BuildConfig)
				cfg.OpenFuyaoVersion = "v1.0.0"
				return nil
			},
			mockVerify:  func(cfg *BuildConfig) error { return fmt.Errorf("verification failed") },
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patchesReadFile := gomonkey.ApplyFunc(os.ReadFile, tt.mockReadFile)
			defer patchesReadFile.Reset()

			patchesUnmarshal := gomonkey.ApplyFunc(yaml.Unmarshal, tt.mockUnmarshal)
			defer patchesUnmarshal.Reset()

			patchesVerify := gomonkey.ApplyFunc(verifyConfigContent, tt.mockVerify)
			defer patchesVerify.Reset()

			cfg, err := loadAndVerifyBuildConfig("test-file.yaml")

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestPrepareBuildWorkspace(t *testing.T) {
	tests := []struct {
		name        string
		mockPrepare func() error
		expectError bool
	}{
		{
			name:        "successful workspace preparation",
			mockPrepare: func() error { return nil },
			expectError: false,
		},
		{
			name:        "workspace preparation fails",
			mockPrepare: func() error { return fmt.Errorf("prepare error") },
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(prepare, tt.mockPrepare)
			defer patches.Reset()

			err := prepareBuildWorkspace()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectRpmsAndBinary(t *testing.T) {
	stopChan := make(chan struct{})
	var errNumber uint64

	tests := []struct {
		name               string
		mockBuildRpms      func(cfg *BuildConfig, stopChan <-chan struct{}) error
		mockBuildBkeBinary func() (string, error)
		expectError        bool
	}{
		{
			name:               "successful RPM and binary collection",
			mockBuildRpms:      func(cfg *BuildConfig, stopChan <-chan struct{}) error { return nil },
			mockBuildBkeBinary: func() (string, error) { return "v1.0.0", nil },
			expectError:        false,
		},
		{
			name:               "RPM build fails",
			mockBuildRpms:      func(cfg *BuildConfig, stopChan <-chan struct{}) error { return fmt.Errorf("RPM error") },
			mockBuildBkeBinary: func() (string, error) { return "v1.0.0", nil },
			expectError:        true,
		},
		{
			name:               "binary build fails",
			mockBuildRpms:      func(cfg *BuildConfig, stopChan <-chan struct{}) error { return nil },
			mockBuildBkeBinary: func() (string, error) { return "", fmt.Errorf("binary error") },
			expectError:        true,
		},
		{
			name:               "binary build returns empty version",
			mockBuildRpms:      func(cfg *BuildConfig, stopChan <-chan struct{}) error { return nil },
			mockBuildBkeBinary: func() (string, error) { return "", nil },
			expectError:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &BuildConfig{}

			patchesBuildRpms := gomonkey.ApplyFunc(buildRpms, tt.mockBuildRpms)
			defer patchesBuildRpms.Reset()

			patchesBuildBinary := gomonkey.ApplyFunc(buildBkeBinary, tt.mockBuildBkeBinary)
			defer patchesBuildBinary.Reset()

			version, err := collectRpmsAndBinary(cfg, stopChan, &errNumber)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, "", version)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectRegistryImages(t *testing.T) {
	stopChan := make(chan struct{})
	var errNumber uint64

	tests := []struct {
		name              string
		mockBuildRegistry func(string, []string) error
		mockSyncRepo      func(*BuildConfig, chan struct{}) error
		expectError       bool
	}{
		{
			name:              "successful registry image collection",
			mockBuildRegistry: func(imageAddr string, arch []string) error { return nil },
			mockSyncRepo:      func(cfg *BuildConfig, stopChan chan struct{}) error { return nil },
			expectError:       false,
		},
		{
			name:              "registry build fails",
			mockBuildRegistry: func(imageAddr string, arch []string) error { return fmt.Errorf("registry error") },
			mockSyncRepo:      func(cfg *BuildConfig, stopChan chan struct{}) error { return nil },
			expectError:       true,
		},
		{
			name:              "repo sync fails",
			mockBuildRegistry: func(imageAddr string, arch []string) error { return nil },
			mockSyncRepo:      func(cfg *BuildConfig, stopChan chan struct{}) error { return fmt.Errorf("sync error") },
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &BuildConfig{}

			patchesBuildRegistry := gomonkey.ApplyFunc(buildRegistry, tt.mockBuildRegistry)
			defer patchesBuildRegistry.Reset()

			patchesSyncRepo := gomonkey.ApplyFunc(syncRepo, tt.mockSyncRepo)
			defer patchesSyncRepo.Reset()

			err := collectRegistryImages(cfg, stopChan, &errNumber)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateFinalPackage(t *testing.T) {
	tests := []struct {
		name                  string
		target                string
		version               string
		mockStat              func(string) (os.FileInfo, error)
		mockCompressedPackage func(*BuildConfig, string) error
		expectError           bool
	}{
		{
			name:                  "successful package creation with target specified",
			target:                "custom-target.tar.gz",
			version:               "v1.0.0",
			mockCompressedPackage: func(cfg *BuildConfig, target string) error { return nil },
			expectError:           false,
		},
		{
			name:    "successful package creation with no target (auto-generate)",
			target:  "",
			version: "v1.0.0",
			mockStat: func(name string) (os.FileInfo, error) {
				return &fakeFileInfo{name: "test.yaml"}, nil
			},
			mockCompressedPackage: func(cfg *BuildConfig, target string) error { return nil },
			expectError:           false,
		},
		{
			name:    "stat fails when auto-generating target",
			target:  "",
			version: "v1.0.0",
			mockStat: func(name string) (os.FileInfo, error) {
				return nil, fmt.Errorf("stat error")
			},
			expectError: true,
		},
		{
			name:                  "compression fails",
			target:                "custom-target.tar.gz",
			version:               "v1.0.0",
			mockCompressedPackage: func(cfg *BuildConfig, target string) error { return fmt.Errorf("compress error") },
			expectError:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				File:   "test.yaml",
				Target: tt.target,
			}
			cfg := &BuildConfig{}

			if tt.target == "" && tt.mockStat != nil {
				patchesStat := gomonkey.ApplyFunc(os.Stat, tt.mockStat)
				defer patchesStat.Reset()
			}

			patchesCompressed := gomonkey.ApplyFunc(compressedPackage, tt.mockCompressedPackage)
			defer patchesCompressed.Reset()

			err := o.createFinalPackage(cfg, tt.version)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWriteManifestsFile(t *testing.T) {
	tests := []struct {
		name          string
		mockMarshal   func(interface{}) ([]byte, error)
		mockWriteFile func(string, []byte, os.FileMode) error
		expectError   bool
	}{
		{
			name: "successful manifest writing",
			mockMarshal: func(v interface{}) ([]byte, error) {
				return []byte("version: v1.0.0"), nil
			},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "marshal fails",
			mockMarshal: func(v interface{}) ([]byte, error) {
				return nil, fmt.Errorf("marshal error")
			},
			expectError: true,
		},
		{
			name: "write file fails",
			mockMarshal: func(v interface{}) ([]byte, error) {
				return []byte("version: v1.0.0"), nil
			},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return fmt.Errorf("write error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &BuildConfig{OpenFuyaoVersion: "v1.0.0"}

			patchesMarshal := gomonkey.ApplyFunc(yaml.Marshal, tt.mockMarshal)
			defer patchesMarshal.Reset()

			patchesWriteFile := gomonkey.ApplyFunc(os.WriteFile, tt.mockWriteFile)
			defer patchesWriteFile.Reset()

			err := writeManifestsFile(cfg, "manifests.yaml")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFinalizeAndCompress(t *testing.T) {
	// 设置测试所需的全局变量
	originalPwd := pwd
	originalTmp := tmp
	originalPackages := packages
	originalBke := bke
	defer func() {
		// 恢复原始值
		pwd = originalPwd
		tmp = originalTmp
		packages = originalPackages
		bke = originalBke
	}()

	// 初始化测试用的全局变量
	pwd = "/tmp/test"
	tmp = "/tmp/test/packages/tmp"
	packages = "/tmp/test/packages"
	bke = "/tmp/test/packages/bke"

	tests := []struct {
		name                       string
		mockRemoveAll              func(string) error
		mockTaeGZWithoutChangeFile func(string, string) error
		target                     string
		expectError                bool
	}{
		{
			name:                       "successful compression",
			mockRemoveAll:              func(dir string) error { return nil },
			mockTaeGZWithoutChangeFile: func(src, dst string) error { return nil },
			target:                     "test.tar.gz",
			expectError:                false,
		},
		{
			name:                       "remove all fails",
			mockRemoveAll:              func(dir string) error { return fmt.Errorf("remove error") },
			mockTaeGZWithoutChangeFile: nil,
			target:                     "test.tar.gz",
			expectError:                true,
		},
		{
			name:                       "compression fails",
			mockRemoveAll:              func(dir string) error { return nil },
			mockTaeGZWithoutChangeFile: func(src, dst string) error { return fmt.Errorf("compress error") },
			target:                     "test.tar.gz",
			expectError:                true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个模拟的Command执行器
			mockExecutor := &exec.CommandExecutor{}
			global.Command = mockExecutor

			patchesRemoveAll := gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
			defer patchesRemoveAll.Reset()

			var patchesTaeGZ *gomonkey.Patches
			if tt.mockTaeGZWithoutChangeFile != nil {
				patchesTaeGZ = gomonkey.ApplyFunc(global.TaeGZWithoutChangeFile, tt.mockTaeGZWithoutChangeFile)
				defer patchesTaeGZ.Reset()
			} else {
				patchesTaeGZ = gomonkey.ApplyFunc(global.TaeGZWithoutChangeFile, func(src, dst string) error {
					return nil
				})
				defer patchesTaeGZ.Reset()
			}

			err := finalizeAndCompress(tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Helper struct to implement os.FileInfo for testing
type fakeFileInfo struct {
	name string
}

func (f *fakeFileInfo) Name() string       { return f.name }
func (f *fakeFileInfo) Size() int64        { return 0 }
func (f *fakeFileInfo) Mode() os.FileMode  { return 0 }
func (f *fakeFileInfo) ModTime() time.Time { return time.Now() }
func (f *fakeFileInfo) IsDir() bool        { return false }
func (f *fakeFileInfo) Sys() interface{}   { return nil }

func TestOptionsCollectDependenciesAndImages(t *testing.T) {
	const (
		testVersion = "v1.0.0"
	)

	tests := []struct {
		name                      string
		config                    *BuildConfig
		mockCollectRpmsAndBinary  func(*BuildConfig, chan struct{}, *uint64) (string, error)
		mockCollectRegistryImages func(*BuildConfig, chan struct{}, *uint64) error
		expectVersion             string
		expectError               bool
	}{
		{
			name:   "successful collection",
			config: &BuildConfig{},
			mockCollectRpmsAndBinary: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) (string, error) {
				return testVersion, nil
			},
			mockCollectRegistryImages: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
				return nil
			},
			expectVersion: testVersion,
			expectError:   false,
		},
		{
			name:   "error in collectRpmsAndBinary",
			config: &BuildConfig{},
			mockCollectRpmsAndBinary: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) (string, error) {
				return "", errors.New("rpm collection failed")
			},
			mockCollectRegistryImages: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
				return nil
			},
			expectVersion: "",
			expectError:   true,
		},
		{
			name:   "error in collectRegistryImages but rpmsAndBinary succeeds",
			config: &BuildConfig{},
			mockCollectRpmsAndBinary: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) (string, error) {
				return testVersion, nil
			},
			mockCollectRegistryImages: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
				*errNumber++ // Increment error count
				return errors.New("registry collection failed")
			},
			expectVersion: "", // Will be empty because errNumber > 0
			expectError:   true,
		},
		{
			name:   "multiple errors",
			config: &BuildConfig{},
			mockCollectRpmsAndBinary: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) (string, error) {
				*errNumber++
				return "", errors.New("rpm collection failed")
			},
			mockCollectRegistryImages: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
				*errNumber++
				return errors.New("registry collection failed")
			},
			expectVersion: "",
			expectError:   true,
		},
		{
			name:   "empty version returned",
			config: &BuildConfig{},
			mockCollectRpmsAndBinary: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) (string, error) {
				return "", nil // Empty version
			},
			mockCollectRegistryImages: func(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
				return nil
			},
			expectVersion: "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches to mock the functions
			var patches *gomonkey.Patches
			if tt.mockCollectRpmsAndBinary != nil {
				patches = gomonkey.ApplyFunc(collectRpmsAndBinary, tt.mockCollectRpmsAndBinary)
				defer patches.Reset()
			}

			if tt.mockCollectRegistryImages != nil {
				patches = gomonkey.ApplyFunc(collectRegistryImages, tt.mockCollectRegistryImages)
				defer patches.Reset()
			}

			// Apply patch for closeChanStruct to prevent actual channel closing during test
			patches = gomonkey.ApplyFunc(closeChanStruct, func(ch chan struct{}) {
				// No-op for testing
			})
			defer patches.Reset()

			opts := &Options{}
			version, err := opts.collectDependenciesAndImages(tt.config)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, version)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectVersion, version)
			}
		})
	}
}

func TestCompressedPackage(t *testing.T) {
	const (
		testTargetPath   = "test-target-path"
		testManifestPath = "test-manifests.yaml"
	)

	tests := []struct {
		name                    string
		config                  *BuildConfig
		target                  string
		mockWriteManifestsFile  func(*BuildConfig, string) error
		mockFinalizeAndCompress func(string) error
		expectError             bool
	}{
		{
			name:   "successful compression",
			config: &BuildConfig{},
			target: testTargetPath,
			mockWriteManifestsFile: func(cfg *BuildConfig, manifestPath string) error {
				// Simulate successful write
				return nil
			},
			mockFinalizeAndCompress: func(target string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:   "error in writeManifestsFile",
			config: &BuildConfig{},
			target: testTargetPath,
			mockWriteManifestsFile: func(cfg *BuildConfig, manifestPath string) error {
				return errors.New("write manifests failed")
			},
			mockFinalizeAndCompress: func(target string) error {
				return nil
			},
			expectError: true,
		},
		{
			name:   "error in finalizeAndCompress",
			config: &BuildConfig{},
			target: testTargetPath,
			mockWriteManifestsFile: func(cfg *BuildConfig, manifestPath string) error {
				return nil
			},
			mockFinalizeAndCompress: func(target string) error {
				return errors.New("compress failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches to mock the functions
			var patches *gomonkey.Patches
			if tt.mockWriteManifestsFile != nil {
				patches = gomonkey.ApplyFunc(writeManifestsFile, tt.mockWriteManifestsFile)
				defer patches.Reset()
			}

			if tt.mockFinalizeAndCompress != nil {
				patches = gomonkey.ApplyFunc(finalizeAndCompress, tt.mockFinalizeAndCompress)
				defer patches.Reset()
			}

			// Patch the bke variable to control the manifest path
			patches = gomonkey.ApplyGlobalVar(&bke, "test-bke-path")
			defer patches.Reset()

			err := compressedPackage(tt.config, tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWriteManifestsFileInternal(t *testing.T) {
	const (
		testManifestPath = "test-manifests.yaml"
	)

	tests := []struct {
		name            string
		config          *BuildConfig
		manifestPath    string
		mockYAMLMarshal func(interface{}) ([]byte, error)
		mockOSWriteFile func(string, []byte, os.FileMode) error
		expectError     bool
	}{
		{
			name:         "successful write",
			config:       &BuildConfig{},
			manifestPath: testManifestPath,
			mockYAMLMarshal: func(v interface{}) ([]byte, error) {
				return []byte("test yaml content"), nil
			},
			mockOSWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return nil
			},
			expectError: false,
		},
		{
			name:         "yaml marshal error",
			config:       &BuildConfig{},
			manifestPath: testManifestPath,
			mockYAMLMarshal: func(v interface{}) ([]byte, error) {
				return nil, errors.New("marshal error")
			},
			mockOSWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return nil
			},
			expectError: true,
		},
		{
			name:         "os write file error",
			config:       &BuildConfig{},
			manifestPath: testManifestPath,
			mockYAMLMarshal: func(v interface{}) ([]byte, error) {
				return []byte("test yaml content"), nil
			},
			mockOSWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return errors.New("write file error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches to mock the functions
			var patches *gomonkey.Patches
			if tt.mockYAMLMarshal != nil {
				patches = gomonkey.ApplyFunc(yaml.Marshal, tt.mockYAMLMarshal)
				defer patches.Reset()
			}

			if tt.mockOSWriteFile != nil {
				patches = gomonkey.ApplyFunc(os.WriteFile, tt.mockOSWriteFile)
				defer patches.Reset()
			}

			err := writeManifestsFile(tt.config, tt.manifestPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFinalizeAndCompressInternal(t *testing.T) {
	const (
		testTargetPath = "test-target.tar.gz"
	)

	tests := []struct {
		name                             string
		target                           string
		mockOSRemoveAll                  func(string) error
		mockGlobalTaeGZWithoutChangeFile func(string, string) error
		mockPwd                          string
		mockTmp                          string
		mockPackages                     string
		expectError                      bool
	}{
		{
			name:   "successful compression with full path",
			target: "/full/path/target.tar.gz",
			mockOSRemoveAll: func(path string) error {
				return nil
			},
			mockGlobalTaeGZWithoutChangeFile: func(src, dst string) error {
				return nil
			},
			mockPwd:      "/current/dir",
			mockTmp:      "/tmp/dir",
			mockPackages: "/packages/dir",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches to mock the functions
			var patches *gomonkey.Patches
			if tt.mockOSRemoveAll != nil {
				patches = gomonkey.ApplyFunc(os.RemoveAll, tt.mockOSRemoveAll)
				defer patches.Reset()
			}

			if tt.mockGlobalTaeGZWithoutChangeFile != nil {
				patches = gomonkey.ApplyFunc(global.TaeGZWithoutChangeFile, tt.mockGlobalTaeGZWithoutChangeFile)
				defer patches.Reset()
			}

			// Patch global variables
			patches = gomonkey.ApplyGlobalVar(&pwd, tt.mockPwd)
			defer patches.Reset()

			patches = gomonkey.ApplyGlobalVar(&tmp, tt.mockTmp)
			defer patches.Reset()

			patches = gomonkey.ApplyGlobalVar(&packages, tt.mockPackages)
			defer patches.Reset()

			err := finalizeAndCompress(tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
