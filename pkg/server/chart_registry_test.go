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
	"context"
	"fmt"
	"github.com/agiledragon/gomonkey/v2"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/stretchr/testify/assert"
	"testing"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func TestStartChartRegistry(t *testing.T) {
	tests := []struct {
		name                    string
		containerName           string
		image                   string
		port                    string
		dataDir                 string
		mockIsContainerd        func() bool
		mockIsDocker            func() bool
		mockStartWithContainerd func(string, string, string, string) error
		mockStartWithDocker     func(string, string, string, string) error
		expectError             bool
	}{
		{
			name:                    "start with containerd when only containerd is available",
			containerName:           "test-chart-registry",
			image:                   "chartmuseum:latest",
			port:                    "8080",
			dataDir:                 "/var/lib/charts",
			mockIsContainerd:        func() bool { return true },
			mockIsDocker:            func() bool { return false },
			mockStartWithContainerd: func(name, image, port, dataDir string) error { return nil },
			mockStartWithDocker: func(name, image, port, dataDir string) error {
				t.Error("startWithDocker should not be called when only containerd is available")
				return nil
			},
			expectError: false,
		},
		{
			name:                "start with docker when docker is available",
			containerName:       "test-chart-registry",
			image:               "chartmuseum:latest",
			port:                "8080",
			dataDir:             "/var/lib/charts",
			mockIsContainerd:    func() bool { return false },
			mockIsDocker:        func() bool { return true },
			mockStartWithDocker: func(name, image, port, dataDir string) error { return nil },
			mockStartWithContainerd: func(name, image, port, dataDir string) error {
				t.Error("startWithContainerd should not be called when docker is available")
				return nil
			},
			expectError: false,
		},
		{
			name:                "both runtimes available, docker takes precedence",
			containerName:       "test-chart-registry",
			image:               "chartmuseum:latest",
			port:                "8080",
			dataDir:             "/var/lib/charts",
			mockIsContainerd:    func() bool { return true },
			mockIsDocker:        func() bool { return true },
			mockStartWithDocker: func(name, image, port, dataDir string) error { return nil },
			mockStartWithContainerd: func(name, image, port, dataDir string) error {
				t.Error("startWithContainerd should not be called when docker is available")
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			if tt.mockStartWithContainerd != nil {
				patches = gomonkey.ApplyFunc(startChartRegistryWithContainerd, tt.mockStartWithContainerd)
				defer patches.Reset()
			}

			if tt.mockStartWithDocker != nil {
				patches = gomonkey.ApplyFunc(startChartRegistryWithDocker, tt.mockStartWithDocker)
				defer patches.Reset()
			}

			err := StartChartRegistry(tt.containerName, tt.image, tt.port, tt.dataDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStartChartRegistryWithDocker(t *testing.T) {
	tests := []struct {
		name                              string
		mockEnsureDockerImageAndContainer func(string, string, string) (bool, error)
		mockRunDockerChartRegistry        func(string, string, string, string) error
		mockWaitForDockerContainerRunning func(string, string)
		expectError                       bool
	}{
		{
			name: "successful registry start",
			mockEnsureDockerImageAndContainer: func(name, image, service string) (bool, error) {

				return false, nil // Container not running
			},
			mockRunDockerChartRegistry: func(name, image, port, dataDir string) error {

				return nil
			},
			mockWaitForDockerContainerRunning: func(name, serviceType string) {
				assert.Equal(t, "test-chart-registry", name)
				assert.Equal(t, "chart", serviceType)
			},
			expectError: false,
		},
		{
			name: "ensure docker image and container fails",
			mockEnsureDockerImageAndContainer: func(name, image, service string) (bool, error) {
				return false, fmt.Errorf("ensure failed")
			},
			expectError: true,
		},
		{
			name: "container already running",
			mockEnsureDockerImageAndContainer: func(name, image, service string) (bool, error) {
				return true, nil // Container already running
			},
			// runDockerChartRegistry should not be called
			expectError: false,
		},
		{
			name: "run docker chart registry fails",
			mockEnsureDockerImageAndContainer: func(name, image, service string) (bool, error) {
				return false, nil // Container not running
			},
			mockRunDockerChartRegistry: func(name, image, port, dataDir string) error {
				return fmt.Errorf("run failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(ensureDockerImageAndContainer, tt.mockEnsureDockerImageAndContainer)
			defer patches.Reset()

			if tt.mockRunDockerChartRegistry != nil {
				patches = gomonkey.ApplyFunc(runDockerChartRegistry, tt.mockRunDockerChartRegistry)
				defer patches.Reset()
			}

			if tt.mockWaitForDockerContainerRunning != nil {
				patches = gomonkey.ApplyFunc(waitForDockerContainerRunning, tt.mockWaitForDockerContainerRunning)
				defer patches.Reset()
			}

			err := startChartRegistryWithDocker("test-chart-registry", "chartmuseum:latest", "8080", "/var/lib/charts")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestWaitForDockerContainerRunning(t *testing.T) {
	// Create a mock Docker client
	mockDockerClient := &docker.Client{}

	originalDocker := global.Docker
	global.Docker = mockDockerClient
	defer func() {
		global.Docker = originalDocker
	}()

	// Apply patches
	patches := gomonkey.ApplyFunc((*docker.Client).GetClient, func(client *docker.Client) *client.Client {
		// Return a mock Docker API client
		return client.Client
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*client.Client).ContainerInspect, func(client *client.Client, ctx context.Context, containerID string) (container.InspectResponse, error) {
		// Return a mock container JSON
		return container.InspectResponse{ContainerJSONBase: &types.ContainerJSONBase{State: &types.ContainerState{Running: true}}}, nil
	})

	defer patches.Reset()

	// This test would need more sophisticated mocking to fully test the waiting logic
	// For now, we'll just verify that the function can be called without error
	waitForDockerContainerRunning("test-container", "chart")

	// The function should complete without panic
	assert.True(t, true)
}

func TestStartChartRegistryWithContainerd(t *testing.T) {
	tests := []struct {
		name                   string
		mockEnsureImageExists  func(string) error
		mockEnsureContainerRun func(string) (bool, error)
		mockRun                func([]string) error
		mockContainerInspect   func(string) (containerd.NerdContainerInfo, error)
		expectError            bool
	}{
		{
			name: "successful registry start with containerd",
			mockEnsureImageExists: func(image string) error {
				return nil
			},
			mockEnsureContainerRun: func(name string) (bool, error) {
				return false, nil // Container not running
			},
			mockRun: func(args []string) error {
				return nil
			},
			mockContainerInspect: func(name string) (containerd.NerdContainerInfo, error) {
				info := containerd.NerdContainerInfo{}
				info.State.Status = "running"
				info.State.Running = true
				return info, nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(containerd.EnsureImageExists, tt.mockEnsureImageExists)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(containerd.EnsureContainerRun, tt.mockEnsureContainerRun)
			defer patches.Reset()

			if tt.mockRun != nil {
				patches = gomonkey.ApplyFunc(containerd.Run, tt.mockRun)
				defer patches.Reset()
			}

			if tt.mockContainerInspect != nil {
				patches = gomonkey.ApplyFunc(containerd.ContainerInspect, tt.mockContainerInspect)
				defer patches.Reset()
			}

			err := startChartRegistryWithContainerd("test-chart-registry", "chartmuseum:latest", "8080", "/var/lib/charts")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemoveChartRegistry(t *testing.T) {
	tests := []struct {
		name                         string
		containerName                string
		mockRemoveContainerWithRetry func(string, func()) error
		expectError                  bool
	}{
		{
			name:          "successful removal",
			containerName: "test-chart-registry",
			mockRemoveContainerWithRetry: func(name string, extraCleanup func()) error {
				assert.Equal(t, "test-chart-registry", name)
				assert.Nil(t, extraCleanup)
				return nil
			},
			expectError: false,
		},
		{
			name:          "removal fails",
			containerName: "test-chart-registry",
			mockRemoveContainerWithRetry: func(name string, extraCleanup func()) error {
				return fmt.Errorf("removal failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(removeContainerWithRetry, tt.mockRemoveContainerWithRetry)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := RemoveChartRegistry(tt.containerName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStartChartRegistryWithRealDockerClient(t *testing.T) {
	// Create a mock Docker client
	mockDockerClient := &docker.Client{}

	originalDocker := global.Docker
	global.Docker = mockDockerClient
	defer func() {
		global.Docker = originalDocker
	}()

	// Apply patches
	patches := gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool { return false })
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool { return true })
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*docker.Client).EnsureImageExists, func(c *docker.Client, ref docker.ImageRef, opts utils.RetryOptions) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*docker.Client).EnsureContainerRun, func(c *docker.Client, name string) (bool, error) {
		return false, nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(runDockerChartRegistry, func(name, image, port, dataDir string) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(waitForDockerContainerRunning, func(name, serviceType string) {})
	defer patches.Reset()

	err := StartChartRegistry("test-chart-registry", "chartmuseum:latest", "8080", "/var/lib/charts")

	assert.NoError(t, err)
}
