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

package config

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
	configv1beta1 "gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"
	confv1beta1 "gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"
	"gopkg.openfuyao.cn/cluster-api-provider-bke/common/security"
	yaml2 "sigs.k8s.io/yaml"
)

const (
	testNumericZero   = 0
	testNumericOne    = 1
	testNumericTwo    = 2
	testNumericThree  = 3
	testNumericFour   = 4
	testDefaultPort   = "5000"
	testDefaultDomain = "test.domain.com"

	testIPv4SegmentA = 192
	testIPv4SegmentB = 168
	testIPv4SegmentC = 1
	testIPv4SegmentD = 1
)

func TestGenerateControllerParam(t *testing.T) {
	tests := []struct {
		name             string
		domain           string
		customExtraValue string
		wantSandbox      string
		wantOffline      string
	}{
		{
			name:             "normal domain without custom extra",
			domain:           "registry.example.com",
			customExtraValue: "",
			wantOffline:      "true",
		},
		{
			name:             "domain with custom other repo containing domain",
			domain:           "registry.example.com",
			customExtraValue: "custom.repo.com/image",
			wantOffline:      "false",
		},
		{
			name:             "domain with custom other repo not containing domain",
			domain:           "registry.example.com",
			customExtraValue: "other.repo.com/image:name",
			wantOffline:      "false",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalCustomExtra := global.CustomExtra["otherRepo"]
			global.CustomExtra["otherRepo"] = tt.customExtraValue
			defer func() {
				global.CustomExtra["otherRepo"] = originalCustomExtra
			}()

			sandbox, offline := GenerateControllerParam(tt.domain)

			assert.NotEmpty(t, sandbox)
			assert.Equal(t, tt.wantOffline, offline)
		})
	}
}

func TestGenerateControllerParamWithDifferentDomains(t *testing.T) {
	tests := []struct {
		name   string
		domain string
	}{
		{
			name:   "standard registry domain",
			domain: "registry.bocloud.com",
		},
		{
			name:   "IP address with port",
			domain: fmt.Sprintf("%d.%d.%d.%d:5000", testIPv4SegmentA, testIPv4SegmentB, testIPv4SegmentC, testIPv4SegmentD),
		},
		{
			name:   "localhost with port",
			domain: "localhost:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalCustomExtra := global.CustomExtra["otherRepo"]
			global.CustomExtra["otherRepo"] = ""
			defer func() {
				global.CustomExtra["otherRepo"] = originalCustomExtra
			}()

			sandbox, offline := GenerateControllerParam(tt.domain)

			assert.Contains(t, sandbox, tt.domain)
			assert.Equal(t, "true", offline)
		})
	}
}

func TestOptionsEnsureDirectory(t *testing.T) {
	tests := []struct {
		name         string
		directory    string
		mockExists   func(string) bool
		mockMkdirAll func(string, os.FileMode) error
		expectError  bool
	}{
		{
			name:      "directory already exists",
			directory: "/existing/dir",
			mockExists: func(path string) bool {
				return true
			},
			expectError: false,
		},
		{
			name:      "directory creation succeeds",
			directory: "/new/dir",
			mockExists: func(path string) bool {
				return false
			},
			mockMkdirAll: func(path string, perm os.FileMode) error {
				return nil
			},
			expectError: false,
		},
		{
			name:      "directory creation fails",
			directory: "/fail/dir",
			mockExists: func(path string) bool {
				return false
			},
			mockMkdirAll: func(path string, perm os.FileMode) error {
				return os.ErrPermission
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.Exists, tt.mockExists)
			defer patches.Reset()

			if tt.mockMkdirAll != nil {
				patches = gomonkey.ApplyFunc(os.MkdirAll, tt.mockMkdirAll)
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			opts := &Options{Directory: tt.directory}
			err := opts.ensureDirectory()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptionsCreateBKECluster(t *testing.T) {
	opts := &Options{}
	cluster := opts.createBKECluster()

	assert.Equal(t, "bke-cluster", cluster.Name)
	assert.Equal(t, "bke-cluster", cluster.Namespace)
	assert.Equal(t, "bke.bocloud.com/v1beta1", cluster.APIVersion)
	assert.Equal(t, "BKECluster", cluster.Kind)
	assert.NotNil(t, cluster.Spec.KubeletConfigRef)
	assert.Equal(t, "bke-kubelet", cluster.Spec.KubeletConfigRef.Name)
	assert.Equal(t, "bke-kubelet", cluster.Spec.KubeletConfigRef.Namespace)
}

func TestOptionsApplyCustomConfig(t *testing.T) {
	tests := []struct {
		name        string
		customExtra map[string]string
		imageRepo   confv1beta1.Repo
		yumRepo     confv1beta1.Repo
		chartRepo   confv1beta1.Repo
		ntpServer   string
		checkFunc   func(*testing.T, *confv1beta1.BKEConfig)
	}{
		{
			name:        "empty configs should not modify",
			customExtra: nil,
			imageRepo:   confv1beta1.Repo{},
			yumRepo:     confv1beta1.Repo{},
			chartRepo:   confv1beta1.Repo{},
			ntpServer:   "",
			checkFunc: func(t *testing.T, cfg *confv1beta1.BKEConfig) {
				assert.Nil(t, cfg.CustomExtra)
			},
		},
		{
			name:        "with custom extra",
			customExtra: map[string]string{"key": "value"},
			imageRepo:   confv1beta1.Repo{},
			yumRepo:     confv1beta1.Repo{},
			chartRepo:   confv1beta1.Repo{},
			ntpServer:   "",
			checkFunc: func(t *testing.T, cfg *confv1beta1.BKEConfig) {
				assert.NotNil(t, cfg.CustomExtra)
				assert.Equal(t, "value", cfg.CustomExtra["key"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &confv1beta1.BKEConfig{}
			opts := &Options{}

			opts.applyCustomConfig(cfg, tt.customExtra, tt.imageRepo, tt.yumRepo, tt.chartRepo, tt.ntpServer)

			tt.checkFunc(t, cfg)
		})
	}
}

func TestOptionsOptimizeKubeClient(t *testing.T) {
	opts := &Options{}
	cfg := &confv1beta1.BKEConfig{}

	opts.optimizeKubeClient(cfg)

	assert.NotNil(t, cfg.Cluster.APIServer)
	assert.NotNil(t, cfg.Cluster.ControllerManager)
	assert.NotNil(t, cfg.Cluster.Scheduler)
	assert.NotNil(t, cfg.Cluster.Kubelet)

	assert.Contains(t, cfg.Cluster.APIServer.ExtraArgs, "max-mutating-requests-inflight")
	assert.Contains(t, cfg.Cluster.ControllerManager.ExtraArgs, "kube-api-qps")
	assert.Contains(t, cfg.Cluster.Scheduler.ExtraArgs, "kube-api-qps")
	assert.Contains(t, cfg.Cluster.Kubelet.ExtraArgs, "kube-api-qps")
}

func TestUpdateCorednsAntiAffinity(t *testing.T) {
	tests := []struct {
		name                 string
		nodeCount            int
		expectedAntiAffinity string
	}{
		{
			name:                 "single node",
			nodeCount:            1,
			expectedAntiAffinity: "false",
		},
		{
			name:                 "multiple nodes",
			nodeCount:            2,
			expectedAntiAffinity: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &confv1beta1.BKEConfig{
				Addons: []confv1beta1.Product{
					{
						Name:    "coredns",
						Version: "v1.10.1",
						Param:   nil,
					},
				},
			}

			updateCorednsAntiAffinityByCount(cfg, tt.nodeCount)

			assert.Equal(t, tt.expectedAntiAffinity, cfg.Addons[testNumericZero].Param["EnableAntiAffinity"])
		})
	}
}

func TestUpdateCorednsAntiAffinityWithoutCoredns(t *testing.T) {
	cfg := &confv1beta1.BKEConfig{
		Addons: []confv1beta1.Product{
			{
				Name:    "other-addon",
				Version: "v1.0.0",
			},
		},
	}

	updateCorednsAntiAffinityByCount(cfg, 2)

	assert.Nil(t, cfg.Addons[testNumericZero].Param)
}

func TestOptionsApplyProductSpecificConfig(t *testing.T) {
	tests := []struct {
		name        string
		product     string
		expectPanic bool
		checkFunc   func(*testing.T, *confv1beta1.BKEConfig)
	}{
		{
			name:    "fuyao-portal product",
			product: "fuyao-portal",
			checkFunc: func(t *testing.T, cfg *confv1beta1.BKEConfig) {
				assert.GreaterOrEqual(t, len(cfg.Addons), testNumericTwo)
			},
		},
		{
			name:    "fuyao-business product",
			product: "fuyao-business",
			checkFunc: func(t *testing.T, cfg *confv1beta1.BKEConfig) {
				assert.NotNil(t, cfg)
			},
		},
		{
			name:    "fuyao-allinone product",
			product: "fuyao-allinone",
			checkFunc: func(t *testing.T, cfg *confv1beta1.BKEConfig) {
				assert.GreaterOrEqual(t, len(cfg.Addons), testNumericTwo)
			},
		},
		{
			name:    "unsupported product",
			product: "unknown-product",
			checkFunc: func(t *testing.T, cfg *confv1beta1.BKEConfig) {
				assert.NotNil(t, cfg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			cfg := &confv1beta1.BKEConfig{}
			opts := &Options{Product: tt.product}

			opts.applyProductSpecificConfig(cfg, "sandbox-image", "false")

			tt.checkFunc(t, cfg)
		})
	}
}

func TestOptionsSetBaseAddons(t *testing.T) {
	opts := &Options{}
	cfg := &confv1beta1.BKEConfig{}

	opts.setBaseAddons(cfg)

	assert.NotEmpty(t, cfg.Addons)
	assert.Equal(t, testNumericFour, len(cfg.Addons))

	var addonNames []string
	for _, addon := range cfg.Addons {
		addonNames = append(addonNames, addon.Name)
	}
	assert.Contains(t, addonNames, "kubeproxy")
	assert.Contains(t, addonNames, "calico")
	assert.Contains(t, addonNames, "coredns")
	assert.Contains(t, addonNames, "bkeagent-deployer")
}

func TestOptionsCreateClusterAPIAddon(t *testing.T) {
	opts := &Options{}
	addon := opts.createClusterAPIAddon("test-sandbox", "false")

	assert.Equal(t, "cluster-api", addon.Name)
	assert.Equal(t, "v1.4.3", addon.Version)
	assert.True(t, addon.Block)
	assert.Equal(t, "false", addon.Param["offline"])
	assert.Equal(t, "test-sandbox", addon.Param["sandbox"])
}

func TestOptionsCreateSystemControllerAddon(t *testing.T) {
	opts := &Options{}
	addon := opts.createSystemControllerAddon()

	assert.Equal(t, "openfuyao-system-controller", addon.Name)
	assert.Equal(t, "latest", addon.Version)
	assert.Contains(t, addon.Param["helmRepo"], "helm.openfuyao.cn")
}

func TestOptionsLogUnsupportedProduct(t *testing.T) {
	opts := &Options{Product: "unknown-product"}

	var capturedMsg string
	patches := gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {
		capturedMsg = msg
	})
	defer patches.Reset()

	opts.logUnsupportedProduct()

	assert.Contains(t, capturedMsg, "unknown-product")
}

func TestOptionsEncryptDecryptString(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		isEncrypt bool
	}{
		{
			name:      "encrypt single string",
			args:      []string{"password123"},
			isEncrypt: true,
		},
		{
			name:      "decrypt single string",
			args:      []string{"encrypted-password"},
			isEncrypt: false,
		},
		{
			name:      "encrypt multiple strings",
			args:      []string{"pass1", "pass2", "pass3"},
			isEncrypt: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(security.AesEncrypt, func(s string) (string, error) {
				return "encrypted-" + s, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(security.AesDecrypt, func(s string) (string, error) {
				if strings.HasPrefix(s, "encrypted-") {
					return strings.TrimPrefix(s, "encrypted-"), nil
				}
				return s, nil
			})
			defer patches.Reset()

			opts := &Options{Args: tt.args}

			if tt.isEncrypt {
				err := opts.EncryptString()
				assert.NoError(t, err)
			} else {
				err := opts.DecryptString()
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptionsEncryptDecryptStringWithError(t *testing.T) {
	patches := gomonkey.ApplyFunc(security.AesEncrypt, func(s string) (string, error) {
		return "", os.ErrPermission
	})
	defer patches.Reset()

	opts := &Options{Args: []string{"test"}}
	err := opts.EncryptString()

	assert.Error(t, err)
}

func TestOptionsDecryptStringWithError(t *testing.T) {
	patches := gomonkey.ApplyFunc(security.AesDecrypt, func(s string) (string, error) {
		return "", os.ErrPermission
	})
	defer patches.Reset()

	opts := &Options{Args: []string{"encrypted-test"}}
	err := opts.DecryptString()

	assert.Error(t, err)
}

func TestOptionsLoadClusterConfig(t *testing.T) {
	tests := []struct {
		name         string
		fileContent  string
		mockReadFile func(string) ([]byte, error)
		expectError  bool
	}{
		{
			name:        "valid yaml content",
			fileContent: "spec:\n  clusterConfig:\n    nodes: []",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("spec:\n  clusterConfig:\n    nodes: []"), nil
			},
			expectError: true,
		},
		{
			name:        "file read error",
			fileContent: "",
			mockReadFile: func(filename string) ([]byte, error) {
				return nil, os.ErrNotExist
			},
			expectError: true,
		},
		{
			name:        "invalid yaml",
			fileContent: "invalid: yaml: content: [",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("invalid: yaml: content: ["), nil
			},
			expectError: true,
		},
		{
			name:        "empty cluster config",
			fileContent: "spec: {}",
			mockReadFile: func(filename string) ([]byte, error) {
				return []byte("spec: {}"), nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadFile, tt.mockReadFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(yaml2.Unmarshal, func(data []byte, v interface{}) error {
				if strings.Contains(string(data), "invalid") {
					return os.ErrInvalid
				}
				return nil
			})
			defer patches.Reset()

			opts := &Options{File: "/test/config.yaml"}
			_, err := opts.loadClusterConfig()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestOptionsEncryptBKENodePassword(t *testing.T) {
	tests := []struct {
		name             string
		password         string
		mockDecryptError bool
		expectedPassword string
	}{
		{
			name:             "already encrypted",
			password:         "enc-pass",
			mockDecryptError: false,
			expectedPassword: "enc-pass",
		},
		{
			name:             "needs encryption",
			password:         "plain-pass",
			mockDecryptError: true,
			expectedPassword: "enc-plain-pass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.mockDecryptError {
				patches := gomonkey.ApplyFunc(security.AesDecrypt, func(s string) (string, error) {
					return "", os.ErrInvalid
				})
				defer patches.Reset()
			} else {
				patches := gomonkey.ApplyFunc(security.AesDecrypt, func(s string) (string, error) {
					return strings.TrimPrefix(s, "enc-"), nil
				})
				defer patches.Reset()
			}

			patches := gomonkey.ApplyFunc(security.AesEncrypt, func(s string) (string, error) {
				return "enc-" + s, nil
			})
			defer patches.Reset()

			node := confv1beta1.BKENode{}
			node.Spec.Password = tt.password
			node.Spec.Hostname = "test-node"
			opts := &Options{}

			result := opts.encryptBKENodePassword(node)

			assert.Equal(t, tt.expectedPassword, result.Spec.Password)
		})
	}
}

func TestOptionsDecryptBKENodePassword(t *testing.T) {
	tests := []struct {
		name             string
		password         string
		mockDecryptError bool
		expectedPassword string
	}{
		{
			name:             "valid encrypted password",
			password:         "enc-pass",
			mockDecryptError: false,
			expectedPassword: "dec-enc-pass",
		},
		{
			name:             "decrypt fails",
			password:         "invalid-enc",
			mockDecryptError: true,
			expectedPassword: "invalid-enc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.mockDecryptError {
				patches := gomonkey.ApplyFunc(security.AesDecrypt, func(s string) (string, error) {
					return "dec-" + s, nil
				})
				defer patches.Reset()
			} else {
				patches := gomonkey.ApplyFunc(security.AesDecrypt, func(s string) (string, error) {
					return "", os.ErrInvalid
				})
				defer patches.Reset()
			}

			node := confv1beta1.BKENode{}
			node.Spec.Password = tt.password
			node.Spec.Hostname = "test-node"
			opts := &Options{}

			result := opts.decryptBKENodePassword(node)

			assert.Equal(t, tt.expectedPassword, result.Spec.Password)
		})
	}
}

func TestOptionsSaveProcessedConfig(t *testing.T) {
	tests := []struct {
		name          string
		isEncrypt     bool
		mockMarshal   func(interface{}) ([]byte, error)
		mockGetwd     func() (string, error)
		mockWriteFile func(string, []byte, os.FileMode) error
		expectError   bool
	}{
		{
			name:      "encrypt success",
			isEncrypt: true,
			mockMarshal: func(v interface{}) ([]byte, error) {
				return []byte("yaml content"), nil
			},
			mockGetwd: func() (string, error) {
				return "/tmp", nil
			},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return nil
			},
			expectError: false,
		},
		{
			name:      "decrypt success",
			isEncrypt: false,
			mockMarshal: func(v interface{}) ([]byte, error) {
				return []byte("yaml content"), nil
			},
			mockGetwd: func() (string, error) {
				return "/tmp", nil
			},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return nil
			},
			expectError: false,
		},
		{
			name:      "marshal fails",
			isEncrypt: true,
			mockMarshal: func(v interface{}) ([]byte, error) {
				return nil, os.ErrInvalid
			},
			expectError: true,
		},
		{
			name:      "write file fails",
			isEncrypt: true,
			mockMarshal: func(v interface{}) ([]byte, error) {
				return []byte("yaml content"), nil
			},
			mockGetwd: func() (string, error) {
				return "/tmp", nil
			},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return os.ErrPermission
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(yaml2.Marshal, tt.mockMarshal)
			defer patches.Reset()

			if tt.mockGetwd != nil {
				patches = gomonkey.ApplyFunc(os.Getwd, tt.mockGetwd)
				defer patches.Reset()
			}

			if tt.mockWriteFile != nil {
				patches = gomonkey.ApplyFunc(os.WriteFile, tt.mockWriteFile)
				defer patches.Reset()
			}

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			conf := &configv1beta1.BKECluster{}
			conf.Name = "test-cluster"
			opts := &Options{}

			err := opts.saveProcessedConfig(conf, tt.isEncrypt)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
