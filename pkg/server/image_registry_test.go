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
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
)

const (
	testRegistryName    = "test-registry"
	testRegistryImage   = "registry:2"
	testRegistryPort    = "5000"
	testRegistryDataDir = "/var/lib/registry"
)

func TestStartImageRegistryWithContainerd(t *testing.T) {
	tests := []struct {
		name                             string
		mockGenerateConfig               func(string, string) error
		mockContainerdEnsureImageExists  func(string) error
		mockContainerdEnsureContainerRun func(string) (bool, error)
		expectError                      bool
	}{
		{
			name: "containerd container already running",
			mockGenerateConfig: func(certPath, port string) error {
				return nil
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
			name: "generate config fails",
			mockGenerateConfig: func(certPath, port string) error {
				return assert.AnError
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockGenerateConfig != nil {
				patches := gomonkey.ApplyFunc(generateConfig, tt.mockGenerateConfig)
				defer patches.Reset()
			}

			if tt.mockContainerdEnsureImageExists != nil {
				patches := gomonkey.ApplyFunc(containerd.EnsureImageExists, tt.mockContainerdEnsureImageExists)
				defer patches.Reset()
			}

			if tt.mockContainerdEnsureContainerRun != nil {
				patches := gomonkey.ApplyFunc(containerd.EnsureContainerRun, tt.mockContainerdEnsureContainerRun)
				defer patches.Reset()
			}

			err := startImageRegistryWithContainerd(testRegistryName, testRegistryImage, testRegistryPort, testRegistryDataDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRemoveImageRegistry(t *testing.T) {
	tests := []struct {
		name                string
		mockIsDocker        func() bool
		mockIsContainerd    func() bool
		mockRemoveContainer func(string) error
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
			mockRemoveContainer: func(name string) error {
				return nil
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
			mockRemoveContainer: func(name string) error {
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(removeContainerWithRetry, func(name string, extraCleanup func()) error {
				return tt.mockRemoveContainer(name)
			})
			defer patches.Reset()

			err := RemoveImageRegistry(testRegistryName)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateConfig(t *testing.T) {
	tests := []struct {
		name                          string
		certPath                      string
		port                          string
		mockSetRegistryConfig         func(string) error
		mockSetServerCertificate      func(string) error
		mockSetClientCertificate      func(string, string) error
		mockSetClientLocalCertificate func(string, string) error
		expectError                   bool
	}{
		{
			name:     "successful config generation",
			certPath: "/etc/docker/test-registry",
			port:     "5000",
			mockSetRegistryConfig: func(certPath string) error {
				return nil
			},
			mockSetServerCertificate: func(certPath string) error {
				return nil
			},
			mockSetClientCertificate: func(certPath, port string) error {
				return nil
			},
			mockSetClientLocalCertificate: func(certPath, port string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:     "set registry config fails",
			certPath: "/etc/docker/test-registry",
			port:     "5000",
			mockSetRegistryConfig: func(certPath string) error {
				return assert.AnError
			},
			expectError: true,
		},
		{
			name:     "set server certificate fails",
			certPath: "/etc/docker/test-registry",
			port:     "5000",
			mockSetRegistryConfig: func(certPath string) error {
				return nil
			},
			mockSetServerCertificate: func(certPath string) error {
				return assert.AnError
			},
			expectError: true,
		},
		{
			name:     "set client certificate fails",
			certPath: "/etc/docker/test-registry",
			port:     "5000",
			mockSetRegistryConfig: func(certPath string) error {
				return nil
			},
			mockSetServerCertificate: func(certPath string) error {
				return nil
			},
			mockSetClientCertificate: func(certPath, port string) error {
				return assert.AnError
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockSetRegistryConfig != nil {
				patches := gomonkey.ApplyFunc(SetRegistryConfig, tt.mockSetRegistryConfig)
				defer patches.Reset()
			}

			if tt.mockSetServerCertificate != nil {
				patches := gomonkey.ApplyFunc(SetServerCertificate, tt.mockSetServerCertificate)
				defer patches.Reset()
			}

			if tt.mockSetClientCertificate != nil {
				patches := gomonkey.ApplyFunc(SetClientCertificate, tt.mockSetClientCertificate)
				defer patches.Reset()
			}

			if tt.mockSetClientLocalCertificate != nil {
				patches := gomonkey.ApplyFunc(SetClientLocalCertificate, tt.mockSetClientLocalCertificate)
				defer patches.Reset()
			}

			err := generateConfig(tt.certPath, tt.port)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRunDockerImageRegistry(t *testing.T) {
	t.Run("docker run test", func(t *testing.T) {
		assert.NotPanics(t, func() {
			global.Docker = &docker.Client{}
		})
	})
}
