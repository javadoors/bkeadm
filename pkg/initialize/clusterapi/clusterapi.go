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

package clusterapi

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/k8s"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

var (
	//go:embed cluster-api.yaml
	clusterAPI []byte
	//go:embed cluster-api-bke.yaml
	clusterAPIBKE []byte
	//go:embed webhook-secrets.yaml
	certManager []byte
)

func ensureK8sClient() error {
	if global.K8s != nil {
		return nil
	}
	var err error
	global.K8s, err = k8s.NewKubernetesClient("")
	return err
}

func writeClusterAPITemplates(tmplDir string) error {
	if err := os.MkdirAll(tmplDir, utils.DefaultDirPermission); err != nil {
		return err
	}

	files := map[string][]byte{
		filepath.Join(tmplDir, "cert-manager.yaml"):    certManager,
		filepath.Join(tmplDir, "cluster-api.yaml"):     clusterAPI,
		filepath.Join(tmplDir, "cluster-api-bke.yaml"): clusterAPIBKE,
	}

	for path, content := range files {
		if err := os.WriteFile(path, content, utils.DefaultFilePermission); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}
	return nil
}

func installClusterAPIWithRetry(yamlFile, repo string) error {
	for {
		time.Sleep(utils.DefaultMinCheckSeconds * time.Second)
		err := global.K8s.InstallYaml(yamlFile, map[string]string{"repo": repo}, "")
		if err == nil {
			return nil
		}
		log.BKEFormat(log.WARN, "Installation failed. Try again in 5 seconds")
		log.Debugf("err: %v", err)
	}
}

func waitForClusterAPIPodsRunning() error {
	client := global.K8s.GetClient()
	namespace := "cluster-system"

	for {
		time.Sleep(time.Duration(rand.IntnRange(utils.DefaultMinCheckSeconds, utils.DefaultMaxCheckSeconds)) * time.Second)
		log.BKEFormat(log.INFO, "Wait for the cluster-api container to running...")

		pods, err := client.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
		if err != nil {
			continue
		}

		if len(pods.Items) == 0 {
			continue
		}

		if areAllPodsRunning(pods.Items) {
			log.BKEFormat(log.INFO, "Cluster-api container running")
			return nil
		}
	}
}

func areAllPodsRunning(pods []corev1.Pod) bool {
	for _, pod := range pods {
		if pod.Status.Phase == corev1.PodRunning {
			continue
		}

		if len(pod.Status.ContainerStatuses) > 0 {
			lastContainer := pod.Status.ContainerStatuses[len(pod.Status.ContainerStatuses)-1]
			if lastContainer.State.Waiting != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("Container %s status %s", pod.Name, lastContainer.State.Waiting.Reason))
			}
		}
		return false
	}
	return true
}

func DeployClusterAPI(repo, manifestsVersion, providerVersion string) error {
	if err := ensureK8sClient(); err != nil {
		return err
	}

	tmplDir := filepath.Join(global.Workspace, "tmpl")
	if err := writeClusterAPITemplates(tmplDir); err != nil {
		return err
	}

	log.BKEFormat(log.INFO, "Install Certificate Management...")
	certManagerFile := filepath.Join(tmplDir, "cert-manager.yaml")
	if err := global.K8s.InstallYaml(certManagerFile, map[string]string{"repo": repo}, ""); err != nil {
		return err
	}

	log.BKEFormat(log.INFO, "Install the Cluster API...")
	clusterAPIFile := filepath.Join(tmplDir, "cluster-api.yaml")
	if err := installClusterAPIWithRetry(clusterAPIFile, repo); err != nil {
		return err
	}

	clusterAPIBKEFile := filepath.Join(tmplDir, "cluster-api-bke.yaml")
	params := map[string]string{
		"repo":             repo,
		"manifestsVersion": manifestsVersion,
		"providerVersion":  providerVersion,
	}
	if err := global.K8s.InstallYaml(clusterAPIBKEFile, params, ""); err != nil {
		return err
	}

	return waitForClusterAPIPodsRunning()
}
