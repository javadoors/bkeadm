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

package server

import (
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
)

const (
	testYumName     = "test-yum"
	testYumImage    = "yum-repo:latest"
	testYumPort     = "8080"
	testYumDataDir  = "/var/yum"
	testServiceName = "yum warehouse"
)

func TestStartYumRegistryWithDocker(t *testing.T) {
	tests := []struct {
		name                        string
		mockIsDocker                func() bool
		mockIsContainerd            func() bool
		mockEnsureImageAndContainer func(name, image, serviceName string) (bool, error)
		mockRunDockerYumRegistry    func(string, string, string, string) error
		expectError                 bool
	}{
		{
			name: "successful start with docker",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
			mockEnsureImageAndContainer: func(name, image, serviceName string) (bool, error) {
				return false, nil
			},
			mockRunDockerYumRegistry: func(name, image, yumRegistryPort, yumDataDirectory string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "yum already running",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
			mockEnsureImageAndContainer: func(name, image, serviceName string) (bool, error) {
				return true, nil
			},
			expectError: false,
		},
		{
			name: "ensure image and container fails",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
			mockEnsureImageAndContainer: func(name, image, serviceName string) (bool, error) {
				return false, assert.AnError
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			if tt.mockEnsureImageAndContainer != nil {
				patches = gomonkey.ApplyFunc(ensureDockerImageAndContainer, tt.mockEnsureImageAndContainer)
				defer patches.Reset()
			}

			if tt.mockRunDockerYumRegistry != nil {
				patches = gomonkey.ApplyFunc(runDockerYumRegistry, tt.mockRunDockerYumRegistry)
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(waitForDockerContainerRunning, func(name, serviceType string) {})
			defer patches.Reset()

			err := StartYumRegistry(testYumName, testYumImage, testYumPort, testYumDataDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStartYumRegistryWithContainerd(t *testing.T) {
	tests := []struct {
		name                             string
		mockIsDocker                     func() bool
		mockIsContainerd                 func() bool
		mockContainerdEnsureImageExists  func(string) error
		mockContainerdEnsureContainerRun func(string) (bool, error)
		mockContainerdRun                func([]string) error
		mockContainerdInspect            func(string) (containerd.NerdContainerInfo, error)
		expectError                      bool
	}{
		{
			name: "successful start with containerd",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockContainerdEnsureImageExists: func(image string) error {
				return nil
			},
			mockContainerdEnsureContainerRun: func(name string) (bool, error) {
				return false, nil
			},
			mockContainerdRun: func(script []string) error {
				return nil
			},
			mockContainerdInspect: func(name string) (containerd.NerdContainerInfo, error) {
				return containerd.NerdContainerInfo{
					State: struct {
						Status     string `json:"Status"`
						Running    bool   `json:"Running"`
						Paused     bool   `json:"Paused"`
						Restarting bool   `json:"Restarting"`
						Pid        uint   `json:"Pid"`
						ExitCode   uint   `json:"ExitCode"`
						FinishedAt string `json:"FinishedAt"`
					}{
						Running: true,
					},
				}, nil
			},
			expectError: false,
		},
		{
			name: "containerd container already running",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockContainerdEnsureImageExists: func(image string) error {
				return nil
			},
			mockContainerdEnsureContainerRun: func(name string) (bool, error) {
				return true, nil
			},
			expectError: false,
		},
		{
			name: "containerd ensure image exists fails",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockContainerdEnsureImageExists: func(image string) error {
				return assert.AnError
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			if tt.mockContainerdEnsureImageExists != nil {
				patches = gomonkey.ApplyFunc(containerd.EnsureImageExists, tt.mockContainerdEnsureImageExists)
				defer patches.Reset()
			}

			if tt.mockContainerdEnsureContainerRun != nil {
				patches = gomonkey.ApplyFunc(containerd.EnsureContainerRun, tt.mockContainerdEnsureContainerRun)
				defer patches.Reset()
			}

			if tt.mockContainerdRun != nil {
				patches = gomonkey.ApplyFunc(containerd.Run, tt.mockContainerdRun)
				defer patches.Reset()
			}

			if tt.mockContainerdInspect != nil {
				patches = gomonkey.ApplyFunc(containerd.ContainerInspect, tt.mockContainerdInspect)
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(utils.FileExists, func(string) bool {
				return true
			})
			defer patches.Reset()

			err := StartYumRegistry(testYumName, testYumImage, testYumPort, testYumDataDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStartYumRegistryNoRuntime(t *testing.T) {
	patches := gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
		return false
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool {
		return false
	})
	defer patches.Reset()

	err := StartYumRegistry(testYumName, testYumImage, testYumPort, testYumDataDir)

	assert.NoError(t, err)
}

func TestRemoveYumRegistry(t *testing.T) {
	tests := []struct {
		name                string
		mockIsDocker        func() bool
		mockIsContainerd    func() bool
		mockRemoveContainer func(*docker.Client, string) error
		mockContainerExists func(*docker.Client, string) (types.ContainerJSON, bool)
		expectError         bool
	}{
		{
			name: "successful removal with docker",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
			mockRemoveContainer: func(c *docker.Client, name string) error {
				return nil
			},
			mockContainerExists: func(c *docker.Client, name string) (types.ContainerJSON, bool) {
				return types.ContainerJSON{}, false
			},
			expectError: false,
		},
		{
			name: "successful removal with containerd",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockRemoveContainer: func(c *docker.Client, name string) error {
				return nil
			},
			mockContainerExists: func(c *docker.Client, name string) (types.ContainerJSON, bool) {
				return types.ContainerJSON{}, false
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalDocker := global.Docker
			global.Docker = &docker.Client{}
			defer func() {
				global.Docker = originalDocker
			}()

			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			if tt.mockRemoveContainer != nil {
				patches = gomonkey.ApplyFunc((*docker.Client).ContainerRemove, tt.mockRemoveContainer)
				defer patches.Reset()
			}

			if tt.mockContainerExists != nil {
				patches = gomonkey.ApplyFunc((*docker.Client).ContainerExists, tt.mockContainerExists)
				defer patches.Reset()
			}

			err := RemoveYumRegistry(testYumName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunDockerYumRegistry(t *testing.T) {
	tests := []struct {
		name        string
		mockRun     func(*docker.Client, *container.Config, *container.HostConfig, *network.NetworkingConfig, *specs.Platform, string) error
		expectError bool
	}{
		{
			name: "successful docker run",
			mockRun: func(c *docker.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, name string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "docker run fails",
			mockRun: func(c *docker.Client, config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, name string) error {
				return assert.AnError
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalDocker := global.Docker
			global.Docker = &docker.Client{}
			defer func() {
				global.Docker = originalDocker
			}()

			patches := gomonkey.ApplyFunc((*docker.Client).Run, tt.mockRun)
			defer patches.Reset()

			err := runDockerYumRegistry(testYumName, testYumImage, testYumPort, testYumDataDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStartYumRegistryWithContainerdNotRunning(t *testing.T) {
	inspectCallCount := 0

	patches := gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
		return false
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool {
		return true
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(containerd.EnsureImageExists, func(image string) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(containerd.EnsureContainerRun, func(name string) (bool, error) {
		return false, nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(containerd.Run, func(script []string) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(containerd.ContainerInspect, func(name string) (containerd.NerdContainerInfo, error) {
		inspectCallCount++
		if inspectCallCount < 2 {
			return containerd.NerdContainerInfo{
				State: struct {
					Status     string `json:"Status"`
					Running    bool   `json:"Running"`
					Paused     bool   `json:"Paused"`
					Restarting bool   `json:"Restarting"`
					Pid        uint   `json:"Pid"`
					ExitCode   uint   `json:"ExitCode"`
					FinishedAt string `json:"FinishedAt"`
				}{
					Running: false,
				},
			}, nil
		}
		return containerd.NerdContainerInfo{
			State: struct {
				Status     string `json:"Status"`
				Running    bool   `json:"Running"`
				Paused     bool   `json:"Paused"`
				Restarting bool   `json:"Restarting"`
				Pid        uint   `json:"Pid"`
				ExitCode   uint   `json:"ExitCode"`
				FinishedAt string `json:"FinishedAt"`
			}{
				Running: true,
			},
		}, nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(utils.FileExists, func(string) bool {
		return true
	})
	defer patches.Reset()

	err := StartYumRegistry(testYumName, testYumImage, testYumPort, testYumDataDir)

	assert.NoError(t, err)
}
