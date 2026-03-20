/******************************************************************
 * Copyright (c) 2024 Bocloud Technologies Co., Ltd.
 * installer is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain n copy of Mulan PSL v2 at:
 *          http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 ******************************************************************/

package cmd

import (
	"github.com/spf13/cobra"

	"gopkg.openfuyao.cn/bkeadm/pkg/root"
)

var (
	doc     bool
	options root.Options
)

// rootCmd is the base command executed when no subcommands are provided.
var rootCmd = &cobra.Command{
	Use:   "bke",
	Short: "Bocloud Enterprise Kubernetes deployment tool.",
	Long: `Bocloud Enterprise Kubernetes deployment tool.
It provides an integrated solution for Kubernetes cluster deployment,
operation, maintenance, and governance.`,
	Run: func(cmd *cobra.Command, args []string) {
		options.Args = args
		if doc {
			options.PrintDoc()
			return
		}
		options.Print()
	},
}

// Execute runs the root command.
// It should be called once in the main function.
func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize()
	rootCmd.PersistentFlags().StringVar(
		&options.KubeConfig,
		"kubeconfig",
		"",
		"Path to the Kubernetes configuration file.",
	)
	rootCmd.PersistentFlags().BoolVar(
		&doc,
		"doc",
		false,
		"Display command documentation.",
	)

	// Register all subcommands
	registerInitCommand()
	registerResetCommand()
	registerStartCommand()
	registerStatusCommand()
	registerVersionCommand()
	registerConfigCommand()
	registerRegistryCommand()
	registerBuildCommand()
	registerClusterCommand()
	registerRemoveCommand()
	registerCommandCommand()
}
