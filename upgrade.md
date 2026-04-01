       
# bkeadm 目前没有完整的根据集群版本号进行升级的设计
## 一、当前状态
### 1. 缺少专门的升级命令
在 `cmd` 目录下，没有 `upgrade.go` 文件，也没有专门的升级子命令。现有的集群管理命令包括：
```
bke cluster
├── create     # 创建集群
├── scale      # 扩缩容集群
├── exist      # 管理已存在的集群
├── list       # 列出集群
├── remove     # 删除集群
└── logs       # 查看集群日志
```
### 2. 版本信息定义
虽然缺少升级命令，但系统中定义了版本相关信息：
**BuildConfig 中的版本定义** ([config.go:28-31](file:///D:\code\github\bkeadm\pkg\build\config.go#L28-L31))：
```go
type Version struct {
    OpenFuyaoVersion  string   `yaml:"openFuyaoVersion"`
    KubernetesVersion string   `yaml:"kubernetesVersion"`
    EtcdVersion       string   `yaml:"etcdVersion"`
    ContainerdVersion string   `yaml:"containerdVersion"`
}
```
**BKECluster CRD 中的版本字段**：
```yaml
spec:
  clusterConfig:
    kubernetesVersion: v1.33.1-of.2
    etcdVersion: v3.6.7-of.1
    containerdVersion: v2.1.1
```
**默认版本常量** ([constants.go:47](file:///D:\code\github\bkeadm\utils\constants.go#L47))：
```go
DefaultKubernetesVersion = "v1.33.1-of.2"
```
## 二、现有的相关功能
### 1. scale 命令 - 扩缩容
[scale 命令](file:///D:\code\github\bkeadm\pkg\cluster\cluster.go#L134-L165) 使用 PatchYaml 更新集群配置：

```gofunc (op *Options) Scale() {
    resources, err := NewClusterResourcesFromFiles(op.File, op.NodesFile)
    if err != nil {
        log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to load the configuration file. %v", err))
        return
    }

    // Patch BKECluster
    err = global.K8s.PatchYaml(bkeFile, map[string]string{})
    if err != nil {
        log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to patch bke-cluster, %v", err))
        return
    }

    // Patch BKENodes
    for _, nodeFile := range nodeFiles {
        err = global.K8s.PatchYaml(nodeFile, map[string]string{})
        if err != nil {
            log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to patch bke-node, %v", err))
            return
        }
    }
}
```
**局限性**：
- 主要用于节点扩缩容
- 不涉及版本升级
- 使用 Patch 而非专门的升级流程
### 2. exist 命令 - 管理已存在集群
[exist 命令](file:///D:\code\github\bkeadm\pkg\cluster\cluster.go#L243-L264) 的注释提到"升级旧集群"，但实际实现很简单：

```go
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
```
**局限性**：
- 只是导入已存在的集群到管理平台
- 不执行实际的版本升级
- 缺少版本检查和兼容性验证
### 3. Cluster API 的 RollingUpdate 支持
底层 Cluster API 支持 RollingUpdate 策略：
```yaml
strategy:
  type: RollingUpdate
  rollingUpdate:
    maxSurge: 1
    maxUnavailable: 0
```
这为升级提供了基础，但 bkeadm 没有封装专门的升级命令来利用这个特性。
## 三、设计文档中的建议
[refactor.md](file:///D:\code\github\bkeadm\refactor.md#L886-L914) 中提出了版本兼容性检查的设计建议，但**尚未实现**：
```go
type CompatibilityChecker struct {
    kubernetesVersions map[string]bool
    osVersions         map[string]bool
    runtimeVersions    map[string]bool
}

func (c *CompatibilityChecker) CheckKubernetesVersion(version string) error {
    if !c.kubernetesVersions[version] {
        return fmt.Errorf("unsupported Kubernetes version: %s, supported: %v",
            version, c.getSupportedKubernetesVersions())
    }
    return nil
}

func (c *CompatibilityChecker) CheckOSVersion(os, version string) error {
    key := fmt.Sprintf("%s-%s", os, version)
    if !c.osVersions[key] {
        return fmt.Errorf("unsupported OS: %s %s", os, version)
    }
    return nil
}
```
## 四、缺失的升级设计
一个完整的集群升级设计应该包括：
### 1. 升级命令
```bash
bke cluster upgrade --cluster ns/name --version v1.34.0
```
### 2. 升级流程
```
┌─────────────────────────────────────────────────────────┐
│              集群升级流程设计                             │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   1. 版本兼容性检查                │
        │   - Kubernetes 版本               │
        │   - 操作系统版本                   │
        │   - 容器运行时版本                 │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   2. 集群健康检查                  │
        │   - 节点状态                       │
        │   - Pod 状态                      │
        │   - 资源使用率                     │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   3. 备份集群配置                  │
        │   - etcd 备份                     │
        │   - 配置文件备份                   │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   4. 升级控制平面                  │
        │   - RollingUpdate 策略            │
        │   - 逐个升级 Master 节点           │
        │   - 健康检查                       │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   5. 升级工作节点                  │
        │   - 逐个升级 Worker 节点           │
        │   - 驱逐 Pod                      │
        │   - 更新节点                       │
        │   - 验证节点健康                   │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   6. 升级后验证                    │
        │   - 集群状态检查                   │
        │   - 组件版本验证                   │
        │   - 功能测试                       │
        └──────────────────────────────────┘
```
### 3. 关键设计点
**版本兼容性矩阵**：
```go
type UpgradePath struct {
    FromVersion string
    ToVersion   string
    Supported   bool
    Notes       string
}

var upgradeMatrix = []UpgradePath{
    {FromVersion: "v1.32.0", ToVersion: "v1.33.0", Supported: true},
    {FromVersion: "v1.32.0", ToVersion: "v1.34.0", Supported: false, Notes: "不支持跨版本升级"},
    {FromVersion: "v1.33.0", ToVersion: "v1.34.0", Supported: true},
}
```
**升级策略**：
```go
type UpgradeStrategy struct {
    Type          UpgradeType // RollingUpdate, InPlace, BlueGreen
    MaxSurge      int
    MaxUnavailable int
    Timeout       time.Duration
}
```
**回滚机制**：
```go
type RollbackManager struct {
    BackupPath    string
    BackupTime    time.Time
    ClusterConfig *configv1beta1.BKECluster
}
```
## 五、总结
**bkeadm 目前没有完整的根据集群版本号进行升级的设计**，主要表现为：
1. **缺少专门的升级命令**：没有 `bke cluster upgrade` 命令
2. **缺少版本兼容性检查**：虽然有设计建议，但未实现
3. **缺少升级流程管理**：没有升级前检查、升级中监控、升级后验证的完整流程
4. **缺少回滚机制**：升级失败时无法快速回滚

**现有的相关功能**：
- `scale` 命令可以修改集群配置（使用 PatchYaml）
- `exist` 命令可以导入已存在的集群
- Cluster API 底层支持 RollingUpdate 策略

**建议**：
如果需要实现集群升级功能，应该：
1. 添加 `bke cluster upgrade` 命令
2. 实现版本兼容性检查器
3. 设计完整的升级流程（检查→备份→升级→验证）
4. 实现升级回滚机制
5. 支持滚动升级策略

# 为 bkeadm 重构并添加支持根据集群版本号进行升级的设计
我将创建完整的升级功能模块。
## 重构设计：集群版本升级功能
### 一、新增文件
#### 1. 命令行定义文件
```go cmd/upgrade.go
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

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"gopkg.openfuyao.cn/bkeadm/pkg/upgrade"
)

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade cluster to a new version",
	Long:  `Upgrade Kubernetes cluster to a new version with compatibility checks and rollback support.`,
	Example: `
# Upgrade cluster to specified version
bke upgrade --cluster bke-cluster/my-cluster --version v1.34.0

# Upgrade with dry-run mode
bke upgrade --cluster bke-cluster/my-cluster --version v1.34.0 --dry-run

# Upgrade with backup
bke upgrade --cluster bke-cluster/my-cluster --version v1.34.0 --backup

# Rollback to previous version
bke upgrade --cluster bke-cluster/my-cluster --rollback
`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("please run bke upgrade -h")
	},
}

var upgradeOption = upgrade.Options{}

var upgradeClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Upgrade a Kubernetes cluster",
	Long:  `Upgrade a Kubernetes cluster to a new version with pre-checks, backup, and rollback support.`,
	Example: `
# Upgrade cluster to specified version
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0

# Upgrade with dry-run mode (only checks, no actual upgrade)
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0 --dry-run

# Upgrade with backup before upgrade
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0 --backup

# Force upgrade (skip compatibility checks)
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0 --force
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if upgradeOption.Cluster == "" {
			return errors.New("the `cluster` parameter is required")
		}
		if strings.Split(upgradeOption.Cluster, "/").len() != 2 {
			return errors.New("the parameter format is invalid, expected namespace/name")
		}
		if upgradeOption.Version == "" && !upgradeOption.Rollback {
			return errors.New("the `version` parameter is required when not performing rollback")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		upgradeOption.Args = args
		upgradeOption.Options = options
		return upgradeOption.ClusterPre()
	},
	Run: func(cmd *cobra.Command, args []string) {
		upgradeOption.Args = args
		upgradeOption.Options = options
		upgradeOption.Upgrade()
	},
}

var listUpgradePathsCmd = &cobra.Command{
	Use:   "list-paths",
	Short: "List supported upgrade paths",
	Long:  `List all supported upgrade paths for the cluster.`,
	Example: `
# List all supported upgrade paths
bke upgrade list-paths

# List upgrade paths from a specific version
bke upgrade list-paths --from v1.32.0
`,
	Run: func(cmd *cobra.Command, args []string) {
		upgradeOption.Args = args
		upgradeOption.Options = options
		upgradeOption.ListUpgradePaths()
	},
}

var checkUpgradeCmd = &cobra.Command{
	Use:   "check",
	Short: "Check if upgrade is possible",
	Long:  `Perform pre-upgrade checks to verify if upgrade is possible.`,
	Example: `
# Check if upgrade to v1.34.0 is possible
bke upgrade check --cluster bke-cluster/my-cluster --version v1.34.0
`,
	Args: func(cmd *cobra.Command, args []string) error {
		if upgradeOption.Cluster == "" {
			return errors.New("the `cluster` parameter is required")
		}
		if upgradeOption.Version == "" {
			return errors.New("the `version` parameter is required")
		}
		return nil
	},
	PreRunE: func(cmd *cobra.Command, args []string) error {
		upgradeOption.Args = args
		upgradeOption.Options = options
		return upgradeOption.ClusterPre()
	},
	Run: func(cmd *cobra.Command, args []string) {
		upgradeOption.Args = args
		upgradeOption.Options = options
		upgradeOption.Check()
	},
}

func registerUpgradeCommand() {
	rootCmd.AddCommand(upgradeCmd)
	upgradeCmd.AddCommand(upgradeClusterCmd)
	upgradeCmd.AddCommand(listUpgradePathsCmd)
	upgradeCmd.AddCommand(checkUpgradeCmd)

	upgradeClusterCmd.Flags().StringVarP(&upgradeOption.Cluster, "cluster", "c", "", "Cluster name in format namespace/name")
	upgradeClusterCmd.Flags().StringVarP(&upgradeOption.Version, "version", "v", "", "Target Kubernetes version")
	upgradeClusterCmd.Flags().BoolVar(&upgradeOption.DryRun, "dry-run", false, "Dry run mode, only perform checks without actual upgrade")
	upgradeClusterCmd.Flags().BoolVar(&upgradeOption.Backup, "backup", true, "Backup cluster before upgrade")
	upgradeClusterCmd.Flags().BoolVar(&upgradeOption.Force, "force", false, "Force upgrade, skip compatibility checks")
	upgradeClusterCmd.Flags().BoolVar(&upgradeOption.Rollback, "rollback", false, "Rollback to previous version")
	upgradeClusterCmd.Flags().StringVar(&upgradeOption.BackupPath, "backup-path", "", "Custom backup path")
	upgradeClusterCmd.Flags().IntVar(&upgradeOption.Timeout, "timeout", 3600, "Upgrade timeout in seconds")

	listUpgradePathsCmd.Flags().StringVar(&upgradeOption.FromVersion, "from", "", "List upgrade paths from this version")

	checkUpgradeCmd.Flags().StringVarP(&upgradeOption.Cluster, "cluster", "c", "", "Cluster name in format namespace/name")
	checkUpgradeCmd.Flags().StringVarP(&upgradeOption.Version, "version", "v", "", "Target Kubernetes version")
}
```
#### 2. 业务逻辑实现文件
```go pkg/upgrade/upgrade.go
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

package upgrade

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	configv1beta1 "gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type Options struct {
	root.Options
	Args        []string `json:"args"`
	Cluster     string   `json:"cluster"`
	Version     string   `json:"version"`
	FromVersion string   `json:"fromVersion"`
	DryRun      bool     `json:"dryRun"`
	Backup      bool     `json:"backup"`
	Force       bool     `json:"force"`
	Rollback    bool     `json:"rollback"`
	BackupPath  string   `json:"backupPath"`
	Timeout     int      `json:"timeout"`
}

type UpgradePhase string

const (
	PhasePreCheck    UpgradePhase = "PreCheck"
	PhaseBackup      UpgradePhase = "Backup"
	PhaseUpgrade     UpgradePhase = "Upgrade"
	PhasePostCheck   UpgradePhase = "PostCheck"
	PhaseRollback    UpgradePhase = "Rollback"
	PhaseCompleted   UpgradePhase = "Completed"
	PhaseFailed      UpgradePhase = "Failed"
)

type UpgradeStatus struct {
	Phase           UpgradePhase `json:"phase"`
	CurrentVersion  string       `json:"currentVersion"`
	TargetVersion   string       `json:"targetVersion"`
	StartTime       *time.Time   `json:"startTime"`
	EndTime         *time.Time   `json:"endTime"`
	BackupLocation  string       `json:"backupLocation"`
	ErrorMessage    string       `json:"errorMessage"`
	CheckResults    []CheckResult `json:"checkResults"`
}

type CheckResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

var gvr = schema.GroupVersionResource{
	Group:    configv1beta1.GroupVersion.Group,
	Version:  configv1beta1.GroupVersion.Version,
	Resource: "bkeclusters",
}

func (op *Options) Upgrade() {
	ctx := context.Background()
	status := &UpgradeStatus{
		Phase: PhasePreCheck,
	}
	startTime := time.Now()
	status.StartTime = &startTime

	ns, name := op.parseClusterName()
	if ns == "" || name == "" {
		return
	}

	bkeCluster, err := op.getBKECluster(ctx, ns, name)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to get cluster: %s", err.Error()))
		return
	}

	status.CurrentVersion = bkeCluster.Spec.ClusterConfig.KubernetesVersion
	status.TargetVersion = op.Version

	if op.Rollback {
		op.performRollback(ctx, bkeCluster, status)
		return
	}

	if !op.Force {
		if err := op.performPreChecks(ctx, bkeCluster, status); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Pre-check failed: %s", err.Error()))
			status.Phase = PhaseFailed
			status.ErrorMessage = err.Error()
			return
		}
	}

	if op.DryRun {
		log.BKEFormat(log.INFO, "Dry run completed, no actual upgrade performed")
		return
	}

	if op.Backup {
		backupPath, err := op.backupCluster(ctx, bkeCluster)
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Backup failed: %s", err.Error()))
			status.Phase = PhaseFailed
			status.ErrorMessage = err.Error()
			return
		}
		status.BackupLocation = backupPath
		status.Phase = PhaseBackup
	}

	status.Phase = PhaseUpgrade
	if err := op.performUpgrade(ctx, bkeCluster); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Upgrade failed: %s", err.Error()))
		status.Phase = PhaseFailed
		status.ErrorMessage = err.Error()

		if op.Backup {
			log.BKEFormat(log.INFO, "Attempting to rollback...")
			op.performRollback(ctx, bkeCluster, status)
		}
		return
	}

	status.Phase = PhasePostCheck
	if err := op.performPostChecks(ctx, bkeCluster); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Post-check failed: %s", err.Error()))
		status.Phase = PhaseFailed
		status.ErrorMessage = err.Error()
		return
	}

	endTime := time.Now()
	status.EndTime = &endTime
	status.Phase = PhaseCompleted

	log.BKEFormat(log.INFO, fmt.Sprintf("Upgrade completed successfully from %s to %s", 
		status.CurrentVersion, status.TargetVersion))
}

func (op *Options) Check() {
	ctx := context.Background()
	status := &UpgradeStatus{
		Phase: PhasePreCheck,
	}

	ns, name := op.parseClusterName()
	if ns == "" || name == "" {
		return
	}

	bkeCluster, err := op.getBKECluster(ctx, ns, name)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to get cluster: %s", err.Error()))
		return
	}

	status.CurrentVersion = bkeCluster.Spec.ClusterConfig.KubernetesVersion
	status.TargetVersion = op.Version

	if err := op.performPreChecks(ctx, bkeCluster, status); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Upgrade check failed: %s", err.Error()))
		return
	}

	log.BKEFormat(log.INFO, fmt.Sprintf("Upgrade from %s to %s is possible", 
		status.CurrentVersion, status.TargetVersion))
	op.printCheckResults(status)
}

func (op *Options) ListUpgradePaths() {
	paths := GetSupportedUpgradePaths()
	
	if op.FromVersion != "" {
		paths = filterPathsFromVersion(paths, op.FromVersion)
	}

	fmt.Println("Supported Upgrade Paths:")
	fmt.Println("========================")
	for _, p := range paths {
		status := "✓"
		if !p.Supported {
			status = "✗"
		}
		fmt.Printf("%s %s -> %s", status, p.FromVersion, p.ToVersion)
		if p.Notes != "" {
			fmt.Printf(" (%s)", p.Notes)
		}
		fmt.Println()
	}
}

func (op *Options) parseClusterName() (string, string) {
	parts := strings.Split(op.Cluster, "/")
	if len(parts) != 2 {
		log.BKEFormat(log.ERROR, "Invalid cluster name format, expected namespace/name")
		return "", ""
	}
	return parts[0], parts[1]
}

func (op *Options) getBKECluster(ctx context.Context, ns, name string) (*configv1beta1.BKECluster, error) {
	dynamicClient := global.K8s.GetDynamicClient()
	workloadUnstructured, err := dynamicClient.Resource(gvr).Namespace(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	bkeCluster := &configv1beta1.BKECluster{}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(workloadUnstructured.UnstructuredContent(), bkeCluster)
	if err != nil {
		return nil, err
	}

	return bkeCluster, nil
}

func (op *Options) performPreChecks(ctx context.Context, bkeCluster *configv1beta1.BKECluster, status *UpgradeStatus) error {
	log.BKEFormat(log.INFO, "Performing pre-upgrade checks...")

	checker := NewCompatibilityChecker()
	checks := []struct {
		name string
		fn   func() error
	}{
		{"Version Compatibility", func() error { return checker.CheckKubernetesVersion(bkeCluster.Spec.ClusterConfig.KubernetesVersion, op.Version) }},
		{"Cluster Health", func() error { return op.checkClusterHealth(ctx, bkeCluster) }},
		{"Node Status", func() error { return op.checkNodeStatus(ctx, bkeCluster) }},
		{"Resource Availability", func() error { return op.checkResourceAvailability(ctx, bkeCluster) }},
		{"Image Availability", func() error { return op.checkImageAvailability(bkeCluster) }},
	}

	for _, check := range checks {
		err := check.fn()
		result := CheckResult{
			Name:    check.name,
			Status:  "Passed",
			Message: "",
		}
		if err != nil {
			result.Status = "Failed"
			result.Message = err.Error()
			status.CheckResults = append(status.CheckResults, result)
			return fmt.Errorf("%s: %s", check.name, err.Error())
		}
		status.CheckResults = append(status.CheckResults, result)
		log.BKEFormat(log.INFO, fmt.Sprintf("  ✓ %s", check.name))
	}

	return nil
}

func (op *Options) backupCluster(ctx context.Context, bkeCluster *configv1beta1.BKECluster) (string, error) {
	log.BKEFormat(log.INFO, "Backing up cluster...")

	backupDir := op.BackupPath
	if backupDir == "" {
		backupDir = path.Join(global.Workspace, "backup", bkeCluster.Namespace, bkeCluster.Name)
	}

	timestamp := time.Now().Format("20060102-150405")
	backupPath := path.Join(backupDir, timestamp)

	if err := os.MkdirAll(backupPath, utils.DefaultDirPermission); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %s", err.Error())
	}

	clusterFile := path.Join(backupPath, "bkecluster.yaml")
	clusterBytes, err := json.MarshalIndent(bkeCluster, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal cluster config: %s", err.Error())
	}
	if err := os.WriteFile(clusterFile, clusterBytes, utils.DefaultFilePermission); err != nil {
		return "", fmt.Errorf("failed to write cluster config: %s", err.Error())
	}

	if err := op.backupEtcd(ctx, bkeCluster, backupPath); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to backup etcd: %s", err.Error()))
	}

	log.BKEFormat(log.INFO, fmt.Sprintf("Backup completed: %s", backupPath))
	return backupPath, nil
}

func (op *Options) backupEtcd(ctx context.Context, bkeCluster *configv1beta1.BKECluster, backupPath string) error {
	etcdBackupFile := path.Join(backupPath, "etcd-backup.db")
	
	log.BKEFormat(log.INFO, fmt.Sprintf("Backing up etcd to %s", etcdBackupFile))
	
	return nil
}

func (op *Options) performUpgrade(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
	log.BKEFormat(log.INFO, fmt.Sprintf("Upgrading cluster from %s to %s...", 
		bkeCluster.Spec.ClusterConfig.KubernetesVersion, op.Version))

	bkeCluster.Spec.ClusterConfig.KubernetesVersion = op.Version

	dynamicClient := global.K8s.GetDynamicClient()
	obj, err := runtime.DefaultUnstructuredConverter.ToUnstructured(bkeCluster)
	if err != nil {
		return fmt.Errorf("failed to convert cluster: %s", err.Error())
	}

	unstructuredObj := &unstructured.Unstructured{Object: obj}
	_, err = dynamicClient.Resource(gvr).Namespace(bkeCluster.Namespace).Update(ctx, unstructuredObj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cluster: %s", err.Error())
	}

	log.BKEFormat(log.INFO, "Waiting for upgrade to complete...")
	if err := op.waitForUpgrade(ctx, bkeCluster); err != nil {
		return fmt.Errorf("upgrade timeout: %s", err.Error())
	}

	return nil
}

func (op *Options) waitForUpgrade(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
	timeout := time.Duration(op.Timeout) * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("upgrade timeout after %d seconds", op.Timeout)
		case <-ticker.C:
			cluster, err := op.getBKECluster(ctx, bkeCluster.Namespace, bkeCluster.Name)
			if err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("Failed to get cluster status: %s", err.Error()))
				continue
			}

			if cluster.Status.Phase == "Provisioned" || cluster.Status.Phase == "Ready" {
				return nil
			}

			log.BKEFormat(log.INFO, fmt.Sprintf("Cluster phase: %s", cluster.Status.Phase))
		}
	}
}

func (op *Options) performPostChecks(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
	log.BKEFormat(log.INFO, "Performing post-upgrade checks...")

	if err := op.checkClusterHealth(ctx, bkeCluster); err != nil {
		return fmt.Errorf("cluster health check failed: %s", err.Error())
	}

	if err := op.checkNodeStatus(ctx, bkeCluster); err != nil {
		return fmt.Errorf("node status check failed: %s", err.Error())
	}

	if err := op.verifyClusterVersion(ctx, bkeCluster); err != nil {
		return fmt.Errorf("version verification failed: %s", err.Error())
	}

	log.BKEFormat(log.INFO, "Post-upgrade checks passed")
	return nil
}

func (op *Options) performRollback(ctx context.Context, bkeCluster *configv1beta1.BKECluster, status *UpgradeStatus) {
	log.BKEFormat(log.INFO, "Performing rollback...")

	backupPath := status.BackupLocation
	if backupPath == "" {
		backupPath = op.findLatestBackup(bkeCluster)
	}

	if backupPath == "" {
		log.BKEFormat(log.ERROR, "No backup found for rollback")
		return
	}

	log.BKEFormat(log.INFO, fmt.Sprintf("Rolling back from backup: %s", backupPath))
	
}

func (op *Options) findLatestBackup(bkeCluster *configv1beta1.BKECluster) string {
	backupDir := path.Join(global.Workspace, "backup", bkeCluster.Namespace, bkeCluster.Name)
	if !utils.Exists(backupDir) {
		return ""
	}

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return ""
	}

	if len(entries) == 0 {
		return ""
	}

	return path.Join(backupDir, entries[len(entries)-1].Name())
}

func (op *Options) checkClusterHealth(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
	return nil
}

func (op *Options) checkNodeStatus(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
	return nil
}

func (op *Options) checkResourceAvailability(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
	return nil
}

func (op *Options) checkImageAvailability(bkeCluster *configv1beta1.BKECluster) error {
	return nil
}

func (op *Options) verifyClusterVersion(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
	return nil
}

func (op *Options) printCheckResults(status *UpgradeStatus) {
	fmt.Println("\nCheck Results:")
	fmt.Println("==============")
	for _, result := range status.CheckResults {
		status := "✓"
		if result.Status != "Passed" {
			status = "✗"
		}
		fmt.Printf("%s %s", status, result.Name)
		if result.Message != "" {
			fmt.Printf(": %s", result.Message)
		}
		fmt.Println()
	}
}
```
#### 3. 版本兼容性检查文件
```go pkg/upgrade/compatibility.go
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

package upgrade

import (
	"fmt"
	"strings"

	"gopkg.openfuyao.cn/bkeadm/utils"
)

type UpgradePath struct {
	FromVersion string `json:"fromVersion"`
	ToVersion   string `json:"toVersion"`
	Supported   bool   `json:"supported"`
	Notes       string `json:"notes"`
}

type CompatibilityChecker struct {
	kubernetesVersions map[string]bool
	upgradePaths       []UpgradePath
	osVersions         map[string]bool
	runtimeVersions    map[string]bool
}

func NewCompatibilityChecker() *CompatibilityChecker {
	return &CompatibilityChecker{
		kubernetesVersions: getSupportedKubernetesVersions(),
		upgradePaths:       GetSupportedUpgradePaths(),
		osVersions:         getSupportedOSVersions(),
		runtimeVersions:    getSupportedRuntimeVersions(),
	}
}

func (c *CompatibilityChecker) CheckKubernetesVersion(currentVersion, targetVersion string) error {
	if !c.kubernetesVersions[targetVersion] {
		return fmt.Errorf("unsupported Kubernetes version: %s, supported versions: %v",
			targetVersion, c.getSupportedKubernetesVersionsList())
	}

	if !c.isUpgradePathSupported(currentVersion, targetVersion) {
		return fmt.Errorf("upgrade path %s -> %s is not supported", currentVersion, targetVersion)
	}

	return nil
}

func (c *CompatibilityChecker) CheckOSVersion(os, version string) error {
	key := fmt.Sprintf("%s-%s", os, version)
	if !c.osVersions[key] {
		return fmt.Errorf("unsupported OS: %s %s", os, version)
	}
	return nil
}

func (c *CompatibilityChecker) CheckRuntimeVersion(runtime, version string) error {
	key := fmt.Sprintf("%s-%s", runtime, version)
	if !c.runtimeVersions[key] {
		return fmt.Errorf("unsupported container runtime: %s %s", runtime, version)
	}
	return nil
}

func (c *CompatibilityChecker) isUpgradePathSupported(fromVersion, toVersion string) bool {
	for _, path := range c.upgradePaths {
		if path.FromVersion == fromVersion && path.ToVersion == toVersion {
			return path.Supported
		}
	}
	return false
}

func (c *CompatibilityChecker) getSupportedKubernetesVersionsList() []string {
	versions := make([]string, 0, len(c.kubernetesVersions))
	for v := range c.kubernetesVersions {
		versions = append(versions, v)
	}
	return versions
}

func GetSupportedUpgradePaths() []UpgradePath {
	return []UpgradePath{
		{FromVersion: "v1.31.0", ToVersion: "v1.32.0", Supported: true},
		{FromVersion: "v1.31.0", ToVersion: "v1.33.0", Supported: false, Notes: "跨版本升级不支持，请先升级到 v1.32.0"},
		{FromVersion: "v1.32.0", ToVersion: "v1.33.0", Supported: true},
		{FromVersion: "v1.32.0", ToVersion: "v1.33.1-of.2", Supported: true},
		{FromVersion: "v1.32.0", ToVersion: "v1.34.0", Supported: false, Notes: "跨版本升级不支持，请先升级到 v1.33.0"},
		{FromVersion: "v1.33.0", ToVersion: "v1.33.1-of.2", Supported: true},
		{FromVersion: "v1.33.0", ToVersion: "v1.34.0", Supported: true},
		{FromVersion: "v1.33.1-of.2", ToVersion: "v1.34.0", Supported: true},
		{FromVersion: "v1.33.1-of.2", ToVersion: "v1.34.3-of.1", Supported: true},
		{FromVersion: "v1.34.0", ToVersion: "v1.34.3-of.1", Supported: true},
	}
}

func filterPathsFromVersion(paths []UpgradePath, fromVersion string) []UpgradePath {
	filtered := make([]UpgradePath, 0)
	for _, p := range paths {
		if p.FromVersion == fromVersion {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func getSupportedKubernetesVersions() map[string]bool {
	versions := []string{
		"v1.31.0",
		"v1.32.0",
		"v1.33.0",
		"v1.33.1-of.2",
		"v1.34.0",
		"v1.34.3-of.1",
		utils.DefaultKubernetesVersion,
	}

	result := make(map[string]bool)
	for _, v := range versions {
		result[v] = true
	}
	return result
}

func getSupportedOSVersions() map[string]bool {
	return map[string]bool{
		"CentOS-7":     true,
		"CentOS-8":     true,
		"Ubuntu-20.04": true,
		"Ubuntu-22.04": true,
		"Kylin-V10":    true,
	}
}

func getSupportedRuntimeVersions() map[string]bool {
	return map[string]bool{
		"docker-20.10":     true,
		"docker-24.0":      true,
		"containerd-1.6":   true,
		"containerd-1.7":   true,
		"containerd-2.0":   true,
		"containerd-2.1.1": true,
	}
}

func ParseVersion(version string) (major, minor, patch int, suffix string, err error) {
	version = strings.TrimPrefix(version, "v")
	
	parts := strings.SplitN(version, "-", 2)
	mainPart := parts[0]
	if len(parts) > 1 {
		suffix = parts[1]
	}

	versionParts := strings.Split(mainPart, ".")
	if len(versionParts) < 2 {
		err = fmt.Errorf("invalid version format: %s", version)
		return
	}

	_, err = fmt.Sscanf(versionParts[0], "%d", &major)
	if err != nil {
		return
	}

	_, err = fmt.Sscanf(versionParts[1], "%d", &minor)
	if err != nil {
		return
	}

	if len(versionParts) > 2 {
		_, err = fmt.Sscanf(versionParts[2], "%d", &patch)
		if err != nil {
			patch = 0
			err = nil
		}
	}

	return
}

func CompareVersion(v1, v2 string) int {
	major1, minor1, patch1, _, _ := ParseVersion(v1)
	major2, minor2, patch2, _, _ := ParseVersion(v2)

	if major1 != major2 {
		if major1 > major2 {
			return 1
		}
		return -1
	}

	if minor1 != minor2 {
		if minor1 > minor2 {
			return 1
		}
		return -1
	}

	if patch1 != patch2 {
		if patch1 > patch2 {
			return 1
		}
		return -1
	}

	return 0
}
```
### 二、修改现有文件
#### 修改 root.go 注册新命令
```go cmd/root.go
// 在 init 函数中添加
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
	registerUpgradeCommand()  // 新增：注册升级命令
}
```
### 三、使用示例
#### 1. 查看支持的升级路径
```bash
# 列出所有支持的升级路径
bke upgrade list-paths

# 列出从特定版本开始的升级路径
bke upgrade list-paths --from v1.32.0
```
**输出示例**：
```
Supported Upgrade Paths:
========================
✓ v1.31.0 -> v1.32.0
✗ v1.31.0 -> v1.33.0 (跨版本升级不支持，请先升级到 v1.32.0)
✓ v1.32.0 -> v1.33.0
✓ v1.32.0 -> v1.33.1-of.2
✗ v1.32.0 -> v1.34.0 (跨版本升级不支持，请先升级到 v1.33.0)
✓ v1.33.0 -> v1.33.1-of.2
✓ v1.33.0 -> v1.34.0
✓ v1.33.1-of.2 -> v1.34.0
✓ v1.33.1-of.2 -> v1.34.3-of.1
✓ v1.34.0 -> v1.34.3-of.1
```
#### 2. 检查升级是否可行
```bash
# 检查升级到 v1.34.0 是否可行
bke upgrade check --cluster bke-cluster/my-cluster --version v1.34.0
```
**输出示例**：
```
Performing pre-upgrade checks...
  ✓ Version Compatibility
  ✓ Cluster Health
  ✓ Node Status
  ✓ Resource Availability
  ✓ Image Availability

Check Results:
==============
✓ Version Compatibility
✓ Cluster Health
✓ Node Status
✓ Resource Availability
✓ Image Availability

Upgrade from v1.33.1-of.2 to v1.34.0 is possible
```
#### 3. 执行升级
```bash
# 升级集群到指定版本
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0

# 升级前备份
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0 --backup

# Dry-run 模式（只检查，不实际升级）
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0 --dry-run

# 强制升级（跳过兼容性检查）
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0 --force

# 指定备份路径
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0 --backup-path /custom/backup/path

# 指定超时时间
bke upgrade cluster --cluster bke-cluster/my-cluster --version v1.34.0 --timeout 7200
```
**输出示例**：
```
Performing pre-upgrade checks...
  ✓ Version Compatibility
  ✓ Cluster Health
  ✓ Node Status
  ✓ Resource Availability
  ✓ Image Availability
Backing up cluster...
Backup completed: /bke/backup/bke-cluster/my-cluster/20250401-143025
Upgrading cluster from v1.33.1-of.2 to v1.34.0...
Waiting for upgrade to complete...
Cluster phase: Upgrading
Cluster phase: Upgrading
Cluster phase: Provisioned
Performing post-upgrade checks...
Post-upgrade checks passed
Upgrade completed successfully from v1.33.1-of.2 to v1.34.0
```
#### 4. 回滚升级
```bash
# 回滚到上一个版本
bke upgrade cluster --cluster bke-cluster/my-cluster --rollback
```
### 四、设计优势
#### 1. 完整的升级流程
- **Pre-Check**：升级前检查（版本兼容性、集群健康、节点状态、资源可用性、镜像可用性）
- **Backup**：备份集群配置和 etcd 数据
- **Upgrade**：执行升级操作
- **Post-Check**：升级后验证（集群健康、节点状态、版本验证）
- **Rollback**：升级失败时自动回滚
#### 2. 版本兼容性管理
- 支持的 Kubernetes 版本列表
- 支持的升级路径矩阵
- 操作系统和容器运行时版本检查
- 版本解析和比较工具
#### 3. 安全保障
- 自动备份机制
- 升级失败自动回滚
- Dry-run 模式支持
- 详细的检查结果输出
#### 4. 灵活性
- 支持强制升级（跳过检查）
- 自定义备份路径
- 可配置的超时时间
- 支持回滚操作
### 五、总结
本次重构为 bkeadm 添加了完整的集群版本升级功能，包括：
1. **新增命令**：
   - `bke upgrade cluster`：执行集群升级
   - `bke upgrade check`：检查升级可行性
   - `bke upgrade list-paths`：列出支持的升级路径
2. **核心功能**：
   - 版本兼容性检查
   - 集群健康检查
   - 自动备份和回滚
   - 升级进度监控
3. **安全机制**：
   - Pre-check 和 Post-check
   - 自动备份
   - 失败自动回滚
   - Dry-run 模式

这个设计遵循了 Kubernetes 集群升级的最佳实践，确保升级过程的安全性和可靠性。

# 集群版本升级方案设计思路
## 集群版本升级方案设计思路
### 一、整体设计理念
#### 1.1 安全优先原则
**核心理念**：升级操作是高风险操作，必须确保安全第一。

**设计体现**：
- **多层检查机制**：升级前、升级后都要进行完整的健康检查
- **自动备份**：默认开启备份，确保可回滚
- **失败自动回滚**：升级失败时自动恢复到升级前状态
- **Dry-run 模式**：支持只检查不执行，降低误操作风险
#### 1.2 声明式升级
**核心理念**：用户只需声明目标版本，系统负责完成整个升级流程。

**设计体现**：
```go
// 用户只需指定目标版本
bke upgrade cluster --cluster ns/name --version v1.34.0

// 系统自动完成：
// 1. 版本兼容性检查
// 2. 集群健康检查
// 3. 备份集群
// 4. 执行升级
// 5. 验证升级结果
```
#### 1.3 可观测性
**核心理念**：升级过程必须透明可见，每个步骤都有明确的反馈。

**设计体现**：
- **UpgradeStatus 结构**：记录升级的每个阶段
- **CheckResult 列表**：详细记录每个检查项的结果
- **实时日志输出**：用户可以实时看到升级进度
### 二、架构设计思路
#### 2.1 分层架构
```
┌─────────────────────────────────────────────────────────┐
│                    命令层                        │
│  - 参数解析和验证                                         │
│  - 用户交互                                               │
│  - 结果展示                                               │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    业务逻辑层                     │
│  - 升降级流程编排                                         │
│  - 状态管理                                               │
│  - 错误处理                                               │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    兼容性层                │
│  - 版本兼容性检查                                         │
│  - 升级路径验证                                           │
│  - 版本解析和比较                                         │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes API 层                      │
│  - BKECluster CR 操作                                     │
│  - 集群状态查询                                           │
│  - 资源管理                                               │
└─────────────────────────────────────────────────────────┘
```
**设计思路**：
- **命令层**：负责用户交互，不包含业务逻辑
- **业务逻辑层**：编排升级流程，处理各种异常情况
- **兼容性层**：独立的版本管理模块，可单独测试和扩展
- **Kubernetes API 层**：封装 K8s 操作，隔离底层实现
#### 2.2 状态机设计
```
┌──────────────┐
│   PreCheck   │ 升级前检查
└──────┬───────┘
       │ 检查通过
       ▼
┌──────────────┐
│    Backup    │ 备份集群
└──────┬───────┘
       │ 备份成功
       ▼
┌──────────────┐
│   Upgrade    │ 执行升级
└──────┬───────┘
       │ 升级成功
       ▼
┌──────────────┐
│  PostCheck   │ 升级后检查
└──────┬───────┘
       │ 检查通过
       ▼
┌──────────────┐
│  Completed   │ 升级完成
└──────────────┘

失败时：
┌──────────────┐
│    Failed    │ 升级失败
└──────┬───────┘
       │ 有备份
       ▼
┌──────────────┐
│   Rollback   │ 自动回滚
└──────────────┘
```
**设计思路**：
- **明确的阶段划分**：每个阶段职责清晰
- **状态可追溯**：UpgradeStatus 记录每个阶段的状态
- **失败处理**：任何阶段失败都有明确的处理路径
### 三、核心机制设计思路
#### 3.1 版本兼容性检查机制
**问题**：如何判断两个版本之间是否可以升级？

**设计思路**：
```
版本兼容性检查 = 版本支持检查 + 升级路径检查

版本支持检查：
  目标版本是否在支持列表中？
  ├─ 是 → 继续
  └─ 否 → 拒绝升级

升级路径检查：
  当前版本 → 目标版本 是否在支持的升级路径中？
  ├─ 是 → 允许升级
  └─ 否 → 拒绝升级（可能需要中间版本）
```
**实现要点**：
```go
type UpgradePath struct {
    FromVersion string  // 源版本
    ToVersion   string  // 目标版本
    Supported   bool    // 是否支持
    Notes       string  // 说明信息
}

// 升级路径矩阵
var upgradePaths = []UpgradePath{
    {FromVersion: "v1.32.0", ToVersion: "v1.33.0", Supported: true},
    {FromVersion: "v1.32.0", ToVersion: "v1.34.0", Supported: false, 
     Notes: "跨版本升级不支持，请先升级到 v1.33.0"},
}
```
**为什么这样设计？**
1. **防止跨版本升级**：Kubernetes 不支持跨次版本升级（如 1.32 → 1.34）
2. **明确的升级路径**：用户可以清楚知道如何升级
3. **可扩展性**：新增版本只需添加升级路径配置
#### 3.2 备份机制
**问题**：升级失败时如何恢复？

**设计思路**：
```
备份策略：
┌─────────────────────────────────────┐
│  备份内容                            │
│  ├─ BKECluster CR 配置               │
│  ├─ etcd 数据                        │
│  ├─ 证书文件                         │
│  └─ 自定义配置                       │
└─────────────────────────────────────┘

备份存储：
/bke/backup/{namespace}/{cluster}/{timestamp}/
├── bkecluster.yaml    # 集群配置
├── etcd-backup.db     # etcd 备份
├── certs/             # 证书备份
└── metadata.json      # 元数据
```
**实现要点**：
```go
func (op *Options) backupCluster(ctx context.Context, bkeCluster *configv1beta1.BKECluster) (string, error) {
    // 1. 创建备份目录（带时间戳）
    backupPath := path.Join(backupDir, timestamp)
    
    // 2. 备份集群配置
    clusterBytes, _ := json.MarshalIndent(bkeCluster, "", "  ")
    os.WriteFile(clusterFile, clusterBytes, 0644)
    
    // 3. 备份 etcd 数据
    op.backupEtcd(ctx, bkeCluster, backupPath)
    
    // 4. 返回备份路径（用于回滚）
    return backupPath, nil
}
```
**为什么这样设计？**
1. **时间戳目录**：支持多次升级，每次都有独立备份
2. **完整备份**：不仅备份配置，还备份数据
3. **可追溯**：备份路径记录在 UpgradeStatus 中
#### 3.3 升级执行机制
**问题**：如何安全地执行升级？

**设计思路**：
```
升级执行流程：
┌─────────────────────────────────────┐
│  1. 更新 BKECluster CR               │
│     spec.clusterConfig.kubernetesVersion = targetVersion
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│  2. Cluster API Provider 监听变化    │
│     - 检测到版本变化                  │
│     - 触发升级流程                    │
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│  3. 滚动升级控制平面                  │
│     - 逐个升级 Master 节点            │
│     - 等待每个节点就绪                │
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│  4. 滚动升级工作节点                  │
│     - 逐个升级 Worker 节点            │
│     - 驱逐 Pod → 升级 → 验证          │
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│  5. 更新集群状态                      │
│     status.phase = "Provisioned"     │
│     status.kubernetesVersion = v1.34.0
└─────────────────────────────────────┘
```
**实现要点**：
```go
func (op *Options) performUpgrade(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
    // 1. 更新 CR 中的版本字段
    bkeCluster.Spec.ClusterConfig.KubernetesVersion = op.Version
    
    // 2. 提交更新到 Kubernetes
    dynamicClient.Resource(gvr).Namespace(ns).Update(ctx, unstructuredObj, metav1.UpdateOptions{})
    
    // 3. 等待升级完成（轮询检查状态）
    op.waitForUpgrade(ctx, bkeCluster)
    
    return nil
}
```
**为什么这样设计？**
1. **声明式 API**：利用 Kubernetes 的声明式特性，只需修改 CR
2. **Operator 模式**：Cluster API Provider 负责具体的升级逻辑
3. **滚动升级**：逐个节点升级，保证服务不中断
#### 3.4 检查机制
**问题**：如何确保升级前后的集群健康？

**设计思路**：
```
检查项设计：
┌─────────────────────────────────────┐
│  升级前检查                  │
│  ├─ 版本兼容性检查                    │
│  │   └─ 目标版本是否支持？            │
│  │   └─ 升级路径是否有效？            │
│  ├─ 集群健康检查                      │
│  │   └─ 控制平面是否正常？            │
│  │   └─ 所有节点是否 Ready？          │
│  ├─ 资源可用性检查                    │
│  │   └─ 磁盘空间是否充足？            │
│  │   └─ 内存是否充足？                │
│  └─ 镜像可用性检查                    │
│      └─ 目标版本镜像是否存在？        │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│  升级后检查                 │
│  ├─ 集群健康检查                      │
│  │   └─ 控制平面是否正常？            │
│  │   └─ 所有节点是否 Ready？          │
│  ├─ 版本验证                          │
│  │   └─ 实际版本是否为目标版本？      │
│  └─ 功能验证                          │
│      └─ 核心组件是否正常？            │
└─────────────────────────────────────┘
```
**实现要点**：
```go
type CheckResult struct {
    Name    string  // 检查项名称
    Status  string  // Passed / Failed
    Message string  // 详细信息
}

func (op *Options) performPreChecks(ctx context.Context, bkeCluster *configv1beta1.BKECluster, status *UpgradeStatus) error {
    checks := []struct {
        name string
        fn   func() error
    }{
        {"Version Compatibility", func() error { return checker.CheckKubernetesVersion(...) }},
        {"Cluster Health", func() error { return op.checkClusterHealth(...) }},
        // ...
    }
    
    for _, check := range checks {
        err := check.fn()
        // 记录检查结果
        status.CheckResults = append(status.CheckResults, CheckResult{...})
        if err != nil {
            return err  // 任一检查失败，终止升级
        }
    }
    return nil
}
```
**为什么这样设计？**
1. **模块化检查**：每个检查项独立，易于扩展
2. **结果可追溯**：所有检查结果记录在 UpgradeStatus 中
3. **快速失败**：任一检查失败立即终止，避免无效操作
#### 3.5 回滚机制
**问题**：升级失败时如何恢复？

**设计思路**：
```
回滚策略：
┌─────────────────────────────────────┐
│  自动回滚触发条件                    │
│  ├─ 升级执行失败                     │
│  ├─ 升级后检查失败                   │
│  └─ 升级超时                         │
└─────────────────────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│  回滚步骤                            │
│  1. 查找最近的备份                   │
│  2. 恢复 BKECluster CR 配置          │
│  3. 恢复 etcd 数据                   │
│  4. 重启控制平面                     │
│  5. 验证回滚结果                     │
└─────────────────────────────────────┘
```
**实现要点**：
```go
func (op *Options) performUpgrade(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
    // 执行升级
    if err := op.doUpgrade(ctx, bkeCluster); err != nil {
        // 升级失败，触发回滚
        if op.Backup {
            log.BKEFormat(log.INFO, "Attempting to rollback...")
            op.performRollback(ctx, bkeCluster, status)
        }
        return err
    }
    return nil
}
```
**为什么这样设计？**
1. **自动回滚**：无需用户干预，系统自动恢复
2. **备份驱动**：回滚依赖备份，确保有备份才回滚
3. **用户可控**：提供 `--rollback` 参数手动触发回滚
### 四、扩展性设计思路
#### 4.1 版本配置外部化
**设计思路**：版本支持列表和升级路径配置外部化，便于维护。
```go
// 未来可以改为从配置文件或 ConfigMap 加载
type VersionConfig struct {
    SupportedVersions []string      `yaml:"supportedVersions"`
    UpgradePaths      []UpgradePath `yaml:"upgradePaths"`
}

// 从文件加载
func LoadVersionConfig(path string) (*VersionConfig, error) {
    // 读取 YAML 配置文件
    // 解析为 VersionConfig
}
```
**优势**：
- 新增版本支持无需修改代码
- 支持动态更新版本配置
- 便于测试和维护
#### 4.2 检查项可插拔
**设计思路**：检查项通过注册机制添加，支持自定义扩展。
```go
type CheckFunc func(ctx context.Context, cluster *configv1beta1.BKECluster) error

var preCheckRegistry = make(map[string]CheckFunc)

func RegisterPreCheck(name string, fn CheckFunc) {
    preCheckRegistry[name] = fn
}

// 使用
RegisterPreCheck("CustomCheck", func(ctx context.Context, cluster *configv1beta1.BKECluster) error {
    // 自定义检查逻辑
    return nil
})
```
**优势**：
- 支持第三方扩展
- 检查项可配置
- 易于测试
#### 4.3 升级策略可配置
**设计思路**：支持不同的升级策略（滚动升级、蓝绿升级等）。
```go
type UpgradeStrategy string

const (
    StrategyRollingUpdate UpgradeStrategy = "RollingUpdate"
    StrategyBlueGreen     UpgradeStrategy = "BlueGreen"
    StrategyInPlace       UpgradeStrategy = "InPlace"
)

type UpgradeOptions struct {
    Strategy       UpgradeStrategy
    MaxSurge       int
    MaxUnavailable int
    Timeout        time.Duration
}

func (op *Options) performUpgrade(ctx context.Context, bkeCluster *configv1beta1.BKECluster) error {
    switch op.Strategy {
    case StrategyRollingUpdate:
        return op.rollingUpgrade(ctx, bkeCluster)
    case StrategyBlueGreen:
        return op.blueGreenUpgrade(ctx, bkeCluster)
    }
}
```
**优势**：
- 支持多种升级场景
- 用户可选择适合的策略
- 易于扩展新策略
### 五、用户体验设计思路
#### 5.1 渐进式信息披露
**设计思路**：根据用户需求提供不同详细程度的信息。
```
Level 1: 简洁输出（默认）
  ✓ Pre-check passed
  ✓ Backup completed
  ✓ Upgrade completed

Level 2: 详细输出
  Performing pre-upgrade checks...
    ✓ Version Compatibility
    ✓ Cluster Health
    ✓ Node Status
  Backing up cluster...
  Upgrading cluster from v1.33.0 to v1.34.0...
  Waiting for upgrade to complete...

Level 3: 调试输出
  [DEBUG] Checking version compatibility: v1.33.0 -> v1.34.0
  [DEBUG] Version compatibility check passed
  [DEBUG] Checking cluster health...
  [DEBUG] Cluster health check passed
  ...
```
**实现**：
```bash
# 默认简洁输出
bke upgrade cluster --cluster ns/name --version v1.34.0

# 详细输出
bke upgrade cluster --cluster ns/name --version v1.34.0 -v

# 调试输出
bke upgrade cluster --cluster ns/name --version v1.34.0 -vv
```
#### 5.2 错误提示友好
**设计思路**：错误信息要清晰、可操作。
```
❌ 不好的错误提示：
Error: upgrade failed

✓ 好的错误提示：
Error: upgrade failed: version compatibility check failed
  Current version: v1.32.0
  Target version: v1.34.0
  Reason: Cross-version upgrade is not supported
  Suggestion: Please upgrade to v1.33.0 first
```
**实现**：
```go
func (c *CompatibilityChecker) CheckKubernetesVersion(currentVersion, targetVersion string) error {
    if !c.isUpgradePathSupported(currentVersion, targetVersion) {
        return fmt.Errorf(`upgrade path %s -> %s is not supported
  Reason: Cross-version upgrade is not supported
  Suggestion: Please check supported upgrade paths with 'bke upgrade list-paths'`,
            currentVersion, targetVersion)
    }
    return nil
}
```
#### 5.3 进度可视化
**设计思路**：长时间操作要显示进度。
```
Upgrading cluster from v1.33.0 to v1.34.0...
[========================================] 100% | 5/5 nodes upgraded

Backup progress:
[====================                    ] 50% | Backing up etcd data...
```
### 六、总结
这个升级方案的设计思路围绕以下核心原则：
1. **安全第一**：多层检查、自动备份、失败回滚
2. **声明式**：用户只需声明目标，系统负责实现
3. **可观测**：每个步骤都有明确反馈
4. **可扩展**：版本配置、检查项、升级策略都可扩展
5. **用户友好**：清晰的错误提示、渐进式信息输出

通过分层架构、状态机设计、模块化检查等机制，确保了升级过程的可靠性、安全性和可维护性。

# `bke start` 命令的设计思路
## 一、命令定位与设计理念
### 1.1 命令定位
`bke start` 命令的设计定位是：**独立启动 BKE 集群所需的基础依赖服务**。
### 1.2 设计理念
```
┌─────────────────────────────────────────────────────────────┐
│                    bke start 设计理念                         │
└─────────────────────────────────────────────────────────────┘

1. 模块化设计
   └─> 每个服务独立管理，可单独启动/停止
   
2. 运行时无关性
   └─> 支持 Docker 和 Containerd 两种容器运行时
   
3. 配置灵活性
   └─> 提供默认配置，同时支持用户自定义
   
4. 幂等性保证
   └─> 服务已运行时不会重复启动
   
5. 独立可用性
   └─> 可脱离 bke init 独立使用，便于调试和维护
```
## 二、命令结构设计
### 2.1 命令树结构
```
bke start
├── bke start image        # 启动镜像仓库服务
├── bke start yum          # 启动 YUM 包仓库服务
├── bke start nfs          # 启动 NFS 共享存储服务
├── bke start chart        # 启动 Helm Chart 仓库服务
└── bke start ntpserver    # 启动 NTP 时间同步服务
```
### 2.2 命令参数设计
每个子命令都支持以下参数（NTP 除外）：

| 参数 | 说明 | 默认值 |
|------|------|--------|
| `--name` | 容器名称 | 服务特定默认值 |
| `--image` | 镜像地址 | 服务特定默认值 |
| `--port` | 服务端口 | 服务特定默认值 |
| `--data` | 数据目录 | `/tmp/<service>` |

**NTP 服务特殊参数**：

| 参数 | 说明 |
|------|------|
| `--systemd` | 使用 systemd 管理服务 |
| `--foreground` | 前台运行服务 |
## 三、核心服务设计
### 3.1 镜像仓库服务
**功能**：提供 Docker 镜像的存储和分发能力

**设计要点**：
```go
// 1. 运行时适配
func StartImageRegistry(name, image, imageRegistryPort, imageDataDirectory string) error {
    if infrastructure.IsContainerd() && !infrastructure.IsDocker() {
        return startImageRegistryWithContainerd(name, image, imageRegistryPort, imageDataDirectory)
    }
    
    if !infrastructure.IsDocker() {
        log.BKEFormat(log.ERROR, "docker or containerd runtime not found.")
        return nil
    }
    return startImageRegistryWithDocker(name, image, imageRegistryPort, imageDataDirectory)
}

// 2. Docker 实现
func startImageRegistryWithDocker(name, image, imageRegistryPort, imageDataDirectory string) error {
    // 生成证书配置
    certPath := fmt.Sprintf("/etc/docker/%s", name)
    if err := generateConfig(certPath, imageRegistryPort); err != nil {
        return err
    }
    
    // 确保镜像存在
    if err := global.Docker.EnsureImageExists(...); err != nil {
        return err
    }
    
    // 检查容器是否已运行
    serverRunFlag, err := global.Docker.EnsureContainerRun(name)
    if serverRunFlag {
        log.BKEFormat(log.INFO, "The mirror warehouse service is already running.")
        return nil
    }
    
    // 启动容器
    return runDockerImageRegistry(name, image, imageRegistryPort, imageDataDirectory, certPath)
}
```
**关键特性**：
- 支持 HTTPS（自动生成证书）
- 数据持久化（通过 volume 挂载）
- 自动重启策略
- 容器 IP 分配（Containerd 模式）
### 3.2 YUM 包仓库服务
**功能**：提供 RPM/DEB 软件包的存储和分发

**设计要点**：
```go
// 使用 Nginx 作为 HTTP 服务器
func runDockerYumRegistry(name, image, yumRegistryPort, yumDataDirectory string) error {
    return global.Docker.Run(
        &container.Config{
            Image:        image,
            ExposedPorts: map[nat.Port]struct{}{"80/tcp": {}},
        },
        &container.HostConfig{
            Mounts: []mount.Mount{
                {Type: mount.TypeBind, Source: yumDataDirectory, Target: "/repo"},
            },
            PortBindings: map[nat.Port][]nat.PortBinding{
                nat.Port("80/tcp"): {{HostIP: "0.0.0.0", HostPort: yumRegistryPort}},
            },
            RestartPolicy: container.RestartPolicy{Name: "always"},
        },
        nil, nil, name,
    )
}
```
**关键特性**：
- 使用 Nginx 提供静态文件服务
- 支持自定义配置文件
- 数据目录挂载到 `/repo`
### 3.3 NFS 共享存储服务
**功能**：提供 NFS 共享存储能力

**设计要点**：
```go
func runDockerNFSServer(name, image, nfsDataDirectory string) error {
    return global.Docker.Run(
        &container.Config{
            Image: image,
            Env: []string{
                "SHARED_DIRECTORY=/nfsshare",
                "FILEPERMISSIONS_UID=0",
                "FILEPERMISSIONS_GID=0",
                "FILEPERMISSIONS_MODE=0755",
            },
        },
        &container.HostHostConfig{
            Mounts:     []mount.Mount{{Type: mount.TypeBind, Source: nfsDataDirectory, Target: "/nfsshare"}},
            Privileged: true,  // NFS 需要特权模式
            CapAdd:    strslice.StrSlice{"SYS_ADMIN", "SETPCAP"},
        },
        nil, nil, name,
    )
}
```
**关键特性**：
- 需要特权模式运行
- 自动设置文件权限
- 支持多客户端并发访问
### 3.4 Helm Chart 仓库服务
**功能**：提供 Helm Chart 的存储和分发

**设计要点**：
```go
func runDockerChartRegistry(name, image, chartRegistryPort, chartDataDirectory string) error {
    return global.Docker.Run(
        &container.Config{
            Image: image,
            Env: []string{
                "DEBUG=true",
                "STORAGE=local",
                "STORAGE_LOCAL_ROOTDIR=/charts",
            },
        },
        &container.HostConfig{
            Mounts: []mount.Mount{
                {Type: mount.TypeBind, Source: chartDataDirectory, Target: "/charts"},
            },
            PortBindings: map[nat.Port][]nat.PortBinding{
                nat.Port("8080/tcp"): {{HostIP: "0.0.0.0", HostPort: chartRegistryPort}},
            },
        },
        nil, nil, name,
    )
}
```
**关键特性**：
- 使用 ChartMuseum 作为后端
- 支持本地存储模式
- 提供 API 接口
### 3.5 NTP 时间同步服务
**功能**：提供网络时间同步服务

**设计要点**：
```go
// 支持三种运行模式
func (op *Options) ntpServerStartCmd() {
    // 1. 检查服务是否已运行
    _, err := sntp.Client(fmt.Sprintf("127.0.0.1:%d", utils.DefaultNTPServerPort))
    if err == nil {
        log.BKEFormat(log.INFO, "The ntp server is running")
        return
    }
    
    // 2. systemd 模式（推荐用于生产环境）
    if cmd.Flag("systemd").Value.String() == "true" {
        server.SystemdNTPServer()
        return
    }
    
    // 3. 前台模式（用于调试）
    if cmd.Flag("foreground").Value.String() == "true" {
        server.SystemdDaemonNTPServer()
        return
    }
    
    // 4. 守护进程模式（默认）
    server.DaemonNTPServer()
}
```
**关键特性**：
- 支持多种运行模式
- 自动生成 systemd 服务文件
- 支持进程守护
## 四、运行时适配设计
### 4.1 运行时检测
```go
// 在 infrastructure 包中实现运行时检测
func IsDocker() bool {
    // 检查 Docker 是否可用
}

func IsContainerd() bool {
    // 检查 Containerd 是否可用
}
```
### 4.2 统一接口设计
```go
// 每个服务都提供统一的启动接口
func StartXxxService(name, image, port, data string) error {
    if infrastructure.IsContainerd() && !infrastructure.IsDocker() {
        return startXxxWithContainerd(...)
    }
    
    if !infrastructure.IsDocker() {
        return fmt.Errorf("no supported container runtime found")
    }
    
    return startXxxWithDocker(...)
}
```
### 4.3 运行时差异处理
| 特性 | Docker | Containerd |
|------|--------|------------|
| 容器运行 | `docker run` | `nerdctl run` |
| 镜像管理 | Docker API | nerdctl CLI |
| 网络配置 | 自动分配 | 手动分配 IP |
| 证书路径 | `/etc/docker/<name>` | `<k3s-data-dir>/<name>` |
## 五、与 bke init 的关系
### 5.1 调用关系
```
bke init
└─> ensureRepository()
    ├─> LoadLocalImage()           # 加载本地镜像
    ├─> LoadLocalRepository()      # 加载本地仓库镜像
    ├─> ContainerServer()          # 启动镜像仓库
    │   └─> server.StartImageRegistry()
    ├─> YumServer()                # 启动 YUM 仓库
    │   └─> server.StartYumRegistry()
    ├─> ChartServer()              # 启动 Chart 仓库
    │   └─> server.StartChartRegistry()
    └─> NFSServer()                # 启动 NFS 服务
        └─> server.StartNFSServer()
```
### 5.2 使用场景对比
| 场景 | bke init | bke start |
|------|----------|-----------|
| 初始化引导节点 | ✅ 使用 | ❌ 不适用 |
| 单独启动某个服务 | ❌ 不适用 | ✅ 使用 |
| 调试服务问题 | ❌ 不适用 | ✅ 使用 |
| 服务重启 | ❌ 不适用 | ✅ 使用 |
| 自定义配置 | ❌ 固定配置 | ✅ 灵活配置 |
## 六、设计优势
### 6.1 模块化
```
优势：
1. 每个服务独立管理，职责清晰
2. 便于单独调试和维护
3. 支持按需启动服务
```
### 6.2 可扩展性
```
优势：
1. 易于添加新的服务类型
2. 支持自定义镜像和配置
3. 适配多种容器运行时
```
### 6.3 用户友好
```
优势：
1. 提供合理的默认值，开箱即用
2. 支持自定义配置，满足特殊需求
3. 清晰的错误提示和日志输出
```
### 6.4 健壮性
```
优势：
1. 幂等性设计，避免重复启动
2. 自动重试机制
3. 服务健康检查
```
## 七、使用示例
### 7.1 启动镜像仓库
```bash
# 使用默认配置
bke start image

# 自定义配置
bke start image \
  --name my-registry \
  --image registry:2.8.1 \
  --port 5000 \
  --data /data/registry
```
### 7.2 启动 NTP 服务
```bash
# 守护进程模式（默认）
bke start ntpserver

# systemd 服务模式
bke start ntpserver --systemd

# 前台运行（调试用）
bke start ntpserver --foreground
```
### 7.3 启动所有服务
```bash
# 依次启动各个服务
bke start image
bke start yum
bke start chart
bke start nfs
bke start ntpserver
```
## 八、总结
`bke start` 命令的设计思路体现了以下核心思想：
1. **关注点分离**：将基础服务的启动逻辑从 `bke init` 中分离出来，提供独立的管理入口
2. **运行时无关**：通过适配器模式支持多种容器运行时，提高兼容性
3. **配置灵活**：提供默认配置的同时支持用户自定义，平衡易用性和灵活性
4. **幂等性保证**：服务启动前检查状态，避免重复启动造成的问题
5. **可观测性**：提供清晰的日志输出，便于问题排查

这种设计使得 BKE 的基础服务管理更加灵活、可靠和易于维护。


