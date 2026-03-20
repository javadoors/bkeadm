/******************************************************************
 * Copyright (c) 2025 Bocloud Technologies Co., Ltd.
 * installer is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain n copy of Mulan PSL v2 at:
 *          http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 ******************************************************************/

package main

import (
	"gopkg.openfuyao.cn/bkeadm/cmd"
	"gopkg.openfuyao.cn/bkeadm/utils/version"
)

var gitCommitId = "dev"
var architecture = ""
var timestamp = ""
var ver = "v1.0"

func main() {
	if version.Version == "" {
		version.GitCommitID = gitCommitId
		version.Version = ver
		version.Architecture = architecture
		version.Timestamp = timestamp
	}
	cmd.Execute()
}
