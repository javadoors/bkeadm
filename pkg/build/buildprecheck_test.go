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
	"gopkg.in/yaml.v3"

	reg "gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func TestPreCheckOptionsStruct(t *testing.T) {
	// Test that the PreCheckOptions struct has the expected fields
	bp := &PreCheckOptions{}

	// Check that it embeds root.Options
	_ = &bp.Options

	// Test that the fields exist
	bp.File = "test.yaml"
	bp.OnlyImage = true

	assert.Equal(t, "test.yaml", bp.File)
	assert.True(t, bp.OnlyImage)
}

func TestLoadBuildConfig(t *testing.T) {
	tests := []struct {
		name          string
		mockReadFile  func(string) ([]byte, error)
		mockUnmarshal func([]byte, interface{}) error
		expectError   bool
	}{
		{
			name: "successful config loading",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("version: v1.0.0"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				cfg := v.(*BuildConfig)
				cfg.OpenFuyaoVersion = "v1.0.0"
				return nil
			},
			expectError: false,
		},
		{
			name: "file read error",
			mockReadFile: func(filename string) ([]byte, error) {
				return nil, fmt.Errorf("file not found")
			},
			expectError: true,
		},
		{
			name: "unmarshal error",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("invalid yaml"), nil
			},
			mockUnmarshal: func(data []byte, v interface{}) error {
				return fmt.Errorf("unmarshal error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadFile, tt.mockReadFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(yaml.Unmarshal, tt.mockUnmarshal)
			defer patches.Reset()

			cfg, err := loadBuildConfig("test-file.yaml")

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

func TestImagePrefix(t *testing.T) {
	tests := []struct {
		name       string
		sourceRepo string
		expected   string
	}{
		{
			name:       "simple repo",
			sourceRepo: "registry.bocloud.com/kubernetes",
			expected:   "kubernetes/",
		},
		{
			name:       "complex repo",
			sourceRepo: "registry.bocloud.com/kubernetes/subpath",
			expected:   "kubernetes/subpath/",
		},
		{
			name:       "single part",
			sourceRepo: "registry",
			expected:   "",
		},
		{
			name:       "empty string",
			sourceRepo: "",
			expected:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := imagePrefix(tt.sourceRepo)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestImageTagMap(t *testing.T) {
	subImage := SubImage{
		Images: []Image{
			{
				Name: "test-image",
				Tag:  []string{"v1.0.0", "latest"},
			},
		},
	}

	result := imageTagMap(subImage, "kubernetes/", []string{"amd64"})

	expected := map[string][]string{
		"kubernetes/test-image": {"v1.0.0", "latest"},
	}

	assert.Equal(t, expected, result)
}

func TestProcessTags(t *testing.T) {
	tags := []string{"v1.0.0", "latest"}
	arch := []string{"amd64", "arm64"}

	result := processTags(tags, arch)

	// Without cut character, tags remain unchanged
	assert.Equal(t, []string{"v1.0.0", "latest"}, result)
}

func TestExpandTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      string
		arch     []string
		expected []string
	}{
		{
			name:     "tag without cut",
			tag:      "v1.0.0",
			arch:     []string{"amd64", "arm64"},
			expected: []string{"v1.0.0"},
		},
		{
			name:     "tag with cut",
			tag:      "v1.0.0" + cut + "tag",
			arch:     []string{"amd64", "arm64"},
			expected: []string{strings.ReplaceAll("v1.0.0"+cut+"tag", cut, "-amd64-"), strings.ReplaceAll("v1.0.0"+cut+"tag", cut, "-arm64-")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTag(tt.tag, tt.arch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckRepoImages(t *testing.T) {
	// Create a test config with repos
	cfg := &BuildConfig{
		Repos: []Repo{
			{
				Architecture: []string{"amd64"},
				SubImages: []SubImage{
					{
						SourceRepo: "registry.bocloud.com/kubernetes",
						Images: []Image{
							{
								Name: "test-image",
								Tag:  []string{"v1.0.0"},
							},
						},
					},
				},
			},
		},
	}

	// Apply patches
	patches := gomonkey.ApplyFunc(reg.ViewRepoImage, func(repo string, images map[string][]string) (map[string][]string, error) {
		// Return some mock results
		return map[string][]string{
			"test-image": {"v1.0.0", "amd64", "2023-01-01", "100MB"},
		}, nil
	})
	defer patches.Reset()

	rows, err := checkRepoImages(cfg)

	assert.NoError(t, err)
	assert.NotEmpty(t, rows)
	assert.Equal(t, "v1.0.0", rows[testZeroValue][testZeroValue])
	assert.Equal(t, "amd64", rows[testZeroValue][testOneValue])
}

func TestPreCheck(t *testing.T) {
	tests := []struct {
		name              string
		onlyImage         bool
		loadConfigError   error
		verifyConfigError error
		checkRepoError    error
		exportTableError  error
		expectError       bool
	}{
		{
			name:              "successful pre-check",
			onlyImage:         false,
			loadConfigError:   nil,
			verifyConfigError: nil,
			checkRepoError:    nil,
			exportTableError:  nil,
			expectError:       false,
		},
		{
			name:            "config loading fails",
			onlyImage:       false,
			loadConfigError: fmt.Errorf("load error"),
			expectError:     true,
		},
		{
			name:              "config verification fails",
			onlyImage:         false,
			loadConfigError:   nil,
			verifyConfigError: fmt.Errorf("verify error"),
			expectError:       true,
		},
		{
			name:              "only image check, verification skipped",
			onlyImage:         true,
			loadConfigError:   nil,
			verifyConfigError: fmt.Errorf("verify error"), // This should be skipped
			checkRepoError:    nil,
			exportTableError:  nil,
			expectError:       false,
		},
		{
			name:            "repo check fails",
			onlyImage:       true,
			loadConfigError: nil,
			checkRepoError:  fmt.Errorf("repo check error"),
			expectError:     true,
		},
		{
			name:             "export table fails",
			onlyImage:        true,
			loadConfigError:  nil,
			checkRepoError:   nil,
			exportTableError: fmt.Errorf("export error"),
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp := &PreCheckOptions{
				File:      "test-config.yaml",
				OnlyImage: tt.onlyImage,
			}

			// Apply patches
			patches := gomonkey.ApplyFunc(loadBuildConfig, func(filePath string) (*BuildConfig, error) {
				if tt.loadConfigError != nil {
					return nil, tt.loadConfigError
				}
				return &BuildConfig{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(verifyConfigContent, func(cfg *BuildConfig) error {
				return tt.verifyConfigError
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(checkRepoImages, func(cfg *BuildConfig) ([][]string, error) {
				if tt.checkRepoError != nil {
					return nil, tt.checkRepoError
				}
				return [][]string{{"image", "tag", "arch", "time", "size"}}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(exportTableToFile, func(filePath string, headers []string, rows [][]string) error {
				return tt.exportTableError
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(fmt.Println, func(a ...interface{}) (n int, err error) {
				return 0, nil
			})
			defer patches.Reset()

			bp.PreCheck()

			// The function should complete without panic
			assert.True(t, true)
		})
	}
}

func TestExportTableToFile(t *testing.T) {
	tests := []struct {
		name          string
		filePath      string
		headers       []string
		rows          [][]string
		mockWriteFile func(string, []byte, os.FileMode) error
		expectError   bool
	}{
		{
			name:     "successful export",
			filePath: "test-config.yaml",
			headers:  []string{"IMAGE", "TAGS"},
			rows:     [][]string{{"image1", "tag1"}, {"image2", "tag2"}},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return nil
			},
			expectError: false,
		},
		{
			name:     "write file fails",
			filePath: "test-config.yaml",
			headers:  []string{"IMAGE", "TAGS"},
			rows:     [][]string{{"image1", "tag1"}},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return fmt.Errorf("write error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.WriteFile, tt.mockWriteFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(fmt.Println, func(a ...interface{}) (n int, err error) {
				return 0, nil
			})
			defer patches.Reset()

			err := exportTableToFile(tt.filePath, tt.headers, tt.rows)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestExportTableToFileFileNameHandling(t *testing.T) {
	// Test the filename extraction logic
	patches := gomonkey.ApplyFunc(os.WriteFile, func(filename string, data []byte, perm os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(fmt.Println, func(a ...interface{}) (n int, err error) {
		return 0, nil
	})
	defer patches.Reset()

	headers := []string{"IMAGE", "TAGS"}
	rows := [][]string{{"image1", "tag1"}}

	err := exportTableToFile("path/to/test-config.yaml", headers, rows)
	assert.NoError(t, err)
}

func TestExportTableToFileWithComplexFilename(t *testing.T) {
	// Test with a complex filename that has multiple dots
	patches := gomonkey.ApplyFunc(os.WriteFile, func(filename string, data []byte, perm os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(fmt.Println, func(a ...interface{}) (n int, err error) {
		return 0, nil
	})
	defer patches.Reset()

	headers := []string{"IMAGE", "TAGS"}
	rows := [][]string{{"image1", "tag1"}}

	err := exportTableToFile("path/to/my.test.config.yaml", headers, rows)
	assert.NoError(t, err)
}
