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

package common

import (
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	econd "gopkg.openfuyao.cn/bkeadm/pkg/executor/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
)

func TestLoadLocalRepositoryFromFile(t *testing.T) {
	tests := []struct {
		name               string
		imageFilePath      string
		mockExists         func(string) bool
		mockIsDocker       func() bool
		mockIsContainerd   func() bool
		mockDockerLoad     func(*docker.Client, string) (string, error)
		mockContainerdLoad func(string) error
		expectError        bool
	}{
		{
			name:          "file exists, docker runtime, load successful",
			imageFilePath: "/path/to/image.tar",
			mockExists: func(path string) bool {
				return true
			},
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
			mockDockerLoad: func(c *docker.Client, path string) (string, error) {
				return "loaded successfully", nil
			},
			mockContainerdLoad: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "file exists, containerd runtime, load successful",
			imageFilePath: "/path/to/image.tar",
			mockExists: func(path string) bool {
				return true
			},
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockDockerLoad: func(c *docker.Client, path string) (string, error) {
				return "", nil
			},
			mockContainerdLoad: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "file exists, both runtimes active, both load successful",
			imageFilePath: "/path/to/image.tar",
			mockExists: func(path string) bool {
				return true
			},
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockDockerLoad: func(c *docker.Client, path string) (string, error) {
				return "docker loaded", nil
			},
			mockContainerdLoad: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "file exists, docker runtime, docker load fails",
			imageFilePath: "/path/to/image.tar",
			mockExists: func(path string) bool {
				return true
			},
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
			mockDockerLoad: func(c *docker.Client, path string) (string, error) {
				return "", fmt.Errorf("docker load error")
			},
			mockContainerdLoad: func(path string) error {
				return nil
			},
			expectError: true,
		},
		{
			name:          "file exists, containerd runtime, containerd load fails",
			imageFilePath: "/path/to/image.tar",
			mockExists: func(path string) bool {
				return true
			},
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockDockerLoad: func(c *docker.Client, path string) (string, error) {
				return "", nil
			},
			mockContainerdLoad: func(path string) error {
				return fmt.Errorf("containerd load error")
			},
			expectError: true,
		},
		{
			name:          "file does not exist",
			imageFilePath: "/path/to/nonexistent.tar",
			mockExists: func(path string) bool {
				return false
			},
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockDockerLoad: func(c *docker.Client, path string) (string, error) {
				return "", nil
			},
			mockContainerdLoad: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "file exists, neither runtime active",
			imageFilePath: "/path/to/image.tar",
			mockExists: func(path string) bool {
				return true
			},
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return false
			},
			mockDockerLoad: func(c *docker.Client, path string) (string, error) {
				return "", nil
			},
			mockContainerdLoad: func(path string) error {
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(utils.Exists, tt.mockExists)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc((*docker.Client).Load, tt.mockDockerLoad)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(econd.Load, tt.mockContainerdLoad)
			defer patches.Reset()

			if global.Docker == nil {
				global.Docker = &docker.Client{}
			}

			err := LoadLocalRepositoryFromFile(tt.imageFilePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadLocalRepositoryFromFileBothRuntimesFail(t *testing.T) {
	imageFilePath := "/path/to/image.tar"

	// Apply patches
	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*docker.Client).Load, func(c *docker.Client, path string) (string, error) {
		return "", fmt.Errorf("docker load error")
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(econd.Load, func(path string) error {
		return fmt.Errorf("containerd load error")
	})
	defer patches.Reset()

	err := LoadLocalRepositoryFromFile(imageFilePath)

	// Should return the first error (docker load error)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "docker load error")
}

func TestLoadLocalRepositoryFromFileWithEmptyPath(t *testing.T) {
	// Apply patches
	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return false // Empty path doesn't exist
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*docker.Client).Load, func(c *docker.Client, path string) (string, error) {
		return "", nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(econd.Load, func(path string) error {
		return nil
	})
	defer patches.Reset()

	err := LoadLocalRepositoryFromFile("")

	assert.NoError(t, err)
}

func TestLoadLocalRepositoryFromFileDockerOnly(t *testing.T) {
	imageFilePath := "/path/to/image.tar"

	// Apply patches
	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool {
		return false
	})
	defer patches.Reset()

	loadCalled := false
	patches = gomonkey.ApplyFunc((*docker.Client).Load, func(c *docker.Client, path string) (string, error) {
		loadCalled = true
		return "docker loaded", nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(econd.Load, func(path string) error {
		// This should not be called
		t.Error("Containerd load should not be called when only Docker is active")
		return nil
	})
	defer patches.Reset()

	err := LoadLocalRepositoryFromFile(imageFilePath)

	assert.NoError(t, err)
	assert.True(t, loadCalled)
}

func TestLoadLocalRepositoryFromFileContainerdOnly(t *testing.T) {
	imageFilePath := "/path/to/image.tar"

	// Apply patches
	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
		return false
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*docker.Client).Load, func(c *docker.Client, path string) (string, error) {
		// This should not be called
		t.Error("Docker load should not be called when only Containerd is active")
		return "", nil
	})
	defer patches.Reset()

	loadCalled := false
	patches = gomonkey.ApplyFunc(econd.Load, func(path string) error {
		loadCalled = true
		return nil
	})
	defer patches.Reset()

	err := LoadLocalRepositoryFromFile(imageFilePath)

	assert.NoError(t, err)
	assert.True(t, loadCalled)
}
