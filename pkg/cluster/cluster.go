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
	"context"
	"fmt"
	"os"
	"path"
	"strings"
	"text/tabwriter"

	configv1beta1 "gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yaml2 "sigs.k8s.io/yaml"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/k8s"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type Options struct {
	root.Options
	Args      []string `json:"args"`
	File      string   `json:"file"`      // BKECluster YAML file
	NodesFile string   `json:"nodesFile"` // BKENodes YAML file
	Conf      string   `json:"Conf"`
	NtpServer string   `json:"ntpServer"`
}

const (
	// nsNamePartsCount 表示 namespace/name 格式拆分后的最小部分数
	nsNamePartsCount = 2
)

// marshalAndWriteClusterYAML 将 BKECluster 序列化并写入文件
func marshalAndWriteClusterYAML(bkeCluster *configv1beta1.BKECluster) (string, error) {
	if !strings.HasPrefix(bkeCluster.Namespace, "bke") {
		bkeCluster.Namespace = "bke-" + bkeCluster.Namespace
	}

	by, err := yaml2.Marshal(bkeCluster)
	if err != nil {
		log.BKEFormat(log.ERROR, err.Error())
		return "", err
	}

	if err := os.MkdirAll(path.Join(global.Workspace, "cluster"), utils.DefaultDirPermission); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to create cluster directory: %s", err.Error()))
	}

	bkeFile := path.Join(global.Workspace, "cluster", fmt.Sprintf("%s-%s.yaml", bkeCluster.Namespace, bkeCluster.Name))
	if err := os.WriteFile(bkeFile, by, utils.DefaultFilePermission); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("File generation failure %s", err.Error()))
		return "", err
	}
	return bkeFile, nil
}

var gvr = schema.GroupVersionResource{
	Group:    configv1beta1.GVK.Group,
	Version:  configv1beta1.GVK.Version,
	Resource: "bkeclusters",
}

func (op *Options) List() {
	bclusterlist := &configv1beta1.BKEClusterList{}
	err := global.ListK8sResources(gvr, bclusterlist)
	if err != nil {
		return
	}

	headers := []string{"namespace", "name", "endpoint", "master", "worker", "pause", "dryRun", "reset"}
	var rows [][]string
	for _, bc := range bclusterlist.Items {
		master := 0
		worker := 0
		line := []string{
			bc.Namespace,
			bc.Name,
			fmt.Sprintf("%s:%d", bc.Spec.ControlPlaneEndpoint.Host, bc.Spec.ControlPlaneEndpoint.Port),
			fmt.Sprintf("%d", master),
			fmt.Sprintf("%d", worker),
			fmt.Sprintf("%t", bc.Spec.Pause),
			fmt.Sprintf("%t", bc.Spec.DryRun),
			fmt.Sprintf("%t", bc.Spec.Reset),
		}
		rows = append(rows, line)
	}
	const tabPadding = 2 // 列之间的最小空格数，用于tabwriter对齐
	// 使用tabwriter输出表格
	w := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(w, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(w, strings.Join(row, "\t"))
	}
	err = w.Flush()
	if err != nil {
		fmt.Println("flush tablewriter failed:", err.Error())
	}
}

func (op *Options) Remove() {
	var workloadUnstructured *unstructured.Unstructured
	var err error
	ns := strings.Split(op.Args[0], "/")
	if len(ns) < nsNamePartsCount {
		log.Error("invalid argument format, expected namespace/name")
		return
	}
	dynamicClient := global.K8s.GetDynamicClient()
	workloadUnstructured, err = dynamicClient.Resource(gvr).Namespace(ns[0]).Get(context.TODO(), ns[1], metav1.GetOptions{})
	if err != nil {
		log.Error(err.Error())
		return
	}
	bcluster := &configv1beta1.BKECluster{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(workloadUnstructured.UnstructuredContent(), bcluster)
	if err != nil {
		log.Error(err.Error())
		return
	}
	bcluster.Spec.Reset = true

	t1, err := runtime.DefaultUnstructuredConverter.ToUnstructured(bcluster)
	if err != nil {
		log.Error(err.Error())
		return
	}
	m1 := &unstructured.Unstructured{
		Object: t1,
	}
	_, err = dynamicClient.Resource(gvr).Namespace(bcluster.Namespace).Update(context.TODO(), m1, metav1.UpdateOptions{})
	if err != nil {
		log.Error(err.Error())
		return
	}
}

func (op *Options) Scale() {
	// Parse BKECluster and BKENode resources from separate files
	resources, err := NewClusterResourcesFromFiles(op.File, op.NodesFile)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to load the configuration file. %v", err))
		return
	}

	bkeCluster := resources.BKECluster
	bkeNodes := resources.BKENodes

	bkeFile, err := marshalAndWriteClusterYAML(bkeCluster)
	if err != nil {
		return
	}

	// Write BKENodes YAML for scaling
	nodeFiles, err := marshalAndWriteNodeYAMLs(bkeCluster.Namespace, bkeNodes)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to write BKENode files: %v", err))
		return
	}

	log.BKEFormat(log.INFO, "Patch cluster-api yaml to the cluster")
	err = global.K8s.PatchYaml(bkeFile, map[string]string{})
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to patch bke-cluster, %v", err))
		return
	}

	for _, nodeFile := range nodeFiles {
		err = global.K8s.PatchYaml(nodeFile, map[string]string{})
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to patch bke-node, %v", err))
			return
		}
	}

	log.BKEFormat(log.INFO, fmt.Sprintf("Patched configuration (1 BKECluster + %d BKENodes)", len(bkeNodes)))
	log.BKEFormat(log.INFO, "Waiting for the cluster to start...")
	global.K8s.WatchEventByAnnotation(bkeCluster.Namespace)
}

func (op *Options) Log() {
	ns := strings.Split(op.Args[0], "/")
	global.K8s.WatchEventByAnnotation(ns[0])
}

// loadClusterConfig 加载集群配置
func loadClusterConfig(file string) (*configv1beta1.BKECluster, error) {
	conf := &configv1beta1.BKECluster{}
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	if err = yaml2.Unmarshal(yamlFile, conf); err != nil {
		return nil, err
	}
	if conf.Spec.ClusterConfig == nil {
		return nil, fmt.Errorf("the cluster configuration cannot be empty")
	}
	return conf, nil
}

// createKubeconfigSecret 创建 kubeconfig secret
func createKubeconfigSecret(namespace, name, confPath string) error {
	kubeConfig, err := os.ReadFile(confPath)
	if err != nil {
		return err
	}

	secret := corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name + "-kubeconfig",
		},
		Data: map[string][]byte{
			"value": kubeConfig,
		},
	}
	return global.K8s.CreateSecret(&secret)
}

func (op *Options) ExistsCluster() {
	conf, err := loadClusterConfig(op.File)
	if err != nil {
		log.Error(err.Error())
		return
	}

	if err = k8s.CreateNamespace(global.K8s, conf.Namespace); err != nil {
		log.Error(err.Error())
		return
	}

	if err = createKubeconfigSecret(conf.Namespace, conf.Name, op.Conf); err != nil {
		log.Error(err.Error())
		return
	}

	if err = global.K8s.InstallYaml(op.File, map[string]string{}, ""); err != nil {
		log.Error(err)
		return
	}
}
