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

// Package kubelet 实现kubelet的CRD安装
package kubelet

import (
	_ "embed"
	"fmt"
	"os"
	"time"

	configv1beta1 "gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"
	yaml2 "sigs.k8s.io/yaml"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/k8s"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

var (
	//go:embed kubelet_crd.yaml
	kubeletCrd []byte

	//go:embed kubelet_default.yaml
	kubeletDefault []byte
)

// ApplyKubeletCfg 安装kubelet的CRD
func ApplyKubeletCfg() error {
	err := applyKubeletdCrd()
	if err != nil {
		return fmt.Errorf("apply kubelet crd failed: %s", err.Error())
	}

	err = applyContainerdDefault()
	if err != nil {
		return fmt.Errorf("apply kubelet default failed: %s", err.Error())
	}

	log.BKEFormat(log.INFO, "Apply kubelet crd and default success")

	return nil
}

func applyKubeletdCrd() error {
	var err error
	if global.K8s == nil {
		global.K8s, err = k8s.NewKubernetesClient("")
		if err != nil {
			return err
		}
	}

	kubeletCrdFile := fmt.Sprintf("%s/tmpl/kubelet_crd.yaml", global.Workspace)
	err = os.WriteFile(kubeletCrdFile, kubeletCrd, utils.DefaultFilePermission)
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)

	log.BKEFormat(log.INFO, "Install kubelet CRD...")
	err = global.K8s.InstallYaml(kubeletCrdFile, nil, "")
	if err != nil {
		return err
	}

	return nil
}

func applyContainerdDefault() error {
	runtimeParam := map[string]string{}
	conf := &configv1beta1.KubeletConfig{}
	if err := yaml2.Unmarshal(kubeletDefault, conf); err != nil {
		return fmt.Errorf("unmarshal kubelet default failed: %s", err.Error())
	}

	if err := k8s.CreateNamespace(global.K8s, conf.Namespace); err != nil {
		return err
	}

	kubeletDefaultFile := fmt.Sprintf("%s/tmpl/kubelet_default.yaml", global.Workspace)
	err := os.WriteFile(kubeletDefaultFile, kubeletDefault, utils.DefaultFilePermission)
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)

	log.BKEFormat(log.INFO, fmt.Sprintf("Submit kubelet default yaml to the cluster"))
	err = global.K8s.InstallYaml(kubeletDefaultFile, runtimeParam, "")
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install kubelet default, %v", err))
		return nil
	}
	log.BKEFormat(log.INFO, fmt.Sprintf("Submit the kubelet configuration to the cluster"))

	return nil
}
