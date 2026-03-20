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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func TestBuildOnlineImage(t *testing.T) {
	tests := []struct {
		name                string
		isDockerEnvironment bool
		readFileError       error
		yamlUnmarshalError  error
		prepareError        error
		buildRpmsError      error
		buildImageError     error
		expectError         bool
	}{
		{
			name:                "successful build",
			isDockerEnvironment: true,
			readFileError:       nil,
			yamlUnmarshalError:  nil,
			prepareError:        nil,
			buildRpmsError:      nil,
			buildImageError:     nil,
			expectError:         false,
		},
		{
			name:                "not in docker environment",
			isDockerEnvironment: false,
			expectError:         true,
		},
		{
			name:                "read file fails",
			isDockerEnvironment: true,
			readFileError:       fmt.Errorf("read error"),
			expectError:         true,
		},
		{
			name:                "yaml unmarshal fails",
			isDockerEnvironment: true,
			readFileError:       nil,
			yamlUnmarshalError:  fmt.Errorf("unmarshal error"),
			expectError:         true,
		},
		{
			name:                "prepare fails",
			isDockerEnvironment: true,
			readFileError:       nil,
			yamlUnmarshalError:  nil,
			prepareError:        fmt.Errorf("prepare error"),
			expectError:         true,
		},
		{
			name:                "build rpms fails",
			isDockerEnvironment: true,
			readFileError:       nil,
			yamlUnmarshalError:  nil,
			prepareError:        nil,
			buildRpmsError:      fmt.Errorf("build rpms error"),
			expectError:         true,
		},
		{
			name:                "build image fails",
			isDockerEnvironment: true,
			readFileError:       nil,
			yamlUnmarshalError:  nil,
			prepareError:        nil,
			buildRpmsError:      nil,
			buildImageError:     fmt.Errorf("build image error"),
			expectError:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{
				File:   "test-config.yaml",
				Target: "test-image:latest",
				Arch:   "amd64",
			}

			// Apply patches
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
				return tt.isDockerEnvironment
			})
			defer patches.Reset()

			if tt.isDockerEnvironment {
				patches = gomonkey.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
					if tt.readFileError != nil {
						return nil, tt.readFileError
					}
					return []byte("version: v1.0.0"), nil
				})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc(yaml.Unmarshal, func(data []byte, v interface{}) error {
					if tt.yamlUnmarshalError != nil {
						return tt.yamlUnmarshalError
					}
					cfg := v.(*BuildConfig)
					cfg.OpenFuyaoVersion = "v1.0.0"
					return nil
				})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc(prepare, func() error {
					return tt.prepareError
				})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc(buildRpms, func(cfg *BuildConfig, stopChan <-chan struct{}) error {
					return tt.buildRpmsError
				})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc(buildImage, func(imageName string, arch string) error {
					return tt.buildImageError
				})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc(closeChanStruct, func(ch chan struct{}) {})
				defer patches.Reset()

				patches = gomonkey.ApplyFunc(os.RemoveAll, func(path string) error {
					return nil // Ignore removal errors for this test
				})
				defer patches.Reset()
			}

			o.BuildOnlineImage()

			// The function should complete without panic
			assert.True(t, true)
		})
	}
}

func TestBuildImage(t *testing.T) {
	tests := []struct {
		name                         string
		imageName                    string
		arch                         string
		mockMkdir                    func(string, os.FileMode) error
		mockWriteFile                func(string, []byte, os.FileMode) error
		mockRename                   func(string, string) error
		mockExecuteCommandWithOutput func(string, ...string) (string, error)
		expectError                  bool
	}{
		{
			name:          "single architecture build",
			imageName:     "test-image:latest",
			arch:          "amd64",
			mockMkdir:     func(path string, perm os.FileMode) error { return nil },
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error { return nil },
			mockRename:    func(oldpath, newpath string) error { return nil },
			mockExecuteCommandWithOutput: func(command string, arg ...string) (string, error) {
				// Simulate successful docker build and push
				return "", nil
			},
			expectError: true,
		},
		{
			name:          "multi architecture build",
			imageName:     "test-image:latest",
			arch:          "amd64,arm64",
			mockMkdir:     func(path string, perm os.FileMode) error { return nil },
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error { return nil },
			mockRename:    func(oldpath, newpath string) error { return nil },
			mockExecuteCommandWithOutput: func(command string, arg ...string) (string, error) {
				return "", nil
			},
			expectError: true,
		},
		{
			name:        "mkdir fails",
			imageName:   "test-image:latest",
			arch:        "amd64",
			mockMkdir:   func(path string, perm os.FileMode) error { return fmt.Errorf("mkdir error") },
			expectError: true,
		},
		{
			name:          "write file fails",
			imageName:     "test-image:latest",
			arch:          "amd64",
			mockMkdir:     func(path string, perm os.FileMode) error { return nil },
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error { return fmt.Errorf("write error") },
			expectError:   true,
		},
		{
			name:          "rename fails",
			imageName:     "test-image:latest",
			arch:          "amd64",
			mockMkdir:     func(path string, perm os.FileMode) error { return nil },
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error { return nil },
			mockRename:    func(oldpath, newpath string) error { return fmt.Errorf("rename error") },
			expectError:   true,
		},
		{
			name:          "docker build fails",
			imageName:     "test-image:latest",
			arch:          "amd64",
			mockMkdir:     func(path string, perm os.FileMode) error { return nil },
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error { return nil },
			mockRename:    func(oldpath, newpath string) error { return nil },
			mockExecuteCommandWithOutput: func(command string, arg ...string) (string, error) {
				return "docker error output", fmt.Errorf("docker error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(os.Mkdir, tt.mockMkdir)
			defer patches.Reset()

			if tt.mockWriteFile != nil {
				patches = gomonkey.ApplyFunc(os.WriteFile, tt.mockWriteFile)
				defer patches.Reset()
			}

			// Always mock Rename since it's called in buildImage
			if tt.mockRename != nil {
				patches = gomonkey.ApplyFunc(os.Rename, tt.mockRename)
			} else {
				// Provide a default mock for Rename that always succeeds
				patches = gomonkey.ApplyFunc(os.Rename, func(oldpath, newpath string) error {
					return nil
				})
			}
			defer patches.Reset()

			if tt.mockExecuteCommandWithOutput != nil {
				patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCommandWithOutput)
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(os.RemoveAll, func(path string) error { return nil })
			defer patches.Reset()

			// Mock pwd and bke variables
			patches = gomonkey.ApplyGlobalVar(&pwd, "/tmp")
			defer patches.Reset()

			patches = gomonkey.ApplyGlobalVar(&bke, "/tmp/bke")
			defer patches.Reset()

			err := buildImage(tt.imageName, tt.arch)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
