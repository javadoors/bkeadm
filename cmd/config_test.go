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
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	confv1beta1 "gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"

	"gopkg.openfuyao.cn/bkeadm/pkg/config"
)

const (
	configSubcommandCount = 2
)

func TestConfigCmdInitialization(t *testing.T) {
	tests := []struct {
		name          string
		cmd           *cobra.Command
		expectedUse   string
		expectedShort string
		hasFlags      bool
	}{
		{
			name:          "config command properties",
			cmd:           configCmd,
			expectedUse:   "config",
			expectedShort: "Generate bke configuration.",
			hasFlags:      true,
		},
		{
			name:          "encrypt command properties",
			cmd:           encryptCmd,
			expectedUse:   "encrypt",
			expectedShort: "Encryption configuration file.",
			hasFlags:      true,
		},
		{
			name:          "decrypt command properties",
			cmd:           decryptCmd,
			expectedUse:   "decrypt",
			expectedShort: "Decrypting configuration files.",
			hasFlags:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedUse, tt.cmd.Use)
			assert.Equal(t, tt.expectedShort, tt.cmd.Short)

			if tt.hasFlags {
				// For config command, check if directory and product flags exist
				if tt.cmd == configCmd {
					dirFlag := configCmd.Flags().Lookup("directory")
					assert.NotNil(t, dirFlag)

					productFlag := configCmd.Flags().Lookup("product")
					assert.NotNil(t, productFlag)
				}

				// For encrypt and decrypt commands, check if file flag exists
				if tt.cmd == encryptCmd || tt.cmd == decryptCmd {
					fileFlag := tt.cmd.Flags().Lookup("file")
					assert.NotNil(t, fileFlag)
				}
			}
		})
	}
}

func TestRegisterConfigCommand(t *testing.T) {
	// Find the config command in root commands
	var foundConfigCmd bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "config" {
			foundConfigCmd = true
			// Check if encrypt and decrypt commands are registered as subcommands
			assert.Len(t, cmd.Commands(), configSubcommandCount) // encrypt and decrypt commands

			var foundEncrypt, foundDecrypt bool
			for _, subCmd := range cmd.Commands() {
				if subCmd.Use == "encrypt" {
					foundEncrypt = true
				}
				if subCmd.Use == "decrypt" {
					foundDecrypt = true
				}
			}
			assert.True(t, foundEncrypt, "encrypt command should be registered as subcommand")
			assert.True(t, foundDecrypt, "decrypt command should be registered as subcommand")
			break
		}
	}

	assert.True(t, foundConfigCmd, "config command should be registered in root command")
}

func TestConfigCmdPreRunE(t *testing.T) {
	tests := []struct {
		name            string
		initialDir      string
		initialProduct  string
		mockGetwd       func() (dir string, err error)
		expectedDir     string
		expectedProduct string
		expectError     bool
	}{
		{
			name:           "empty directory gets current working directory",
			initialDir:     "",
			initialProduct: "",
			mockGetwd: func() (dir string, err error) {
				return "/tmp/test", nil
			},
			expectedDir:     "/tmp/test",
			expectedProduct: "boc4.0-portal",
			expectError:     false,
		},
		{
			name:           "non-empty directory remains unchanged",
			initialDir:     "/custom/dir",
			initialProduct: "my-product",
			mockGetwd: func() (dir string, err error) {
				return "/tmp/test", nil // This shouldn't be used
			},
			expectedDir:     "/custom/dir",
			expectedProduct: "my-product",
			expectError:     false,
		},
		{
			name:           "empty product gets default value",
			initialDir:     "/some/dir",
			initialProduct: "",
			mockGetwd: func() (dir string, err error) {
				return "/tmp/test", nil
			},
			expectedDir:     "/some/dir",
			expectedProduct: "boc4.0-portal",
			expectError:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalDir := configOption.Directory
			originalProduct := configOption.Product
			defer func() {
				configOption.Directory = originalDir
				configOption.Product = originalProduct
			}()

			// Set initial values
			configOption.Directory = tt.initialDir
			configOption.Product = tt.initialProduct

			// Apply patches
			var patches *gomonkey.Patches
			if tt.mockGetwd != nil {
				patches = gomonkey.ApplyFunc(os.Getwd, tt.mockGetwd)
				defer patches.Reset()
			}

			// Call PreRunE
			err := configCmd.PreRunE(nil, nil)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedDir, configOption.Directory)
				assert.Equal(t, tt.expectedProduct, configOption.Product)
			}
		})
	}
}

func TestEncryptCmdArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		fileValue   string
		expectError bool
	}{
		{
			name:        "no args and no file should return error",
			args:        []string{},
			fileValue:   "",
			expectError: true,
		},
		{
			name:        "with args but no file should not return error",
			args:        []string{"test"},
			fileValue:   "",
			expectError: false,
		},
		{
			name:        "no args but with file should not return error",
			args:        []string{},
			fileValue:   "test.yaml",
			expectError: false,
		},
		{
			name:        "both args and file should not return error",
			args:        []string{"test"},
			fileValue:   "test.yaml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalFile := encryptOption.File
			defer func() {
				encryptOption.File = originalFile
			}()

			// Set file value
			encryptOption.File = tt.fileValue

			// Call Args validation
			err := encryptCmd.Args(nil, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDecryptCmdArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		fileValue   string
		expectError bool
	}{
		{
			name:        "no args and no file should return error",
			args:        []string{},
			fileValue:   "",
			expectError: true,
		},
		{
			name:        "with args but no file should not return error",
			args:        []string{"test"},
			fileValue:   "",
			expectError: false,
		},
		{
			name:        "no args but with file should not return error",
			args:        []string{},
			fileValue:   "test.yaml",
			expectError: false,
		},
		{
			name:        "both args and file should not return error",
			args:        []string{"test"},
			fileValue:   "test.yaml",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			originalFile := decryptOption.File
			defer func() {
				decryptOption.File = originalFile
			}()

			// Set file value
			decryptOption.File = tt.fileValue

			// Call Args validation
			err := decryptCmd.Args(nil, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigCmdRun(t *testing.T) {
	// Apply patches to mock the Config method
	patches := gomonkey.ApplyFunc((*config.Options).Config,
		func(op *config.Options, customExtra map[string]string, imageRepo, yumRepo, chartRepo confv1beta1.Repo, ntpServer string) {
			// Mock implementation - do nothing
		})
	defer patches.Reset()

	// Create and run the command
	cmd := &cobra.Command{}
	args := []string{"arg1", "arg2"}

	// Capture the state before running
	originalArgs := configOption.Args
	originalOptions := configOption.Options

	configCmd.Run(cmd, args)

	// Verify that args and options were set
	assert.Equal(t, args, configOption.Args)
	assert.Equal(t, options, configOption.Options)

	// Restore original values
	configOption.Args = originalArgs
	configOption.Options = originalOptions
}

func TestEncryptCmdRun(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		fileValue         string
		mockEncryptString func(*config.Options) error
		mockEncryptFile   func(*config.Options) error
	}{
		{
			name:      "run with args calls EncryptString",
			args:      []string{"test"},
			fileValue: "",
			mockEncryptString: func(o *config.Options) error {
				return nil
			},
			mockEncryptFile: func(o *config.Options) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalArgs := encryptOption.Args
			originalOptions := encryptOption.Options
			originalFile := encryptOption.File
			defer func() {
				encryptOption.Args = originalArgs
				encryptOption.Options = originalOptions
				encryptOption.File = originalFile
			}()

			// Set values
			encryptOption.File = tt.fileValue

			// Apply patches to mock the encryption methods
			stringCalled := false
			fileCalled := false

			patches := gomonkey.ApplyFunc((*config.Options).EncryptString, func(o *config.Options) error {
				stringCalled = true
				return tt.mockEncryptString(o)
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc((*config.Options).EncryptFile, func(o *config.Options) error {
				fileCalled = true
				return tt.mockEncryptFile(o)
			})
			defer patches.Reset()

			// Create and run the command
			cmd := &cobra.Command{}
			encryptCmd.Run(cmd, tt.args)

			// Verify that the correct methods were called
			if len(tt.args) > 0 {
				assert.True(t, stringCalled, "EncryptString should be called when args are provided")
			} else {
				assert.False(t, stringCalled, "EncryptString should not be called when no args are provided")
			}

			if tt.fileValue != "" {
				assert.True(t, fileCalled, "EncryptFile should be called when file is provided")
			} else {
				assert.False(t, fileCalled, "EncryptFile should not be called when no file is provided")
			}

			// Verify that args and options were set
			assert.Equal(t, tt.args, encryptOption.Args)
			assert.Equal(t, options, encryptOption.Options)
		})
	}
}

func TestDecryptCmdRun(t *testing.T) {
	tests := []struct {
		name              string
		args              []string
		fileValue         string
		mockDecryptString func(*config.Options) error
		mockDecryptFile   func(*config.Options) error
	}{
		{
			name:      "run with args calls DecryptString",
			args:      []string{"test"},
			fileValue: "",
			mockDecryptString: func(o *config.Options) error {
				return nil
			},
			mockDecryptFile: func(o *config.Options) error {
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalArgs := decryptOption.Args
			originalOptions := decryptOption.Options
			originalFile := decryptOption.File
			defer func() {
				decryptOption.Args = originalArgs
				decryptOption.Options = originalOptions
				decryptOption.File = originalFile
			}()

			// Set values
			decryptOption.File = tt.fileValue

			// Apply patches to mock the decryption methods
			stringCalled := false
			fileCalled := false

			patches := gomonkey.ApplyFunc((*config.Options).DecryptString, func(o *config.Options) error {
				stringCalled = true
				return tt.mockDecryptString(o)
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc((*config.Options).DecryptFile, func(o *config.Options) error {
				fileCalled = true
				return tt.mockDecryptFile(o)
			})
			defer patches.Reset()

			// Create and run the command
			cmd := &cobra.Command{}
			decryptCmd.Run(cmd, tt.args)

			// Verify that the correct methods were called
			if len(tt.args) > 0 {
				assert.True(t, stringCalled, "DecryptString should be called when args are provided")
			} else {
				assert.False(t, stringCalled, "DecryptString should not be called when no args are provided")
			}

			if tt.fileValue != "" {
				assert.True(t, fileCalled, "DecryptFile should be called when file is provided")
			} else {
				assert.False(t, fileCalled, "DecryptFile should not be called when no file is provided")
			}

			// Verify that args and options were set
			assert.Equal(t, tt.args, decryptOption.Args)
			assert.Equal(t, options, decryptOption.Options)
		})
	}
}
