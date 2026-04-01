
# bkeadm 代码文档
## 项目概述
**bkeadm** 是 BKE (Bocloud Enterprise Kubernetes) 的部署管理工具，提供 Kubernetes 集群的部署、运维、治理等全生命周期管理能力。支持在线和离线两种安装模式。
### 项目信息
- **模块路径**: `gopkg.openfuyao.cn/bkeadm`
- **许可证**: Mulan PSL v2
- **版权**: Bocloud Technologies Co., Ltd.
## 架构设计
### 目录结构
```
bkeadm/
├── main.go                 # 程序入口
├── cmd/                    # 命令行入口层
│   ├── root.go            # 根命令定义
│   ├── init.go            # 初始化命令
│   ├── build.go           # 构建命令
│   ├── cluster.go         # 集群管理命令
│   ├── config.go          # 配置生成命令
│   ├── reset.go           # 重置命令
│   ├── registry.go        # 仓库管理命令
│   └── ...
├── pkg/                    # 核心业务逻辑层
│   ├── initialize/        # 初始化模块
│   ├── build/             # 构建模块
│   ├── cluster/           # 集群管理模块
│   ├── infrastructure/    # 基础设施模块
│   ├── executor/          # 执行器抽象层
│   ├── server/            # 服务模块
│   ├── config/            # 配置管理模块
│   ├── global/            # 全局状态管理
│   ├── reset/             # 重置模块
│   └── registry/          # 仓库操作模块
├── utils/                  # 工具函数层
│   ├── constants.go       # 常量定义
│   ├── download.go        # 下载工具
│   ├── tar.go             # 压缩解压工具
│   ├── log/               # 日志模块
│   └── version/           # 版本信息
├── assets/                 # 资源文件
├── build/                  # 构建脚本
└── scripts/               # 辅助脚本
```
## 核心模块详解
### 1. 入口层
#### [main.go](file:///d:/code/github/bkeadm/main.go)
程序主入口，负责初始化版本信息并执行根命令。
```go
func main() {
    if version.Version == "" {
        version.GitCommitID = gitCommitId
        version.Version = ver
        version.Architecture = architecture
        version.Timestamp = timestamp
    }
    cmd.Execute()
}
```
#### [root.go](file:///d:/code/github/bkeadm/cmd/root.go)
定义根命令和全局参数，注册所有子命令。

**核心结构**:
- `rootCmd`: Cobra 根命令
- `Options`: 全局选项（包含 kubeconfig 路径等）

**注册的子命令**:
- `init` - 初始化引导节点
- `reset` - 重置环境
- `start` - 启动服务
- `status` - 查看状态
- `version` - 版本信息
- `config` - 配置生成
- `registry` - 仓库管理
- `build` - 构建离线包
- `cluster` - 集群管理
- `remove` - 移除组件
### 2. 初始化模块 (pkg/initialize)
#### [initialize.go](file:///d:/code/github/bkeadm/pkg/initialize/initialize.go)
初始化引导节点的核心逻辑。

**Options 结构体**:

| 字段 | 类型 | 说明 |
|------|------|------|
| HostIP | string | 主机 IP 地址 |
| Domain | string | 镜像仓库域名 |
| KubernetesPort | string | Kubernetes API 端口 |
| ImageRepoPort | string | 镜像仓库端口 |
| Runtime | string | 容器运行时类型 |
| InstallConsole | bool | 是否安装控制台 |
| EnableNTP | bool | 是否启用 NTP 服务 |

**初始化流程** (`Initialize()` 方法):
```
1. nodeInfo()          - 收集节点信息
2. Validate()          - 参数校验
3. setTimezone()       - 设置时区
4. prepareEnvironment() - 准备环境
5. ensureContainerServer() - 启动容器服务
6. ensureRepository()  - 启动仓库服务
7. ensureClusterAPI()  - 启动 Cluster API
8. ensureConsoleAll()  - 安装控制台（可选）
9. generateClusterConfig() - 生成集群配置
```
#### 子模块
| 模块 | 文件 | 功能 |
|------|------|------|
| bkeagent | [bkeagent.go](file:///d:/code/github/bkeadm/pkg/initialize/bkeagent/bkeagent.go) | 部署 BKE Agent |
| clusterapi | [clusterapi.go](file:///d:/code/github/bkeadm/pkg/initialize/clusterapi/clusterapi.go) | 部署 Cluster API |
| bkeconsole | [install_console.go](file:///d:/code/github/bkeadm/pkg/initialize/bkeconsole/install_console.go) | 部署控制台 |
| repository | [repository.go](file:///d:/code/github/bkeadm/pkg/initialize/repository/repository.go) | 仓库初始化 |
| syscompat | [compat.go](file:///d:/code/github/bkeadm/pkg/initialize/syscompat/compat.go) | 系统兼容性检查 |
| timezone | [timezone.go](file:///d:/code/github/bkeadm/pkg/initialize/timezone/timezone.go) | 时区设置 |
### 3. 构建模块
#### [build.go](file:///d:/code/github/bkeadm/pkg/build/build.go)
构建离线安装包的核心逻辑。

**构建流程** (`Build()` 方法):
```
step.1: loadAndVerifyBuildConfig() - 加载并验证配置
step.2: prepareBuildWorkspace()    - 准备工作空间
step.3: collectDependenciesAndImages() - 收集依赖和镜像
step.4: createFinalPackage()       - 创建最终安装包
```
**支持的构建策略**:
- 离线镜像同步
- 在线镜像拉取
- RPM 包构建
- Helm Chart 打包
#### 关键文件
| 文件 | 功能 |
|------|------|
| [ociloader.go](file:///d:/code/github/bkeadm/pkg/build/ociloader.go) | OCI 镜像加载 |
| [ocisyncimage.go](file:///d:/code/github/bkeadm/pkg/build/ocisyncimage.go) | OCI 镜像同步 |
| [registrysyncimage.go](file:///d:/code/github/bkeadm/pkg/build/registrysyncimage.go) | 仓库镜像同步 |
| [buildrpm.go](file:///d:/code/github/bkeadm/pkg/build/buildrpm.go) | RPM 包构建 |
| [patch.go](file:///d:/code/github/bkeadm/pkg/build/patch.go) | 补丁管理 |
### 4. 集群管理模块
#### [cluster.go](file:///d:/code/github/bkeadm/pkg/cluster/cluster.go)
Kubernetes 集群生命周期管理。

**核心功能**:
- `List()` - 列出所有 BKECluster
- `Create()` - 创建集群
- `Delete()` - 删除集群
- `Upgrade()` - 升级集群
- `Scale()` - 扩缩容节点

**BKECluster 资源**:
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKECluster
metadata:
  namespace: bke-xxx
  name: xxx
spec:
  controlPlaneEndpoint:
    host: 192.168.1.100
    port: 6443
  ...
```
#### [deploy.go](file:///d:/code/github/bkeadm/pkg/cluster/deploy.go)
集群部署逻辑实现。
### 5. 基础设施模块
#### [infrastructure.go](file:///d:/code/github/bkeadm/pkg/infrastructure/infrastructure.go)
容器运行时管理抽象层。

**RuntimeConfig 结构体**:

| 字段 | 说明 |
|------|------|
| Runtime | 运行时类型：docker 或 containerd |
| RuntimeStorage | 运行时存储路径 |
| Domain | 镜像仓库域名 |
| ImageRepoPort | 镜像仓库端口 |

**核心函数**:
- `IsDocker()` - 检查 Docker 是否安装
- `IsContainerd()` - 检查 Containerd 是否安装
- `RuntimeInstall()` - 安装容器运行时
#### 子模块
| 模块 | 文件 | 功能 |
|------|------|------|
| containerd | [containerd.go](file:///d:/code/github/bkeadm/pkg/infrastructure/containerd/containerd.go) | Containerd 安装配置 |
| dockerd | [docker.go](file:///d:/code/github/bkeadm/pkg/infrastructure/dockerd/docker.go) | Docker 安装配置 |
| k3s | [k3s.go](file:///d:/code/github/bkeadm/pkg/infrastructure/k3s/k3s.go) | K3s 轻量级 Kubernetes |
| kubelet | [kubelet.go](file:///d:/code/github/bkeadm/pkg/infrastructure/kubelet/kubelet.go) | Kubelet 配置管理 |
### 6. 执行器模块
#### [k8s/k8s.go](file:///d:/code/github/bkeadm/pkg/executor/k8s/k8s.go)
Kubernetes 客户端封装。

**KubernetesClient 接口**:
```go
type KubernetesClient interface {
    GetClient() kubernetes.Interface
    GetDynamicClient() dynamic.Interface
    InstallYaml(filename string, variable map[string]string, ns string) error
    PatchYaml(filename string, variable map[string]string) error
    UninstallYaml(filename string, ns string) error
    WatchEventByAnnotation(namespace string)
    CreateNamespace(namespace *corev1.Namespace) error
    CreateSecret(secret *corev1.Secret) error
    GetNamespace(filename string) (string, error)
}
```
#### [containerd/containerd.go](file:///d:/code/github/bkeadm/pkg/executor/containerd/containerd.go)
Containerd 客户端封装，支持 nerdctl 操作。
#### [docker/docker.go](file:///d:/code/github/bkeadm/pkg/executor/docker/docker.go)
Docker 客户端封装，提供镜像和容器操作。
#### [exec/exec.go](file:///d:/code/github/bkeadm/pkg/executor/exec/exec.go)
命令执行器抽象，支持本地命令执行。
### 7. 服务模块
#### [image_registry.go](file:///d:/code/github/bkeadm/pkg/server/image_registry.go)
镜像仓库服务管理。

**核心函数**:
- `StartImageRegistry()` - 启动镜像仓库
- `startImageRegistryWithDocker()` - 使用 Docker 启动
- `startImageRegistryWithContainerd()` - 使用 Containerd 启动

**默认镜像**: `registry:2.8.1`
#### [yum_registry.go](file:///d:/code/github/bkeadm/pkg/server/yum_registry.go)
YUM 软件源服务管理。

**默认镜像**: `nginx:1.23.0-alpine`
#### [chart_registry.go](file:///d:/code/github/bkeadm/pkg/server/chart_registry.go)
Helm Chart 仓库服务管理。

**默认镜像**: `helm/chartmuseum:v0.16.2`
#### [ntp_server.go](file:///d:/code/github/bkeadm/pkg/server/ntp_server.go)
NTP 时间同步服务管理。
### 8. 全局状态管理
#### [global.go](file:///d:/code/github/bkeadm/pkg/global/global.go)
全局变量和工具函数。

**全局变量**:
```go
var (
    Docker      docker.DockerClient      // Docker 客户端
    Containerd  containerd.ContainerdClient // Containerd 客户端
    K8s         k8s.KubernetesClient     // Kubernetes 客户端
    Command     exec.Executor            // 命令执行器
    Workspace   string                   // 工作空间路径
    CustomExtra map[string]string        // 自定义扩展配置
)
```
**默认工作空间**: `/bke`

**工具函数**:
- `TarGZ()` - 创建 tar.gz 压缩包
- `TarGZWithDir()` - 带目录的压缩
- `TaeGZWithoutChangeFile()` - 忽略文件变更的压缩
### 9. 配置管理模块
#### [config.go](file:///d:/code/github/bkeadm/pkg/config/config.go)
集群配置文件生成。

**核心方法**:
- `Config()` - 生成配置文件
- `GenerateControllerParam()` - 生成 Containerd 参数

**生成的配置文件**:
- BKECluster YAML
- Containerd 配置
- Kubelet 配置
### 10. 重置模块
#### [reset.go](file:///d:/code/github/bkeadm/pkg/reset/reset.go)
环境重置和清理。

**重置流程** (`Reset()` 方法):
```
1. RemoveLocalKubernetes() - 移除本地 Kubernetes
2. RemoveContainerService() - 移除容器服务（可选）
3. RemoveNtpService() - 移除 NTP 服务（可选）
4. removeAllInOne() - 移除所有组件（可选）
5. source.ResetSource() - 重置软件源
```
## 常量定义
### [constants.go](file:///d:/code/github/bkeadm/utils/constants.go)

#### 服务名称
```go
const (
    LocalKubernetesName     = "kubernetes"
    LocalImageRegistryName  = "bocloud_image_registry"
    LocalYumRegistryName    = "bocloud_yum_registry"
    LocalChartRegistryName  = "bocloud_chart_registry"
    LocalNFSRegistryName    = "bocloud_nfs_registry"
)
```
#### 默认端口
```go
const (
    DefaultKubernetesPort   = "36443"
    DefaultChartRegistryPort = "38080"
    DefaultNTPServerPort    = 123
    DefaultAgentHealthPort  = "58080"
)
```
#### 默认镜像
```go
const (
    DefaultLocalK3sRegistry = "rancher/k3s:v1.25.16-k3s4"
    DefaultLocalImageRegistry = "registry:2.8.1"
    DefaultLocalYumRegistry = "nginx:1.23.0-alpine"
    DefaultLocalChartRegistry = "helm/chartmuseum:v0.16.2"
)
```
#### 文件路径
```go
const (
    ImageFile           = "volumes/registry.image"
    ImageDataDirectory  = "mount/image_registry"
    SourceDataDirectory = "mount/source_registry"
    ChartDataDirectory  = "mount/charts"
)
```
## 命令行使用示例
### 初始化引导节点
```bash
# 基本初始化
bke init

# 安装控制台
bke init --installConsole=true

# 使用配置文件
bke init --file bkecluster.yaml

# 使用外部镜像仓库
bke init --otherRepo cr.openfuyao.cn/openfuyao --otherSource http://192.168.1.120:40080
```
### 构建离线包
```bash
bke build --file build.yaml --target output.tar.gz
```
### 集群管理
```bash
# 列出集群
bke cluster list

# 创建集群
bke cluster create --file cluster.yaml

# 删除集群
bke cluster delete --namespace bke-test --name my-cluster
```
### 重置环境
```bash
# 仅重置 Kubernetes
bke reset

# 完全重置
bke reset --all

# 重置并清理数据
bke reset --mount
```
## 设计模式
### 1. 执行器抽象模式
通过接口抽象不同容器运行时：
```
┌─────────────────┐
│   Executor      │
│   Interface     │
└────────┬────────┘
         │
    ┌────┴────┬─────────────┐
    │         │             │
┌───▼───┐ ┌───▼───┐   ┌─────▼─────┐
│Docker │ │Containerd│ │   K8s     │
│Client │ │  Client  │ │  Client   │
└───────┘ └──────────┘ └───────────┘
```
### 2. 命令模式
使用 Cobra 框架实现命令行模式，每个子命令独立封装。
### 3. 配置驱动模式
通过 YAML 配置文件驱动部署行为，支持灵活的部署选项。
## 依赖关系
```
bkeadm
├── gopkg.openfuyao.cn/cluster-api-provider-bke (BKE Cluster API)
├── github.com/spf13/cobra (CLI 框架)
├── github.com/docker/docker (Docker SDK)
├── k8s.io/client-go (Kubernetes 客户端)
├── github.com/shirou/gopsutil (系统信息)
└── gopkg.in/yaml.v3 (YAML 解析)
```
## 工作空间结构
```
/bke/                          # 默认工作空间
├── tmpl/                      # 模板文件
├── volumes/                   # 数据卷
│   ├── registry.image        # 镜像数据
│   ├── image.tar.gz          # 镜像压缩包
│   └── source.tar.gz         # 源码包
├── mount/                     # 挂载目录
│   ├── image_registry/       # 镜像仓库数据
│   ├── source_registry/      # 软件源数据
│   └── charts/               # Helm Charts
└── cluster/                   # 集群配置
    └── bke-xxx-xxx.yaml      # BKECluster 配置文件
```
此文档涵盖了 bkeadm 项目的主要架构、核心模块和使用方法。如需更详细的某个模块说明，请告知。
        
