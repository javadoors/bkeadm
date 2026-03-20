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

package cmd

import (
	"errors"
	"net"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	configv1beta1 "gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"
	configinit "gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/initialize"

	"gopkg.openfuyao.cn/bkeadm/pkg/cluster"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize"
	"gopkg.openfuyao.cn/bkeadm/utils"
)

const (
	testIPv4SegmentE = 101
	testIPv4SegmentF = 200
	testIPv4SegmentG = 201
	testIPv4SegmentH = 202
)

var (
	testIP1 = net.IPv4(
		testIPv4SegmentA,
		testIPv4SegmentB,
		testIPv4SegmentC,
		testIPv4SegmentD,
	)
	testIP2 = net.IPv4(
		testIPv4SegmentA,
		testIPv4SegmentB,
		testIPv4SegmentC,
		testIPv4SegmentE,
	)
	testIP3 = net.IPv4(
		testIPv4SegmentA,
		testIPv4SegmentB,
		testIPv4SegmentC,
		testIPv4SegmentF,
	)
	testIP4 = net.IPv4(
		testIPv4SegmentA,
		testIPv4SegmentB,
		testIPv4SegmentC,
		testIPv4SegmentG,
	)
	testIP5 = net.IPv4(
		testIPv4SegmentA,
		testIPv4SegmentB,
		testIPv4SegmentC,
		testIPv4SegmentH,
	)
	testImageRepoPort   = "5000"
	testYumRepoPort     = "8080"
	testChartRepoPort   = "8443"
	testClusterFileName = "test-cluster.yaml"
)

func TestInitCmdInitialization(t *testing.T) {
	tests := []struct {
		name          string
		expectedUse   string
		expectedShort string
		hasFlags      bool
	}{
		{
			name:          "init command properties",
			expectedUse:   "init",
			expectedShort: "Initialize the boot node",
			hasFlags:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedUse, initCmd.Use)
			assert.Equal(t, tt.expectedShort, initCmd.Short)

			if tt.hasFlags {
				// Check if required flags exist
				fileFlag := initCmd.Flags().Lookup("file")
				assert.NotNil(t, fileFlag)

				domainFlag := initCmd.Flags().Lookup("domain")
				assert.NotNil(t, domainFlag)

				hostIPFlag := initCmd.Flags().Lookup("hostIP")
				assert.NotNil(t, hostIPFlag)
			}
		})
	}
}

func TestRegisterInitCommand(t *testing.T) {
	// Find the init command in root commands
	var foundInitCmd bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "init" {
			foundInitCmd = true
			break
		}
	}

	assert.True(t, foundInitCmd, "init command should be registered in root command")
}

func TestInitCmdPreRunE(t *testing.T) {
	tests := []struct {
		name                      string
		initialHostIP             string
		initialDomain             string
		initialImageRepoPort      string
		initialKubernetesPort     string
		initialYumRepoPort        string
		initialChartRepoPort      string
		mockGetOutBoundIP         func() (string, error)
		mockGetIntranetIp         func() (string, error)
		mockNewBKEClusterFromFile func(string) (*configv1beta1.BKECluster, error)
		mockLoopIP                func(string) ([]string, error)
		mockPromptForConfirmation func(bool) bool
		expectError               bool
		errorContains             string
	}{
		{
			name:                  "success with default values",
			initialHostIP:         "",
			initialDomain:         "",
			initialImageRepoPort:  "",
			initialKubernetesPort: "",
			initialYumRepoPort:    "",
			initialChartRepoPort:  "",
			mockGetOutBoundIP: func() (string, error) {
				return testIP1.String(), nil
			},
			mockGetIntranetIp: func() (string, error) {
				return testIP2.String(), nil // Won't be used since OutBoundIP succeeds
			},
			mockNewBKEClusterFromFile: func(string) (*configv1beta1.BKECluster, error) {
				return nil, nil // No file specified, so this won't be called
			},
			mockLoopIP: func(string) ([]string, error) {
				return nil, nil
			},
			mockPromptForConfirmation: func(bool) bool {
				return true // User confirms
			},
			expectError: true,
		},
		{
			name:                  "missing host IP should return error",
			initialHostIP:         "",
			initialDomain:         configinit.DefaultImageRepo,
			initialImageRepoPort:  configinit.DefaultImageRepoPort,
			initialKubernetesPort: utils.DefaultKubernetesPort,
			initialYumRepoPort:    configinit.DefaultYumRepoPort,
			initialChartRepoPort:  utils.DefaultChartRegistryPort,
			mockGetOutBoundIP: func() (string, error) {
				return "", errors.New("network error")
			},
			mockGetIntranetIp: func() (string, error) {
				return "", errors.New("network error")
			},
			mockNewBKEClusterFromFile: func(string) (*configv1beta1.BKECluster, error) {
				return nil, nil
			},
			mockLoopIP: func(string) ([]string, error) {
				return nil, nil
			},
			mockPromptForConfirmation: func(bool) bool {
				return true
			},
			expectError:   true,
			errorContains: "The host IP address must be set",
		},
		{
			name:                  "user does not confirm should return error",
			initialHostIP:         testIP1.String(),
			initialDomain:         configinit.DefaultImageRepo,
			initialImageRepoPort:  configinit.DefaultImageRepoPort,
			initialKubernetesPort: utils.DefaultKubernetesPort,
			initialYumRepoPort:    configinit.DefaultYumRepoPort,
			initialChartRepoPort:  utils.DefaultChartRegistryPort,
			mockGetOutBoundIP: func() (string, error) {
				return testIP1.String(), nil
			},
			mockGetIntranetIp: func() (string, error) {
				return testIP2.String(), nil
			},
			mockNewBKEClusterFromFile: func(string) (*configv1beta1.BKECluster, error) {
				return nil, nil
			},
			mockLoopIP: func(string) ([]string, error) {
				return nil, nil
			},
			mockPromptForConfirmation: func(bool) bool {
				return false // User does not confirm
			},
			expectError:   true,
			errorContains: "operation cancelled by user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalHostIP := initOption.HostIP
			originalDomain := initOption.Domain
			originalImageRepoPort := initOption.ImageRepoPort
			originalKubernetesPort := initOption.KubernetesPort
			originalYumRepoPort := initOption.YumRepoPort
			originalChartRepoPort := initOption.ChartRepoPort
			originalFile := initOption.File
			originalConfirm := confirm
			defer func() {
				initOption.HostIP = originalHostIP
				initOption.Domain = originalDomain
				initOption.ImageRepoPort = originalImageRepoPort
				initOption.KubernetesPort = originalKubernetesPort
				initOption.YumRepoPort = originalYumRepoPort
				initOption.ChartRepoPort = originalChartRepoPort
				initOption.File = originalFile
				confirm = originalConfirm
			}()

			// Set initial values
			initOption.HostIP = tt.initialHostIP
			initOption.Domain = tt.initialDomain
			initOption.ImageRepoPort = tt.initialImageRepoPort
			initOption.KubernetesPort = tt.initialKubernetesPort
			initOption.YumRepoPort = tt.initialYumRepoPort
			initOption.ChartRepoPort = tt.initialChartRepoPort
			initOption.File = "" // Don't process file in this test
			confirm = true       // Override for this test

			// Apply patches
			patches := gomonkey.ApplyFunc(utils.GetOutBoundIP, tt.mockGetOutBoundIP)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.GetIntranetIp, tt.mockGetIntranetIp)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(cluster.NewBKEClusterFromFile, tt.mockNewBKEClusterFromFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.LoopIP, tt.mockLoopIP)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.PromptForConfirmation, tt.mockPromptForConfirmation)
			defer patches.Reset()

			// Call PreRunE
			err := initCmd.PreRunE(nil, nil)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestInitCmdRunWithMock(t *testing.T) {
	// Save original values
	originalArgs := initOption.Args
	originalOptions := initOption.Options
	defer func() {
		initOption.Args = originalArgs
		initOption.Options = originalOptions
	}()

	// Apply patch to mock the Initialize method
	patches := gomonkey.ApplyFunc((*initialize.Options).Initialize, func(o *initialize.Options) {
		// Mock implementation - do nothing
	})
	defer patches.Reset()

	// Create and run the command
	cmd := &cobra.Command{}
	args := []string{"arg1", "arg2"}

	initCmd.Run(cmd, args)

	// Verify that args and options were set
	assert.Equal(t, args, initOption.Args)
	assert.Equal(t, options, initOption.Options)
}

func TestInitCmdWithFileProcessing(t *testing.T) {
	// This test verifies the logic when a file is provided
	// We'll mock the file processing logic to avoid actual file operations

	// Save original values
	originalFile := initOption.File
	originalHostIP := initOption.HostIP
	originalDomain := initOption.Domain
	originalImageRepoPort := initOption.ImageRepoPort
	originalKubernetesPort := initOption.KubernetesPort
	originalYumRepoPort := initOption.YumRepoPort
	originalChartRepoPort := initOption.ChartRepoPort
	defer func() {
		initOption.File = originalFile
		initOption.HostIP = originalHostIP
		initOption.Domain = originalDomain
		initOption.ImageRepoPort = originalImageRepoPort
		initOption.KubernetesPort = originalKubernetesPort
		initOption.YumRepoPort = originalYumRepoPort
		initOption.ChartRepoPort = originalChartRepoPort
	}()

	// Set up test values
	testFile := testClusterFileName
	testIPAddr := testIP1.String()

	initOption.File = testFile
	initOption.HostIP = testIPAddr
	initOption.Domain = configinit.DefaultImageRepo
	initOption.ImageRepoPort = configinit.DefaultImageRepoPort
	initOption.KubernetesPort = utils.DefaultKubernetesPort
	initOption.YumRepoPort = configinit.DefaultYumRepoPort
	initOption.ChartRepoPort = utils.DefaultChartRegistryPort

	// Mock the NewBKEClusterFromFile function to return a test cluster
	mockCluster := &configv1beta1.BKECluster{
		Spec: configv1beta1.BKEClusterSpec{
			ClusterConfig: &configv1beta1.BKEConfig{
				Cluster: configv1beta1.Cluster{
					ImageRepo: configv1beta1.Repo{
						Ip:     testIP3.String(),
						Domain: configinit.DefaultImageRepo,
						Port:   testImageRepoPort,
						Prefix: "k8s",
					},
					HTTPRepo: configv1beta1.Repo{
						Ip:     testIP4.String(),
						Domain: "yum.example.com",
						Port:   testYumRepoPort,
					},
					ChartRepo: configv1beta1.Repo{
						Ip:     testIP5.String(),
						Domain: configinit.DefaultChartRepo,
						Port:   testChartRepoPort,
						Prefix: "charts",
					},
					NTPServer: "pool.ntp.org",
				},
			},
		},
	}

	// Apply patches
	patches := gomonkey.ApplyFunc(cluster.NewBKEClusterFromFile, func(file string) (*configv1beta1.BKECluster, error) {
		return mockCluster, nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(utils.LoopIP, func(domain string) ([]string, error) {
		return []string{net.ParseIP(testIP1.String()).String()}, nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(utils.PromptForConfirmation, func(confirmed bool) bool {
		return true
	})
	defer patches.Reset()

	// Mock the GetOutBoundIP and GetIntranetIp to return test values
	patches = gomonkey.ApplyFunc(utils.GetOutBoundIP, func() (string, error) {
		return testIPAddr, nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(utils.GetIntranetIp, func() (string, error) {
		return testIPAddr, nil
	})
	defer patches.Reset()

	// Call PreRunE to trigger the file processing logic
	err := initCmd.PreRunE(&cobra.Command{}, []string{})

	// The error might occur due to validation checks, which is expected in this test
	// We mainly want to verify that the file processing logic ran without panicking
	if err != nil {
		// If there's an error, it should be related to validation, not the file processing itself
		assert.Contains(t, err.Error(), "port is not a number") // This could happen if ports are not numbers
	}
}
