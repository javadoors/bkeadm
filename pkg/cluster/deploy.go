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

package cluster

import (
	"fmt"
	"os"
	"path"
	"strings"

	configv1beta1 "gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yaml2 "sigs.k8s.io/yaml"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func (op *Options) Cluster() {
	// Parse BKECluster and BKENode resources from separate files
	resources, err := NewClusterResourcesFromFiles(op.File, op.NodesFile)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to load the configuration file. %v", err))
		return
	}

	bkeCluster := resources.BKECluster
	bkeNodes := resources.BKENodes

	if len(op.NtpServer) > 0 {
		bkeCluster.Spec.ClusterConfig.Cluster.NTPServer = op.NtpServer
	}

	// Normalize namespace
	if !strings.HasPrefix(bkeCluster.Namespace, "bke") {
		bkeCluster.Namespace = "bke-" + bkeCluster.Namespace
	}

	// Write BKECluster YAML
	bkefile, err := marshalAndWriteClusterYAML(bkeCluster)
	if err != nil {
		return
	}

	// Write BKENodes YAML
	nodeFiles, err := marshalAndWriteNodeYAMLs(bkeCluster.Namespace, bkeNodes)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to write BKENode files: %v", err))
		return
	}

	namespace := corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: bkeCluster.Namespace,
		},
		Spec:   corev1.NamespaceSpec{},
		Status: corev1.NamespaceStatus{},
	}

	err = global.K8s.CreateNamespace(&namespace)
	if err != nil {
		log.BKEFormat(log.WARN, err.Error())
	}

	log.BKEFormat(log.INFO, "Submit cluster-api yaml to the cluster")

	// Install BKENodes first (webhook needs nodes to set default ControlPlaneEndpoint)
	for _, nodeFile := range nodeFiles {
		err = global.K8s.InstallYaml(nodeFile, map[string]string{}, "")
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install bke-node, %v", err))
			return
		}
	}

	// Install BKECluster after nodes exist
	err = global.K8s.InstallYaml(bkefile, map[string]string{}, "")
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install bke-cluster, %v", err))
		return
	}

	log.BKEFormat(log.INFO, fmt.Sprintf("Submit the configuration to the cluster (1 BKECluster + %d BKENodes)", len(bkeNodes)))

	log.BKEFormat(log.INFO, "Waiting for the cluster to start...")
	global.K8s.WatchEventByAnnotation(bkeCluster.Namespace)
}

// marshalAndWriteNodeYAMLs writes BKENode resources to YAML files
func marshalAndWriteNodeYAMLs(namespace string, nodes []configv1beta1.BKENode) ([]string, error) {
	var files []string

	if err := os.MkdirAll(path.Join(global.Workspace, "cluster"), utils.DefaultDirPermission); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to create cluster directory: %s", err.Error()))
	}

	for i := range nodes {
		node := &nodes[i]
		// Ensure namespace is set
		if node.Namespace == "" {
			node.Namespace = namespace
		}

		by, err := yaml2.Marshal(node)
		if err != nil {
			return nil, err
		}

		nodeFile := path.Join(global.Workspace, "cluster", fmt.Sprintf("%s-%s-node.yaml", namespace, node.Name))
		if err := os.WriteFile(nodeFile, by, utils.DefaultFilePermission); err != nil {
			return nil, err
		}
		files = append(files, nodeFile)
	}

	return files, nil
}
