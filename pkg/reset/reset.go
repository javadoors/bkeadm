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

package reset

import (
	"fmt"
	"os"
	"path"

	"gopkg.openfuyao.cn/cluster-api-provider-bke/common/source"

	econd "gopkg.openfuyao.cn/bkeadm/pkg/executor/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize/syscompat"
	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/pkg/server"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type Options struct {
	root.Options
	Args  []string `json:"args"`
	All   bool     `json:"all"`
	Mount bool     `json:"mount"`
}

func (op *Options) Reset() {
	RemoveLocalKubernetes()
	if op.All {
		RemoveContainerService()
		RemoveNtpService()
		removeAllInOne()
		err := source.ResetSource()
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Reset source error: %s", err.Error()))
		}
		err = syscompat.RepoUpdate()
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Update source error: %s", err.Error()))
		}
	}
	if op.Mount {
		RemoveDataDir()
		if !op.All {
			RemoveContainerService()
			RemoveNtpService()
			err := source.ResetSource()
			if err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("Reset source error: %s", err.Error()))
			}
		}
	}
	log.BKEFormat(log.INFO, "BKE reset completed")
}

func RemoveLocalKubernetes() {
	if infrastructure.IsDocker() {
		log.BKEFormat(log.INFO, "Remove local Kubernetes")
		_ = global.Docker.ContainerRemove(utils.LocalKubernetesName)
		if err := os.RemoveAll("/etc/rancher"); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove /etc/rancher: %v", err))
		}
		if err := os.RemoveAll("/var/lib/rancher"); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove /var/lib/rancher: %v", err))
		}
		return
	}
	if infrastructure.IsContainerd() {
		log.BKEFormat(log.INFO, "Remove local Kubernetes")
		_ = econd.ContainerRemove(utils.LocalKubernetesName)
		if err := os.RemoveAll("/etc/rancher"); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove /etc/rancher: %v", err))
		}
		if err := os.RemoveAll("/var/lib/rancher"); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove /var/lib/rancher: %v", err))
		}
		return
	}
}

func RemoveContainerService() {
	_ = server.RemoveImageRegistry(utils.LocalImageRegistryName)
	_ = server.RemoveChartRegistry(utils.LocalChartRegistryName)
	_ = server.RemoveYumRegistry(utils.LocalYumRegistryName)
	_ = server.RemoveNFSServer(utils.LocalNFSRegistryName)
}

func RemoveNtpService() {
	_ = server.RemoveNTPServer()
}

func RemoveDataDir() {
	err := os.RemoveAll(path.Join(global.Workspace, "mount"))
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Remove mount dir error: %s", err.Error()))
	}
}

func removeAllInOne() {
	cleanKubeletBin()
	cleanKubernetesContainer()
	cleanOtherContainer()
	cleanContainerRuntime()
	cleanNetwork()
	cleanNeedDeleteFile()
}
