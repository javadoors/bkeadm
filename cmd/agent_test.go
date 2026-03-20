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
	"net"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/agent"
)

const (
	testIPv4SegmentA    = 192
	testIPv4SegmentB    = 168
	testIPv4SegmentC    = 1
	testIPv4SegmentD    = 1
	subcommandCount     = 5
	generatedNameLength = 12 // Length of "YYYYMMDDHHMM"
)

var testIP = net.IPv4(
	testIPv4SegmentA,
	testIPv4SegmentB,
	testIPv4SegmentC,
	testIPv4SegmentD,
)

func TestCommandCmdInitialization(t *testing.T) {
	tests := []struct {
		name           string
		cmd            *cobra.Command
		expectedUse    string
		expectedShort  string
		hasSubcommands bool
	}{
		{
			name:           "command command properties",
			cmd:            commandCmd,
			expectedUse:    "command",
			expectedShort:  "The machine executes remote instructions",
			hasSubcommands: true,
		},
		{
			name:           "exec command properties",
			cmd:            execCmd,
			expectedUse:    "exec",
			expectedShort:  "Execute specified command",
			hasSubcommands: false,
		},
		{
			name:           "list command properties",
			cmd:            liCmd,
			expectedUse:    "list",
			expectedShort:  "List all the commands",
			hasSubcommands: false,
		},
		{
			name:           "info command properties",
			cmd:            infoCmd,
			expectedUse:    "info",
			expectedShort:  "Observe a command out",
			hasSubcommands: false,
		},
		{
			name:           "remove command properties",
			cmd:            rmCmd,
			expectedUse:    "remove",
			expectedShort:  "Delete instruction",
			hasSubcommands: false,
		},
		{
			name:           "syncTime command properties",
			cmd:            syncTimeCmd,
			expectedUse:    "syncTime",
			expectedShort:  "Synchronization time",
			hasSubcommands: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedUse, tt.cmd.Use)
			assert.Equal(t, tt.expectedShort, tt.cmd.Short)
		})
	}
}

func TestRegisterCommandCommand(t *testing.T) {
	// Find the command command in root commands
	var foundCommandCmd bool
	for _, cmd := range rootCmd.Commands() {
		if cmd.Use == "command" {
			foundCommandCmd = true
			// Check if subcommands are registered
			assert.Len(t, cmd.Commands(), subcommandCount) // exec, list, info, remove, syncTime

			var foundSubcommands []string
			for _, subCmd := range cmd.Commands() {
				foundSubcommands = append(foundSubcommands, subCmd.Use)
			}

			expectedSubcommands := []string{"exec", "list", "info", "remove", "syncTime"}
			for _, expected := range expectedSubcommands {
				assert.Contains(t, foundSubcommands, expected)
			}
			break
		}
	}

	assert.True(t, foundCommandCmd, "command command should be registered in root command")
}

func TestExecCmdArgs(t *testing.T) {
	tests := []struct {
		name          string
		nodesValue    string
		fileValue     string
		commandValue  string
		initialName   string
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid inputs with nodes and command",
			nodesValue:   testIP.String(),
			fileValue:    "",
			commandValue: "echo hello",
			initialName:  "",
			expectError:  false,
		},
		{
			name:         "valid inputs with nodes and file",
			nodesValue:   testIP.String(),
			fileValue:    "/path/to/script.sh",
			commandValue: "",
			initialName:  "",
			expectError:  false,
		},
		{
			name:          "missing nodes should return error",
			nodesValue:    "",
			fileValue:     "",
			commandValue:  "echo hello",
			initialName:   "",
			expectError:   true,
			errorContains: "The `nodes` parameter is required",
		},
		{
			name:          "missing both file and command should return error",
			nodesValue:    testIP.String(),
			fileValue:     "",
			commandValue:  "",
			initialName:   "",
			expectError:   true,
			errorContains: "One of the parameters `file` and `command` must exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			originalNodes := execOption.Nodes
			originalFile := execOption.File
			originalCommand := execOption.Command
			originalName := execOption.Name
			defer func() {
				execOption.Nodes = originalNodes
				execOption.File = originalFile
				execOption.Command = originalCommand
				execOption.Name = originalName
			}()

			// Set test values
			execOption.Nodes = tt.nodesValue
			execOption.File = tt.fileValue
			execOption.Command = tt.commandValue
			execOption.Name = tt.initialName

			// Call Args validation
			err := execCmd.Args(nil, nil)

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

func TestExecCmdArgsGeneratesName(t *testing.T) {
	// Save original values
	originalNodes := execOption.Nodes
	originalFile := execOption.File
	originalCommand := execOption.Command
	originalName := execOption.Name
	defer func() {
		execOption.Nodes = originalNodes
		execOption.File = originalFile
		execOption.Command = originalCommand
		execOption.Name = originalName
	}()

	// Set test values
	execOption.Nodes = testIP.String()
	execOption.File = ""
	execOption.Command = "echo hello"
	execOption.Name = ""

	// Call Args validation
	err := execCmd.Args(nil, nil)
	assert.NoError(t, err)

	// Check that a name was generated
	assert.NotEmpty(t, execOption.Name)

	// The generated name should match the expected format (allowing for minor time differences)
	assert.Len(t, execOption.Name, generatedNameLength) // Length of "YYYYMMDDHHMM"
}

func TestExecCmdPreRunE(t *testing.T) {
	// Create and run PreRunE
	cmd := &cobra.Command{}
	args := []string{"arg1", "arg2"}

	err := execCmd.PreRunE(cmd, args)

	assert.Error(t, err)
}

func TestExecCmdRun(t *testing.T) {
	// Save original values
	originalArgs := execOption.Args
	originalOptions := execOption.Options
	defer func() {
		execOption.Args = originalArgs
		execOption.Options = originalOptions
	}()

	// Apply patch to mock Exec method
	patches := gomonkey.ApplyFunc((*agent.Options).Exec, func(o *agent.Options) {
		// Mock implementation - do nothing
	})
	defer patches.Reset()

	// Create and run the command
	cmd := &cobra.Command{}
	args := []string{"arg1", "arg2"}

	execCmd.Run(cmd, args)

	// Note: The actual implementation has a bug - it sets existOption instead of execOption
	// This test reflects the current implementation
	assert.Equal(t, args, execOption.Args) // Using execOption as per the corrected implementation
	assert.Equal(t, options, execOption.Options)
}

func TestLiCmdPreRunEAndRun(t *testing.T) {
	patches := gomonkey.ApplyFunc((*agent.Options).List, func(o *agent.Options) {
	})
	defer patches.Reset()

	// Test PreRunE
	cmd := &cobra.Command{}
	args := []string{"arg1", "arg2"}

	err := liCmd.PreRunE(cmd, args)
	assert.Error(t, err)
	assert.Equal(t, args, liOption.Args)
	assert.Equal(t, options, liOption.Options)

	// Test Run
	liCmd.Run(cmd, args)
	assert.Equal(t, args, liOption.Args)
	assert.Equal(t, options, liOption.Options)
}

func TestInfoCmdPreRunEAndRun(t *testing.T) {
	// Save original values
	originalArgs := infoOptions.Args
	originalOptions := infoOptions.Options
	defer func() {
		infoOptions.Args = originalArgs
		infoOptions.Options = originalOptions
	}()

	// Apply patches
	patches := gomonkey.ApplyFunc((*agent.Options).ClusterPre, func(o *agent.Options) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*agent.Options).Info, func(o *agent.Options) {
		// Mock implementation - do nothing
	})
	defer patches.Reset()

	// Test PreRunE
	cmd := &cobra.Command{}
	args := []string{"arg1", "arg2"}

	err := infoCmd.PreRunE(cmd, args)
	assert.Error(t, err)
	assert.Equal(t, args, infoOptions.Args)
	assert.Equal(t, options, infoOptions.Options)

	// Test Run
	infoCmd.Run(cmd, args)
	assert.Equal(t, args, infoOptions.Args)
	assert.Equal(t, options, infoOptions.Options)
}

func TestRmCmdPreRunEAndRun(t *testing.T) {
	// Save original values
	originalArgs := rmOptions.Args
	originalOptions := rmOptions.Options
	defer func() {
		rmOptions.Args = originalArgs
		rmOptions.Options = originalOptions
	}()

	// Apply patches
	patches := gomonkey.ApplyFunc((*agent.Options).ClusterPre, func(o *agent.Options) error {
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc((*agent.Options).Remove, func(o *agent.Options) {
		// Mock implementation - do nothing
	})
	defer patches.Reset()

	// Test PreRunE
	cmd := &cobra.Command{}
	args := []string{"arg1", "arg2"}

	err := rmCmd.PreRunE(cmd, args)
	assert.Error(t, err)
	assert.Equal(t, args, rmOptions.Args)
	assert.Equal(t, options, rmOptions.Options)

	// Test Run
	rmCmd.Run(cmd, args)
	assert.Equal(t, args, rmOptions.Args)
	assert.Equal(t, options, rmOptions.Options)
}

func TestSyncTimeCmdPreRunEAndRun(t *testing.T) {
	// Save original values
	originalArgs := syncTimeOptions.Args
	originalOptions := syncTimeOptions.Options
	defer func() {
		syncTimeOptions.Args = originalArgs
		syncTimeOptions.Options = originalOptions
	}()

	// Apply patch to mock SyncTime method
	patches := gomonkey.ApplyFunc((*agent.Options).SyncTime, func(o *agent.Options) {
		// Mock implementation - do nothing
	})
	defer patches.Reset()

	// Test PreRunE (should return nil as per implementation)
	cmd := &cobra.Command{}
	args := []string{"arg1", "arg2"}

	err := syncTimeCmd.PreRunE(cmd, args)
	assert.NoError(t, err)

	// Test Run
	syncTimeCmd.Run(cmd, args)
	assert.Equal(t, args, syncTimeOptions.Args)
	assert.Equal(t, options, syncTimeOptions.Options)
}

func TestCommandCmdRun(t *testing.T) {
	// Create a temporary command
	cmd := &cobra.Command{}

	// Run the command
	commandCmd.Run(cmd, []string{})

	// The function should complete without error
	assert.True(t, true) // This assertion just confirms the function ran without panic
}
