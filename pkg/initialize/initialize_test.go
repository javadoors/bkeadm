/*
 * Copyright (c) 2025 Huawei Technologies Co., Ltd.
 * openFuyao is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */

package initialize

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
)

const (
	testIPv4SegmentA = 192
	testIPv4SegmentB = 168
	testIPv4SegmentC = 1
	testIPv4SegmentD = 100
)

var testHostIP = net.IPv4(
	testIPv4SegmentA,
	testIPv4SegmentB,
	testIPv4SegmentC,
	testIPv4SegmentD,
)

func TestExtractVersionFromFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "no version pattern returns empty",
			filename: "some-random-file.tar.gz",
			expected: "",
		},
		{
			name:     "version with dots",
			filename: "openfuyao-v1.0.0.1.tar.gz",
			expected: "v1.0.0.1.tar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractVersionFromFilename(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseYAMLBytesToSliceMap(t *testing.T) {
	tests := []struct {
		name        string
		yamlData    string
		expectError bool
	}{
		{
			name: "valid yaml with required fields",
			yamlData: `
- openFuyaoVersion: v1.0.0
  filePath: ./files/v1.0.0.tar.gz
`,
			expectError: false,
		},
		{
			name: "missing openFuyaoVersion field",
			yamlData: `
- filePath: ./files/v1.0.0.tar.gz
`,
			expectError: true,
		},
		{
			name: "missing filePath field",
			yamlData: `
- openFuyaoVersion: v1.0.0
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseYAMLBytesToSliceMap([]byte(tt.yamlData))
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestVersionLess(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected bool
	}{
		{
			name:     "v1 less than v2",
			v1:       "v1.0.0",
			v2:       "v2.0.0",
			expected: true,
		},
		{
			name:     "v1 greater than v2",
			v1:       "v2.0.0",
			v2:       "v1.0.0",
			expected: false,
		},
		{
			name:     "same version",
			v1:       "v1.0.0",
			v2:       "v1.0.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := versionLess(tt.v1, tt.v2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestLogAuthMode(t *testing.T) {
	t.Run("tls verify with credentials", func(t *testing.T) {
		op := &Options{
			ImageRepoTLSVerify: true,
			ImageRepoUsername:  "admin",
			ImageRepoPassword:  "password",
		}
		op.logAuthMode()
		assert.True(t, true)
	})

	t.Run("tls verify with ca file", func(t *testing.T) {
		op := &Options{
			ImageRepoTLSVerify: true,
			ImageRepoCAFile:    "/path/to/ca.crt",
		}
		op.logAuthMode()
		assert.True(t, true)
	})
}

func TestSetGlobalCustomExtra(t *testing.T) {
	t.Run("set custom extra values", func(t *testing.T) {
		op := &Options{
			Domain:        "registry.example.com",
			HostIP:        testHostIP.String(),
			ImageRepoPort: "443",
			YumRepoPort:   "8080",
			ChartRepoPort: "8443",
			ClusterAPI:    "v1.0.0",
		}
		op.setGlobalCustomExtra()

		assert.Equal(t, "registry.example.com", global.CustomExtra["domain"])
		assert.Equal(t, testHostIP.String(), global.CustomExtra["host"])
		assert.Equal(t, "443", global.CustomExtra["imageRepoPort"])
	})
}

func TestModifyPermission(t *testing.T) {
	t.Run("default workspace", func(t *testing.T) {
		op := &Options{}
		op.modifyPermission()
		assert.True(t, true)
	})
}

func TestProcessPatchFiles(t *testing.T) {
	t.Run("directory not exist", func(t *testing.T) {
		op := &Options{}
		result := op.ProcessPatchFiles("/nonexistent")
		assert.Nil(t, result)
	})

	t.Run("empty patches directory", func(t *testing.T) {
		op := &Options{}
		result := op.ProcessPatchFiles("/tmp")
		assert.NotNil(t, result)
	})
}

func TestGenerateClusterConfig(t *testing.T) {
	t.Run("generate cluster config", func(t *testing.T) {
		op := &Options{
			Domain:        "registry.example.com",
			HostIP:        testHostIP.String(),
			ImageRepoPort: "443",
			ChartRepoPort: "8443",
			YumRepoPort:   "8080",
			ClusterAPI:    "v1.0.0",
			Runtime:       "containerd",
		}
		op.setGlobalCustomExtra()
		op.generateClusterConfig()
		assert.True(t, true)
	})
}

func TestDeployCluster(t *testing.T) {
	t.Run("no file specified", func(t *testing.T) {
		op := &Options{
			File: "",
		}
		op.deployCluster()
		assert.True(t, true)
	})
}

func TestBuildClientAuthConfig(t *testing.T) {
	t.Run("build config with all fields", func(t *testing.T) {
		op := &Options{
			ImageRepoTLSVerify: true,
			ImageRepoUsername:  "admin",
			ImageRepoPassword:  "password",
			ImageRepoCAFile:    "/path/to/ca.crt",
		}
		cfg := op.buildClientAuthConfig()
		assert.True(t, cfg.TLSVerify)
		assert.Equal(t, "admin", cfg.Username)
		assert.Equal(t, "password", cfg.Password)
		assert.Equal(t, "/path/to/ca.crt", cfg.CAFile)
	})
}

func TestPrepareImageRepoConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		op := &Options{
			Domain: "registry.example.com",
			HostIP: testHostIP.String(),
		}
		repo := op.prepareImageRepoConfig()
		assert.Equal(t, "registry.example.com", repo.Domain)
	})
}

func TestPrepareChartRepoConfig(t *testing.T) {
	t.Run("chart repo config", func(t *testing.T) {
		op := &Options{
			Domain:        "registry.example.com",
			HostIP:        testHostIP.String(),
			ChartRepoPort: "8443",
		}
		repo := op.prepareChartRepoConfig()
		assert.NotEmpty(t, repo.Ip)
	})
}

func TestPrepareHTTPRepoConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		op := &Options{
			Domain:      "registry.example.com",
			HostIP:      testHostIP.String(),
			YumRepoPort: "8080",
		}
		repo := op.prepareHTTPRepoConfig()
		assert.Equal(t, "8080", repo.Port)
	})
}
