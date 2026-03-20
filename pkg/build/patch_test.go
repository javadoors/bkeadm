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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

const (
	testUint64OneValue      = uint64(2)
	testFileModeReadOnly    = os.FileMode(0644)
	testFileModeReadWrite   = os.FileMode(0755)
	testFileModeReadOnly444 = os.FileMode(0444)
)

func TestPatch(t *testing.T) {
	tests := []struct {
		name                        string
		strategy                    string
		isDockerEnvironment         bool
		checkEnvironmentError       error
		loadAndValidateConfigResult *BuildConfig
		loadAndValidateConfigError  error
		prepareWorkspaceError       error
		collectFilesAndImagesError  error
		createPatchPackageError     error
	}{
		{
			name:                "successful patch with registry strategy",
			strategy:            "registry",
			isDockerEnvironment: true,
			loadAndValidateConfigResult: &BuildConfig{
				Registry: registry{
					ImageAddress: "registry.example.com",
					Architecture: []string{"amd64"},
				},
			},
			loadAndValidateConfigError: nil,
			prepareWorkspaceError:      nil,
			collectFilesAndImagesError: nil,
			createPatchPackageError:    nil,
		},
		{
			name:                "successful patch with oci strategy",
			strategy:            "oci",
			isDockerEnvironment: false, // OCI doesn't require Docker
			loadAndValidateConfigResult: &BuildConfig{
				Registry: registry{
					ImageAddress: "registry.example.com",
					Architecture: []string{"amd64"},
				},
			},
			loadAndValidateConfigError: nil,
			prepareWorkspaceError:      nil,
			collectFilesAndImagesError: nil,
			createPatchPackageError:    nil,
		},
		{
			name:                  "environment check fails",
			strategy:              "registry",
			isDockerEnvironment:   false,
			checkEnvironmentError: fmt.Errorf("docker environment required"),
		},
		{
			name:                       "config loading fails",
			strategy:                   "registry",
			isDockerEnvironment:        true,
			loadAndValidateConfigError: fmt.Errorf("config error"),
		},
		{
			name:                "workspace preparation fails",
			strategy:            "registry",
			isDockerEnvironment: true,
			loadAndValidateConfigResult: &BuildConfig{
				Registry: registry{
					ImageAddress: "registry.example.com",
					Architecture: []string{"amd64"},
				},
			},
			loadAndValidateConfigError: nil,
			prepareWorkspaceError:      fmt.Errorf("workspace error"),
		},
		{
			name:                "collect files and images fails",
			strategy:            "registry",
			isDockerEnvironment: true,
			loadAndValidateConfigResult: &BuildConfig{
				Registry: registry{
					ImageAddress: "registry.example.com",
					Architecture: []string{"amd64"},
				},
			},
			loadAndValidateConfigError: nil,
			prepareWorkspaceError:      nil,
			collectFilesAndImagesError: fmt.Errorf("collection error"),
		},
		{
			name:                "create patch package fails",
			strategy:            "registry",
			isDockerEnvironment: true,
			loadAndValidateConfigResult: &BuildConfig{
				Registry: registry{
					ImageAddress: "registry.example.com",
					Architecture: []string{"amd64"},
				},
			},
			loadAndValidateConfigError: nil,
			prepareWorkspaceError:      nil,
			collectFilesAndImagesError: nil,
			createPatchPackageError:    fmt.Errorf("package error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				File:     "test-config.yaml",
				Target:   "test-patch.tar.gz",
				Strategy: tt.strategy,
			}

			// Apply patches
			patches := gomonkey.ApplyFunc((*Options).checkEnvironment, func(o *Options) error {
				if tt.checkEnvironmentError != nil {
					return tt.checkEnvironmentError
				}
				return nil
			})
			defer patches.Reset()

			if tt.checkEnvironmentError == nil {
				patches = gomonkey.ApplyFunc((*Options).loadAndValidateConfig, func(o *Options) (*BuildConfig, error) {
					return tt.loadAndValidateConfigResult, tt.loadAndValidateConfigError
				})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc((*Options).prepareWorkspace, func(o *Options) error {
					return tt.prepareWorkspaceError
				})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc((*Options).collectFilesAndImages, func(o *Options, cfg *BuildConfig) error {
					return tt.collectFilesAndImagesError
				})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc((*Options).createPatchPackage, func(o *Options, cfg *BuildConfig) error {
					return tt.createPatchPackageError
				})
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			o.Patch()

			// The function should complete without panic
			assert.True(t, true)
		})
	}
}

func TestCheckEnvironment(t *testing.T) {
	tests := []struct {
		name                string
		strategy            string
		isDockerEnvironment bool
		expectedError       bool
	}{
		{
			name:                "registry strategy with docker",
			strategy:            "registry",
			isDockerEnvironment: true,
			expectedError:       false,
		},
		{
			name:                "registry strategy without docker",
			strategy:            "registry",
			isDockerEnvironment: false,
			expectedError:       true,
		},
		{
			name:                "oci strategy without docker (allowed)",
			strategy:            "oci",
			isDockerEnvironment: false,
			expectedError:       false,
		},
		{
			name:                "oci strategy with docker (also allowed)",
			strategy:            "oci",
			isDockerEnvironment: true,
			expectedError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{Strategy: tt.strategy}

			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
				return tt.isDockerEnvironment
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := o.checkEnvironment()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadAndValidateConfig(t *testing.T) {
	tests := []struct {
		name                 string
		strategy             string
		mockReadFile         func(string) ([]byte, error)
		mockUnmarshal        func([]byte, interface{}) error
		registryImageAddress string
		registryArchitecture []string
		expectedError        bool
	}{
		{
			name:     "valid config with registry strategy",
			strategy: "registry",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("registry:\n  imageAddress: registry.example.com\n  architecture: [\"amd64\"]"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				cfg := v.(*BuildConfig)
				cfg.Registry.ImageAddress = "registry.example.com"
				cfg.Registry.Architecture = []string{"amd64"}
				return nil
			},
			registryImageAddress: "registry.example.com",
			registryArchitecture: []string{"amd64"},
			expectedError:        false,
		},
		{
			name:     "missing registry params with registry strategy",
			strategy: "registry",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("registry:\n  imageAddress: \"\"\n  architecture: []"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				cfg := v.(*BuildConfig)
				cfg.Registry.ImageAddress = ""
				cfg.Registry.Architecture = []string{}
				return nil
			},
			registryImageAddress: "",
			registryArchitecture: []string{},
			expectedError:        true,
		},
		{
			name:     "valid config with oci strategy (registry params not required)",
			strategy: "oci",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("registry:\n  imageAddress: \"\"\n  architecture: []"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				cfg := v.(*BuildConfig)
				cfg.Registry.ImageAddress = ""
				cfg.Registry.Architecture = []string{}
				return nil
			},
			registryImageAddress: "",
			registryArchitecture: []string{},
			expectedError:        false,
		},
		{
			name:     "read file error",
			strategy: "registry",
			mockReadFile: func(filename string) ([]byte, error) {
				return nil, fmt.Errorf("read error")
			},
			expectedError: true,
		},
		{
			name:     "unmarshal error",
			strategy: "registry",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("invalid yaml"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				return fmt.Errorf("unmarshal error")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				File:     "test-config.yaml",
				Strategy: tt.strategy,
			}

			patches := gomonkey.ApplyFunc(os.ReadFile, tt.mockReadFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(yaml.Unmarshal, tt.mockUnmarshal)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			cfg, err := o.loadAndValidateConfig()

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, cfg)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, cfg)
			}
		})
	}
}

func TestPatchPrepareWorkspace(t *testing.T) {
	tests := []struct {
		name          string
		mockPrepare   func() error
		mockRemoveAll func(string) error
		expectedError bool
	}{
		{
			name:          "successful preparation",
			mockPrepare:   func() error { return nil },
			mockRemoveAll: func(path string) error { return nil },
			expectedError: false,
		},
		{
			name:          "prepare fails",
			mockPrepare:   func() error { return fmt.Errorf("prepare error") },
			expectedError: true,
		},
		{
			name:          "remove all fails (but doesn't cause error)",
			mockPrepare:   func() error { return nil },
			mockRemoveAll: func(path string) error { return fmt.Errorf("remove error") },
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{}

			patches := gomonkey.ApplyFunc(prepare, tt.mockPrepare)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			// Mock packages variable
			patches = gomonkey.ApplyGlobalVar(&packages, "/tmp/packages")
			defer patches.Reset()

			err := o.prepareWorkspace()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectFilesAndImages(t *testing.T) {
	cfg := &BuildConfig{}

	// Apply patches
	patches := gomonkey.ApplyFunc((*Options).collectHostFiles, func(o *Options, cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*Options).collectPatchFiles, func(o *Options, cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*Options).collectChartFiles, func(o *Options, cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*Options).collectImages, func(o *Options, cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(closeChanStruct, func(ch chan struct{}) {})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
	defer patches.Reset()

	o := &Options{}
	err := o.collectFilesAndImages(cfg)

	assert.NoError(t, err)
}

func TestCollectHostFiles(t *testing.T) {
	cfg := &BuildConfig{}
	stopChan := make(chan struct{})
	var errNumber uint64

	tests := []struct {
		name             string
		mockBuildFiles   func([]File, string, <-chan struct{}) error
		mockTransferFile func(string, string) error
		expectedError    bool
	}{
		{
			name:             "successful collection",
			mockBuildFiles:   func(files []File, dest string, stopChan <-chan struct{}) error { return nil },
			mockTransferFile: func(src, dst string) error { return nil },
			expectedError:    false,
		},
		{
			name:           "build files fails",
			mockBuildFiles: func(files []File, dest string, stopChan <-chan struct{}) error { return fmt.Errorf("build error") },
			expectedError:  true,
		},
		{
			name:             "transfer file fails",
			mockBuildFiles:   func(files []File, dest string, stopChan <-chan struct{}) error { return nil },
			mockTransferFile: func(src, dst string) error { return fmt.Errorf("transfer error") },
			expectedError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{}

			patches := gomonkey.ApplyFunc(buildFiles, tt.mockBuildFiles)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(transferFile, tt.mockTransferFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			// Mock tmpPackagesFiles and bkeVolumes
			patches = gomonkey.ApplyGlobalVar(&tmpPackagesFiles, "/tmp/files")
			defer patches.Reset()

			patches = gomonkey.ApplyGlobalVar(&bkeVolumes, "/tmp/volumes")
			defer patches.Reset()

			err := o.collectHostFiles(cfg, stopChan, &errNumber)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectPatchFiles(t *testing.T) {
	cfg := &BuildConfig{}
	stopChan := make(chan struct{})
	var errNumber uint64

	tests := []struct {
		name             string
		mockBuildFiles   func([]File, string, <-chan struct{}) error
		mockTransferFile func(string, string) error
		expectedError    bool
	}{
		{
			name:             "successful collection",
			mockBuildFiles:   func(files []File, dest string, stopChan <-chan struct{}) error { return nil },
			mockTransferFile: func(src, dst string) error { return nil },
			expectedError:    false,
		},
		{
			name:           "build files fails",
			mockBuildFiles: func(files []File, dest string, stopChan <-chan struct{}) error { return fmt.Errorf("build error") },
			expectedError:  true,
		},
		{
			name:             "transfer file fails",
			mockBuildFiles:   func(files []File, dest string, stopChan <-chan struct{}) error { return nil },
			mockTransferFile: func(src, dst string) error { return fmt.Errorf("transfer error") },
			expectedError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{}

			patches := gomonkey.ApplyFunc(buildFiles, tt.mockBuildFiles)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(transferFile, tt.mockTransferFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			// Mock tmpPackagesPatches and bkeVolumes
			patches = gomonkey.ApplyGlobalVar(&tmpPackagesPatches, "/tmp/patches")
			defer patches.Reset()

			patches = gomonkey.ApplyGlobalVar(&bkeVolumes, "/tmp/volumes")
			defer patches.Reset()

			err := o.collectPatchFiles(cfg, stopChan, &errNumber)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollectImages(t *testing.T) {
	cfg := &BuildConfig{}
	stopChan := make(chan struct{})
	var errNumber uint64

	tests := []struct {
		name                     string
		strategy                 string
		mockBuildRegistry        func(string, []string) error
		mockSyncImagesByStrategy func(*Options, *BuildConfig, chan struct{}, *uint64) error
		expectedError            bool
	}{
		{
			name:                     "registry strategy with successful operations",
			strategy:                 "registry",
			mockBuildRegistry:        func(imageAddr string, arch []string) error { return nil },
			mockSyncImagesByStrategy: func(o *Options, cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error { return nil },
			expectedError:            false,
		},
		{
			name:                     "oci strategy (skip buildRegistry)",
			strategy:                 "oci",
			mockSyncImagesByStrategy: func(o *Options, cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error { return nil },
			expectedError:            false,
		},
		{
			name:              "build registry fails",
			strategy:          "registry",
			mockBuildRegistry: func(imageAddr string, arch []string) error { return fmt.Errorf("build error") },
			expectedError:     true,
		},
		{
			name:              "sync images fails",
			strategy:          "registry",
			mockBuildRegistry: func(imageAddr string, arch []string) error { return nil },
			mockSyncImagesByStrategy: func(o *Options, cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
				return fmt.Errorf("sync error")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{Strategy: tt.strategy}

			if tt.strategy != "oci" {
				patches := gomonkey.ApplyFunc(buildRegistry, tt.mockBuildRegistry)
				defer patches.Reset()
			}

			patches := gomonkey.ApplyFunc((*Options).syncImagesByStrategy, tt.mockSyncImagesByStrategy)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := o.collectImages(cfg, stopChan, &errNumber)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncImagesByStrategy(t *testing.T) {
	cfg := &BuildConfig{}
	stopChan := make(chan struct{})
	var errNumber uint64

	tests := []struct {
		name            string
		strategy        string
		mockSyncRepo    func(*BuildConfig, chan struct{}) error
		mockSyncRepoOCI func(*BuildConfig, chan struct{}) error
		expectedError   bool
	}{
		{
			name:            "registry strategy",
			strategy:        "registry",
			mockSyncRepo:    func(cfg *BuildConfig, stopChan chan struct{}) error { return nil },
			mockSyncRepoOCI: func(cfg *BuildConfig, stopChan chan struct{}) error { return nil },
			expectedError:   false,
		},
		{
			name:            "oci strategy",
			strategy:        "oci",
			mockSyncRepo:    func(cfg *BuildConfig, stopChan chan struct{}) error { return nil },
			mockSyncRepoOCI: func(cfg *BuildConfig, stopChan chan struct{}) error { return nil },
			expectedError:   false,
		},
		{
			name:          "unknown strategy",
			strategy:      "unknown",
			expectedError: true,
		},
		{
			name:          "registry sync fails",
			strategy:      "registry",
			mockSyncRepo:  func(cfg *BuildConfig, stopChan chan struct{}) error { return fmt.Errorf("sync error") },
			expectedError: true,
		},
		{
			name:            "oci sync fails",
			strategy:        "oci",
			mockSyncRepoOCI: func(cfg *BuildConfig, stopChan chan struct{}) error { return fmt.Errorf("oci sync error") },
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{Strategy: tt.strategy}

			if tt.strategy == "registry" {
				patches := gomonkey.ApplyFunc(syncRepo, tt.mockSyncRepo)
				defer patches.Reset()
			} else if tt.strategy == "oci" {
				patches := gomonkey.ApplyFunc(syncRepoOCI, tt.mockSyncRepoOCI)
				defer patches.Reset()
			}

			patches := gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := o.syncImagesByStrategy(cfg, stopChan, &errNumber)

			if tt.expectedError {
				assert.Error(t, err)

			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreatePatchPackage(t *testing.T) {
	cfg := &BuildConfig{
		Registry: registry{
			Architecture: []string{"amd64"},
		},
	}

	tests := []struct {
		name                string
		target              string
		mockStat            func(string) (os.FileInfo, error)
		mockCompressedPatch func(*BuildConfig, string) error
		expectedTarget      string
		expectedError       bool
	}{
		{
			name:   "target specified",
			target: "custom-target.tar.gz",
			mockStat: func(name string) (os.FileInfo, error) {
				return &fakeFileInfo{name: "test-config.yaml"}, nil
			},
			mockCompressedPatch: func(cfg *BuildConfig, target string) error { return nil },
			expectedTarget:      "custom-target.tar.gz",
			expectedError:       false,
		},
		{
			name:   "target not specified (auto-generated)",
			target: "",
			mockStat: func(name string) (os.FileInfo, error) {
				return &fakeFileInfo{name: "test-config.yaml"}, nil
			},
			mockCompressedPatch: func(cfg *BuildConfig, target string) error { return nil },
			expectedTarget:      fmt.Sprintf("bke-patch-test-config-amd64-%s.tar.gz", time.Now().Format("20060102150405")),
			expectedError:       false,
		},
		{
			name:   "stat fails",
			target: "",
			mockStat: func(name string) (os.FileInfo, error) {
				return nil, fmt.Errorf("stat error")
			},
			expectedError: true,
		},
		{
			name:   "compressed patch fails",
			target: "test-target.tar.gz",
			mockStat: func(name string) (os.FileInfo, error) {
				return &fakeFileInfo{name: "test-config.yaml"}, nil
			},
			mockCompressedPatch: func(cfg *BuildConfig, target string) error { return fmt.Errorf("compress error") },
			expectedError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				File:   "test-config.yaml",
				Target: tt.target,
			}

			patches := gomonkey.ApplyFunc(os.Stat, tt.mockStat)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(compressedPatch, tt.mockCompressedPatch)
			defer patches.Reset()

			err := o.createPatchPackage(cfg)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.target == "" {
					// For auto-generated target, check that it starts with the expected prefix
					assert.Contains(t, o.Target, "bke-patch-test-config-amd64-")
					assert.True(t, strings.HasSuffix(o.Target, ".tar.gz"))
				} else {
					assert.Equal(t, tt.expectedTarget, o.Target)
				}
			}
		})
	}
}

func TestTransferFile(t *testing.T) {
	// Create temporary directories for testing
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	// Create a test file in the source directory
	testFile := filepath.Join(srcDir, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), testFileModeReadOnly)
	assert.NoError(t, err)

	// Create a subdirectory with a file
	subDir := filepath.Join(srcDir, "subdir")
	err = os.MkdirAll(subDir, testFileModeReadWrite)
	assert.NoError(t, err)

	subFile := filepath.Join(subDir, "sub.txt")
	err = os.WriteFile(subFile, []byte("sub content"), testFileModeReadOnly)
	assert.NoError(t, err)

	// Test successful transfer
	err = transferFile(srcDir, dstDir)
	assert.NoError(t, err)

	// Verify that files were transferred
	transferredFile := filepath.Join(dstDir, "test.txt")
	assert.True(t, utils.Exists(transferredFile))

	transferredSubFile := filepath.Join(dstDir, "subdir", "sub.txt")
	assert.True(t, utils.Exists(transferredSubFile))

	// Check content
	content, err := os.ReadFile(transferredFile)
	assert.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	subContent, err := os.ReadFile(transferredSubFile)
	assert.NoError(t, err)
	assert.Equal(t, "sub content", string(subContent))
}

func TestTransferFileErrors(t *testing.T) {
	// Test error cases
	err := transferFile("/nonexistent/src", "/dst")
	assert.Error(t, err)

	// Test with a file as source instead of directory
	tempFile := filepath.Join(t.TempDir(), "temp.txt")
	err = os.WriteFile(tempFile, []byte("content"), testFileModeReadOnly)
	assert.NoError(t, err)

	transferFile(tempFile, t.TempDir())
}

func TestCompressedPatch(t *testing.T) {
	cfg := &BuildConfig{}
	target := "test-patch.tar.gz"

	tests := []struct {
		name                    string
		mockRename              func(string, string) error
		mockWriteManifestsFile  func(*BuildConfig, string) error
		mockFinalizeAndCompress func(string) error
		mockRemoveAll           func(string) error
		expectedError           bool
	}{
		{
			name:                    "successful compression",
			mockRename:              func(oldpath, newpath string) error { return nil },
			mockWriteManifestsFile:  func(cfg *BuildConfig, manifestPath string) error { return nil },
			mockFinalizeAndCompress: func(target string) error { return nil },
			mockRemoveAll:           func(path string) error { return nil },
			expectedError:           false,
		},
		{
			name:          "rename fails",
			mockRename:    func(oldpath, newpath string) error { return fmt.Errorf("rename error") },
			expectedError: true,
		},
		{
			name:                   "write manifests fails",
			mockRename:             func(oldpath, newpath string) error { return nil },
			mockWriteManifestsFile: func(cfg *BuildConfig, manifestPath string) error { return fmt.Errorf("manifest error") },
			expectedError:          true,
		},
		{
			name:                    "finalize and compress fails",
			mockRename:              func(oldpath, newpath string) error { return nil },
			mockWriteManifestsFile:  func(cfg *BuildConfig, manifestPath string) error { return nil },
			mockFinalizeAndCompress: func(target string) error { return fmt.Errorf("compress error") },
			expectedError:           true,
		},
		{
			name:                    "remove all fails",
			mockRename:              func(oldpath, newpath string) error { return nil },
			mockWriteManifestsFile:  func(cfg *BuildConfig, manifestPath string) error { return nil },
			mockFinalizeAndCompress: func(target string) error { return nil },
			mockRemoveAll:           func(path string) error { return fmt.Errorf("remove error") },
			expectedError:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.Rename, tt.mockRename)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(writeManifestsFile, tt.mockWriteManifestsFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(finalizeAndCompress, tt.mockFinalizeAndCompress)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
			defer patches.Reset()
			// Mock variables
			patches = gomonkey.ApplyGlobalVar(&bke, "/tmp/bke")
			defer patches.Reset()

			patches = gomonkey.ApplyGlobalVar(&packages, "/tmp/packages")
			defer patches.Reset()

			err := compressedPatch(cfg, target)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
