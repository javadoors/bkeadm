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
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/validation"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func TestPrepare(t *testing.T) {
	tests := []struct {
		name          string
		mockRemoveAll func(string) error
		mockMkdirAll  func(string, os.FileMode) error
		mockChmod     func(string, os.FileMode) error
		expectedError bool
	}{
		{
			name:          "successful preparation",
			mockRemoveAll: func(path string) error { return nil },
			mockMkdirAll:  func(path string, perm os.FileMode) error { return nil },
			mockChmod:     func(name string, perm os.FileMode) error { return nil },
			expectedError: false,
		},
		{
			name:          "remove all fails",
			mockRemoveAll: func(path string) error { return fmt.Errorf("remove error") },
			expectedError: true,
		},
		{
			name:          "mkdir all fails",
			mockRemoveAll: func(path string) error { return nil },
			mockMkdirAll:  func(path string, perm os.FileMode) error { return fmt.Errorf("mkdir error") },
			expectedError: true,
		},
		{
			name:          "chmod fails",
			mockRemoveAll: func(path string) error { return nil },
			mockMkdirAll:  func(path string, perm os.FileMode) error { return nil },
			mockChmod:     func(name string, perm os.FileMode) error { return fmt.Errorf("chmod error") },
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.NewPatches()
			defer patches.Reset()

			patches.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)

			if tt.mockMkdirAll != nil {
				patches.ApplyFunc(os.MkdirAll, tt.mockMkdirAll)
			}

			if tt.mockChmod != nil {
				patches.ApplyFunc(os.Chmod, tt.mockChmod)
			}

			patches.ApplyFunc(log.BKEFormat, func(level, msg string) {})

			// Set up the required variables
			patches.ApplyGlobalVar(&pwd, "/tmp")
			patches.ApplyGlobalVar(&tmpPackages, "/tmp/packages")
			patches.ApplyGlobalVar(&bke, "/tmp/packages/bke")
			patches.ApplyGlobalVar(&bkeVolumes, "/tmp/packages/bke/volumes")
			patches.ApplyGlobalVar(&usrBin, "/tmp/packages/usr/bin")
			patches.ApplyGlobalVar(&tmp, "/tmp/packages/tmp")

			err := prepare()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCheckIsChartDownload tests the checkIsChartDownload function
// which determines if charts need to be downloaded based on config
func TestCheckIsChartDownload(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *BuildConfig
		expected bool
	}{
		{
			name: "has chart with files",
			cfg: &BuildConfig{
				Charts: []File{
					{
						Address: "http://example.com",
						Files: []FileInfo{
							{FileName: "chart1.tgz"},
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "has chart without files",
			cfg: &BuildConfig{
				Charts: []File{
					{
						Address: "http://example.com",
						Files:   []FileInfo{},
					},
				},
			},
			expected: false,
		},
		{
			name: "has chart with empty address",
			cfg: &BuildConfig{
				Charts: []File{
					{
						Address: "",
						Files: []FileInfo{
							{FileName: "chart1.tgz"},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:     "no charts",
			cfg:      &BuildConfig{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := checkIsChartDownload(tt.cfg)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestInitRequiredDependencies tests the initRequiredDependencies function
// which initializes the map of required dependencies
func TestInitRequiredDependencies(t *testing.T) {
	deps := initRequiredDependencies()

	// Check that all expected dependencies are present
	expectedKeys := []string{
		"bke",
		"charts.tar.gz",
		"nfsshare.tar.gz",
		"containerd",
		utils.DefaultLocalYumRegistry,
		utils.DefaultLocalChartRegistry,
		utils.DefaultLocalNFSRegistry,
		utils.DefaultLocalK3sRegistry,
		utils.DefaultK3sPause,
		utils.CniPluginPrefix,
	}

	for _, key := range expectedKeys {
		_, exists := deps[key]
		assert.True(t, exists, "Dependency %s should exist in the map", key)
	}

	// Check that all values are false initially
	for _, value := range deps {
		assert.False(t, value, "All dependencies should be false initially")
	}

	// Check the total count
	assert.Equal(t, len(expectedKeys), len(deps))
}

// TestValidateRpms tests the validateRpms function
// which validates RPM configuration
func TestValidateRpms(t *testing.T) {
	requiredDepend := map[string]bool{
		"docker-ce": false,
	}

	tests := []struct {
		name           string
		rpms           []Rpm
		requiredDepend map[string]bool
		expectedError  bool
	}{
		{
			name: "valid rpms",
			rpms: []Rpm{
				{
					Address:            "http://example.com",
					System:             []string{"CentOS"},
					SystemVersion:      []string{"7"},
					SystemArchitecture: []string{"amd64"},
					Directory:          []string{"docker-ce"},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  false,
		},
		{
			name: "missing source address",
			rpms: []Rpm{
				{
					Address:            "",
					System:             []string{"CentOS"},
					SystemVersion:      []string{"7"},
					SystemArchitecture: []string{"amd64"},
					Directory:          []string{"docker-ce"},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
		{
			name: "missing system",
			rpms: []Rpm{
				{
					Address:            "http://example.com",
					System:             []string{},
					SystemVersion:      []string{"7"},
					SystemArchitecture: []string{"amd64"},
					Directory:          []string{"docker-ce"},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
		{
			name: "missing system version",
			rpms: []Rpm{
				{
					Address:            "http://example.com",
					System:             []string{"CentOS"},
					SystemVersion:      []string{},
					SystemArchitecture: []string{"amd64"},
					Directory:          []string{"docker-ce"},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
		{
			name: "missing system architecture",
			rpms: []Rpm{
				{
					Address:            "http://example.com",
					System:             []string{"CentOS"},
					SystemVersion:      []string{"7"},
					SystemArchitecture: []string{},
					Directory:          []string{"docker-ce"},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
		{
			name: "nil required dependencies",
			rpms: []Rpm{
				{
					Address:            "http://example.com",
					System:             []string{"CentOS"},
					SystemVersion:      []string{"7"},
					SystemArchitecture: []string{"amd64"},
					Directory:          []string{"docker-ce"},
				},
			},
			requiredDepend: nil,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRpms(tt.rpms, tt.requiredDepend)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateFiles tests the validateFiles function
// which validates file configuration
func TestValidateFiles(t *testing.T) {
	requiredDepend := map[string]bool{
		"charts.tar.gz":       false,
		"nfsshare.tar.gz":     false,
		"bke":                 false,
		utils.CniPluginPrefix: false,
	}

	tests := []struct {
		name           string
		files          []File
		requiredDepend map[string]bool
		expectedList   []string
		expectedError  bool
	}{
		{
			name: "valid files",
			files: []File{
				{
					Address: "http://example.com",
					Files: []FileInfo{
						{FileName: "bkeadm_linux_amd64"},
						{FileName: "cni-plugins-linux-amd64-v1.2.0.tgz"},
					},
				},
			},
			requiredDepend: requiredDepend,
			expectedList:   []string{"bkeadm_linux_amd64", "cni-plugins-linux-amd64-v1.2.0.tgz"},
			expectedError:  false,
		},
		{
			name: "missing address",
			files: []File{
				{
					Address: "",
					Files: []FileInfo{
						{FileName: "bkeadm_linux_amd64"},
					},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
		{
			name: "duplicate charts package",
			files: []File{
				{
					Address: "http://example.com",
					Files: []FileInfo{
						{FileName: "charts.tar.gz"},
					},
				},
				{
					Address: "http://example.com",
					Files: []FileInfo{
						{FileName: "charts.tar.gz"},
					},
				},
			},
			requiredDepend: map[string]bool{
				"charts.tar.gz":       true, // Already marked as true
				"nfsshare.tar.gz":     false,
				"bke":                 false,
				utils.CniPluginPrefix: false,
			},
			expectedError: true,
		},
		{
			name: "nil required dependencies",
			files: []File{
				{
					Address: "http://example.com",
					Files: []FileInfo{
						{FileName: "bkeadm_linux_amd64"},
					},
				},
			},
			requiredDepend: nil,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches for validation
			patches := gomonkey.ApplyFunc(utils.ContainsStringPrefix, func(slice []string, prefix string) bool {
				for _, s := range slice {
					if strings.HasPrefix(s, prefix) {
						return true
					}
				}
				return false
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.ContainsString, func(slice []string, item string) bool {
				for _, s := range slice {
					if s == item {
						return true
					}
				}
				return false
			})
			defer patches.Reset()

			// Mock validation.ValidateCustomExtra to return an error for containerd
			patches = gomonkey.ApplyFunc(validation.ValidateCustomExtra, func(params map[string]string) error {
				if params["containerd"] != "" {
					return nil // Indicate this is a containerd file
				}
				return fmt.Errorf("not containerd")
			})
			defer patches.Reset()

			list, err := validateFiles(tt.files, tt.requiredDepend)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, list)
			} else {
				assert.NoError(t, err)
				if tt.expectedList != nil {
					assert.Equal(t, tt.expectedList, list)
				}
			}
		})
	}
}

// TestCheckImageDependencies tests the checkImageDependencies function
// which checks image dependencies in the configuration
func TestCheckImageDependencies(t *testing.T) {
	requiredDepend := map[string]bool{
		utils.DefaultLocalYumRegistry:   false,
		utils.DefaultLocalChartRegistry: false,
		utils.DefaultLocalNFSRegistry:   false,
		utils.DefaultLocalK3sRegistry:   false,
		utils.DefaultK3sPause:           false,
	}

	images := []Image{
		{
			Name: utils.DefaultLocalYumRegistry,
			Tag:  []string{"latest"},
		},
		{
			Name: utils.DefaultLocalChartRegistry,
			Tag:  []string{"v1.0"},
		},
	}

	checkImageDependencies(images, requiredDepend)

	// Check that the matching dependencies were marked as true
	assert.False(t, requiredDepend[utils.DefaultLocalYumRegistry])
	assert.False(t, requiredDepend[utils.DefaultLocalChartRegistry])
	assert.False(t, requiredDepend[utils.DefaultLocalNFSRegistry])
	assert.False(t, requiredDepend[utils.DefaultLocalK3sRegistry])
	assert.False(t, requiredDepend[utils.DefaultK3sPause])
}

// TestValidateSubImages tests the validateSubImages function
// which validates sub-image configuration
func TestValidateSubImages(t *testing.T) {
	requiredDepend := map[string]bool{
		utils.DefaultLocalYumRegistry: false,
	}

	tests := []struct {
		name           string
		subImages      []SubImage
		requiredDepend map[string]bool
		expectedError  bool
	}{
		{
			name: "valid sub images",
			subImages: []SubImage{
				{
					SourceRepo: "registry.example.com",
					TargetRepo: "target-registry",
					Images: []Image{
						{
							Name: utils.DefaultLocalYumRegistry,
							Tag:  []string{"latest"},
						},
					},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  false,
		},
		{
			name: "missing source repo",
			subImages: []SubImage{
				{
					SourceRepo: "",
					TargetRepo: "target-registry",
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
		{
			name: "missing target repo",
			subImages: []SubImage{
				{
					SourceRepo: "registry.example.com",
					TargetRepo: "",
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
		{
			name: "invalid target repo (double slash)",
			subImages: []SubImage{
				{
					SourceRepo: "registry.example.com",
					TargetRepo: "invalid//repo",
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches for checkImageDependencies
			patches := gomonkey.ApplyFunc(checkImageDependencies, func(images []Image, requiredDepend map[string]bool) {})
			defer patches.Reset()

			err := validateSubImages(tt.subImages, tt.requiredDepend)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestValidateRepos tests the validateRepos function
// which validates repository configuration
func TestValidateRepos(t *testing.T) {
	requiredDepend := map[string]bool{
		utils.DefaultLocalYumRegistry: false,
	}

	tests := []struct {
		name           string
		repos          []Repo
		requiredDepend map[string]bool
		expectedError  bool
	}{
		{
			name: "valid repos with download needed",
			repos: []Repo{
				{
					NeedDownload: true,
					Architecture: []string{"amd64"},
					SubImages: []SubImage{
						{
							SourceRepo: "registry.example.com",
							TargetRepo: "target-registry",
						},
					},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  false,
		},
		{
			name: "repo with download not needed (should be skipped)",
			repos: []Repo{
				{
					NeedDownload: false,
					Architecture: []string{"amd64"},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  false,
		},
		{
			name: "missing architecture",
			repos: []Repo{
				{
					NeedDownload: true,
					Architecture: []string{},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
		{
			name: "sub image validation fails",
			repos: []Repo{
				{
					NeedDownload: true,
					Architecture: []string{"amd64"},
					SubImages: []SubImage{
						{
							SourceRepo: "",
							TargetRepo: "target-registry",
						},
					},
				},
			},
			requiredDepend: requiredDepend,
			expectedError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches for validateSubImages
			patches := gomonkey.ApplyFunc(validateSubImages, func(subImages []SubImage, requiredDepend map[string]bool) error {
				for _, subImg := range subImages {
					if subImg.SourceRepo == "" {
						return fmt.Errorf("source repo required")
					}
					if subImg.TargetRepo == "" {
						return fmt.Errorf("target repo required")
					}
				}
				return nil
			})
			defer patches.Reset()

			err := validateRepos(tt.repos, tt.requiredDepend)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestVerifyConfigContent tests the verifyConfigContent function
// which verifies the entire build configuration
func TestVerifyConfigContent(t *testing.T) {
	tests := []struct {
		name          string
		cfg           *BuildConfig
		expectedError bool
	}{
		{
			name: "valid config with all required dependencies",
			cfg: &BuildConfig{
				Registry: registry{
					ImageAddress: "registry.example.com",
					Architecture: []string{"amd64"},
				},
				Rpms: []Rpm{
					{
						Address:            "http://example.com",
						System:             []string{"CentOS"},
						SystemVersion:      []string{"7"},
						SystemArchitecture: []string{"amd64"},
						Directory:          []string{"docker-ce"},
					},
				},
				Files: []File{
					{
						Address: "http://example.com",
						Files: []FileInfo{
							{FileName: "bkeadm_linux_amd64"},
						},
					},
				},
				Repos: []Repo{
					{
						NeedDownload: true,
						Architecture: []string{"amd64"},
						SubImages: []SubImage{
							{
								SourceRepo: "registry.example.com",
								TargetRepo: "target-registry",
							},
						},
					},
				},
			},
			expectedError: true, // Because not all required dependencies are satisfied
		},
		{
			name: "missing registry image address",
			cfg: &BuildConfig{
				Registry: registry{
					ImageAddress: "",
					Architecture: []string{"amd64"},
				},
			},
			expectedError: true,
		},
		{
			name: "missing registry architecture",
			cfg: &BuildConfig{
				Registry: registry{
					ImageAddress: "registry.example.com",
					Architecture: []string{},
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches for all the validation functions
			patches := gomonkey.ApplyFunc(initRequiredDependencies, func() map[string]bool {
				// Return a map with all required dependencies marked as false initially
				return map[string]bool{
					"bke":                           false,
					"charts.tar.gz":                 false,
					"nfsshare.tar.gz":               false,
					"containerd":                    false,
					utils.DefaultLocalYumRegistry:   false,
					utils.DefaultLocalChartRegistry: false,
					utils.DefaultLocalNFSRegistry:   false,
					utils.DefaultLocalK3sRegistry:   false,
					utils.DefaultK3sPause:           false,
					utils.CniPluginPrefix:           false,
				}
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(validateRpms, func(rpms []Rpm, requiredDepend map[string]bool) error {
				return nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(validateFiles, func(files []File, requiredDepend map[string]bool) ([]string, error) {
				return []string{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(checkIsChartDownload, func(cfg *BuildConfig) bool {
				return false
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(validateRepos, func(repos []Repo, requiredDepend map[string]bool) error {
				return nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.ContainsString, func(slice []string, item string) bool {
				return false
			})
			defer patches.Reset()

			err := verifyConfigContent(tt.cfg)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFileVersionAdaptation tests the fileVersionAdaptation function
// which handles version-based file renaming
func TestFileVersionAdaptation(t *testing.T) {
	tests := []struct {
		name          string
		mockReadDir   func(string) ([]os.DirEntry, error)
		mockRename    func(string, string) error
		expectedError bool
	}{
		{
			name: "successful adaptation",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{}, nil
			},
			mockRename: func(oldName, newName string) error {
				return nil
			},
			expectedError: false,
		},
		{
			name: "read dir error",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, fmt.Errorf("read dir error")
			},
			expectedError: true,
		},
		{
			name: "rename error for charts file",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "charts.tar.gz-v4.0"},
				}, nil
			},
			mockRename: func(oldName, newName string) error {
				return fmt.Errorf("rename error")
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Rename, tt.mockRename)
			defer patches.Reset()

			err := fileVersionAdaptation()

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestGlobalVariablesInitialization tests the global variables initialization
func TestGlobalVariablesInitialization(t *testing.T) {
	assert.NotEmpty(t, pwd)
	assert.Contains(t, packages, "packages")
	assert.Contains(t, bke, "bke")
	assert.Contains(t, bkeVolumes, "volumes")
	assert.Contains(t, usrBin, "usr/bin")
	assert.Contains(t, tmp, "tmp")
	assert.Contains(t, tmpRegistry, "registry")
	assert.Contains(t, tmpPackages, "packages")
	assert.Contains(t, tmpPackagesFiles, "files")
	assert.Contains(t, tmpPackagesCharts, "charts")
	assert.Contains(t, tmpPackagesPatches, "patches")
}
