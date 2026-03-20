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

package global

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/exec"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/k8s"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

var (
	Docker      docker.DockerClient
	Containerd  containerd.ContainerdClient
	K8s         k8s.KubernetesClient
	Command     exec.Executor
	Workspace   string
	CustomExtra map[string]string
)

func init() {
	Command = &exec.CommandExecutor{}
	if utils.IsFile("/opt/BKE_WORKSPACE") {
		f, err := os.ReadFile("/opt/BKE_WORKSPACE")
		if err == nil {
			Workspace = string(f)
			Workspace = strings.TrimSpace(Workspace)
			Workspace = strings.TrimRight(Workspace, "\n")
			Workspace = strings.TrimRight(Workspace, "\r")
			Workspace = strings.TrimRight(Workspace, "\t")
		}
	}
	if os.Getenv("BKE_WORKSPACE") != "" {
		Workspace = os.Getenv("BKE_WORKSPACE")
	}
	if Workspace == "" {
		Workspace = "/bke"
	}
	if !utils.Exists(Workspace + "/tmpl") {
		if err := os.MkdirAll(Workspace+"/tmpl", utils.DefaultDirPermission); err != nil {
			log.Warnf("failed to create tmpl directory: %s", err.Error())
		}
	}
	if !utils.Exists(Workspace + "/volumes") {
		if err := os.MkdirAll(Workspace+"/volumes", utils.DefaultDirPermission); err != nil {
			log.Warnf("failed to create volumes directory: %s", err.Error())
		}
	}
	if !utils.Exists(Workspace + "/mount") {
		if err := os.MkdirAll(Workspace+"/mount", utils.DefaultDirPermission); err != nil {
			log.Warnf("failed to create mount directory: %s", err.Error())
		}
	}
	CustomExtra = make(map[string]string)
}

func TarGZ(prefix, target string) error {
	output, err := Command.ExecuteCommandWithOutput("sh", "-c", fmt.Sprintf("cd %s && tar --use-compress-program=pigz -cf %s .", prefix, target))
	if err != nil {
		return errors.New(output + err.Error())
	}
	return nil
}

func TarGZWithDir(prefix, dir, target string) error {
	output, err := Command.ExecuteCommandWithOutput("sh", "-c", fmt.Sprintf("cd %s && tar "+
		"--use-compress-program=pigz -cf %s ./%s", prefix, target, dir))
	if err != nil {
		return errors.New(output + err.Error())
	}
	return nil
}

// TaeGZWithoutChangeFile tar gz without change file
func TaeGZWithoutChangeFile(prefix, target string) error {
	cmd := fmt.Sprintf("cd %s && tar --use-compress-program=pigz -cf %s . --warning=no-file-changed --ignore-failed-read",
		prefix, target)
	output, err := Command.ExecuteCommandWithOutput("sh", "-c", cmd)
	if err != nil {
		return errors.New(output + err.Error())
	}
	return nil
}

func UnTarGZ(dataFile, target string) error {
	output, err := Command.ExecuteCommandWithOutput("sh", "-c", fmt.Sprintf("tar -xzf %s -C %s", dataFile, target))
	if err != nil {
		return errors.New(output + err.Error())
	}
	return nil
}

// ListK8sResources lists k8s resources by GVR and converts them to the target type
func ListK8sResources(gvr schema.GroupVersionResource, target interface{}) error {
	var workloadUnstructured *unstructured.UnstructuredList
	var err error
	dynamicClient := K8s.GetDynamicClient()
	workloadUnstructured, err = dynamicClient.Resource(gvr).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		log.Error(err.Error())
		return err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(workloadUnstructured.UnstructuredContent(), target)
	if err != nil {
		log.Error(err.Error())
		return err
	}
	return nil
}
