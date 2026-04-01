      
       
# bkeadm安装部署时的外部仓库/源依赖
## 一、外部仓库/源清单
### 1. 镜像仓库
| 仓库地址 | 用途 | 使用场景 |
|---------|------|---------|
| `hub.oepkgs.net/openfuyao` | 第三方镜像仓库（默认在线源） | 在线安装时默认使用 |
| `cr.openfuyao.cn/openfuyao` | openFuyao官方镜像仓库 | 官方镜像拉取 |
| `deploy.bocloud.k8s:40443` | 默认内部镜像仓库域名 | 集群内部镜像仓库 |
### 2. Chart仓库
| 仓库地址 | 用途 | 使用场景 |
|---------|------|---------|
| `cr.openfuyao.cn` | 默认Chart仓库 | Helm Chart存储 |
### 3. YUM/HTTP源
| 源地址 | 用途 | 使用场景 |
|-------|------|---------|
| `http.bocloud.k8s:40080` | 默认YUM源域名 | 系统包、运行时文件下载 |
### 4. 版本配置源
| 地址 | 用途 | 使用场景 |
|-----|------|---------|
| `https://openfuyao.obs.cn-north-4.myhuaweicloud.com/openFuyao/version-config/` | 版本配置下载 | 在线获取版本信息 |
### 5. NTP服务器
| 地址 | 用途 | 使用场景 |
|-----|------|---------|
| `cn.pool.ntp.org:123` | 默认NTP服务器 | 时间同步 |
### 6. 基础服务镜像（从外部拉取）
| 镜像 | 用途 | 来源 |
|-----|------|-----|
| `registry:2.8.1` | 本地镜像仓库服务 | Docker Hub / 第三方镜像 |
| `nginx:1.23.0-alpine` | YUM仓库服务 | Docker Hub / 第三方镜像 |
| `helm/chartmuseum:v0.16.2` | Chart仓库服务 | GitHub / 第三方镜像 |
| `openebs/nfs-server-alpine:0.9.0` | NFS服务 | Docker Hub / 第三方镜像 |
| `rancher/k3s:v1.25.16-k3s4` | K3s集群 | Rancher / 第三方镜像 |
| `rancher/mirrored-pause:3.6` | K3s pause镜像 | Rancher / 第三方镜像 |
## 二、架构图
```mermaid
graph TB
    subgraph "bkeadm 安装部署外部依赖架构"
        subgraph "引导节点 Bootstrap Node"
            BKEADM[bkeadm init]
            
            subgraph "本地服务 Local Services"
                IMG_REG[镜像仓库<br/>registry:2.8.1<br/>端口: 40443]
                YUM_REG[YUM仓库<br/>nginx:1.23.0-alpine<br/>端口: 40080]
                CHART_REG[Chart仓库<br/>chartmuseum:v0.16.2<br/>端口: 38080]
                NFS_SVC[NFS服务<br/>nfs-server-alpine:0.9.0<br/>端口: 2049]
                K3S[K3s集群<br/>rancher/k3s:v1.25.16<br/>端口: 36443]
            end
        end
        
        subgraph "外部镜像仓库 External Image Registries"
            HUB_OEPKGS[hub.oepkgs.net/openfuyao<br/>第三方镜像仓库<br/>默认在线源]
            CR_OPENFUYAO[cr.openfuyao.cn/openfuyao<br/>openFuyao官方仓库]
            DEPLOY_BOCLOUD[deploy.bocloud.k8s:40443<br/>内部镜像仓库域名]
        end
        
        subgraph "外部Chart仓库 External Chart Repository"
            CHART_OPENFUYAO[cr.openfuyao.cn<br/>Chart仓库<br/>端口: 443]
        end
        
        subgraph "外部YUM/HTTP源 External YUM/HTTP Source"
            HTTP_BOCLOUD[http.bocloud.k8s:40080<br/>YUM源域名<br/>系统包/运行时文件]
        end
        
        subgraph "外部配置源 External Config Source"
            VERSION_CONFIG[华为云OBS<br/>openfuyao.obs.cn-north-4.myhuaweicloud.com<br/>版本配置下载]
        end
        
        subgraph "外部NTP服务 External NTP Service"
            NTP_POOL[cn.pool.ntp.org:123<br/>NTP时间同步服务器]
        end
        
        BKEADM -->|1. 拉取基础服务镜像| HUB_OEPKGS
        BKEADM -->|1. 拉取基础服务镜像| CR_OPENFUYAO
        BKEADM -->|2. 下载运行时文件| HTTP_BOCLOUD
        BKEADM -->|3. 下载版本配置| VERSION_CONFIG
        BKEADM -->|4. 时间同步| NTP_POOL
        BKEADM -->|5. 拉取Chart| CHART_OPENFUYAO
        
        HUB_OEPKGS -.->|镜像同步| IMG_REG
        CR_OPENFUYAO -.->|镜像同步| IMG_REG
        HTTP_BOCLOUD -.->|文件同步| YUM_REG
        CHART_OPENFUYAO -.->|Chart同步| CHART_REG
        
        IMG_REG -->|提供镜像| K3S
        YUM_REG -->|提供系统包| K3S
        CHART_REG -->|提供Helm Chart| K3S
        NFS_SVC -->|提供存储| K3S
    end
    
    style BKEADM fill:#e1f5ff,stroke:#0066cc,stroke-width:3px
    style HUB_OEPKGS fill:#fff4e6,stroke:#ff9800,stroke-width:2px
    style CR_OPENFUYAO fill:#fff4e6,stroke:#ff9800,stroke-width:2px
    style DEPLOY_BOCLOUD fill:#fff4e6,stroke:#ff9800,stroke-width:2px
    style CHART_OPENFUYAO fill:#e8f5e9,stroke:#4caf50,stroke-width:2px
    style HTTP_BOCLOUD fill:#f3e5f5,stroke:#9c27b0,stroke-width:2px
    style VERSION_CONFIG fill:#fce4ec,stroke:#e91e63,stroke-width:2px
    style NTP_POOL fill:#e0f2f1,stroke:#009688,stroke-width:2px
```
## 三、依赖关系详解
### 1. 离线安装模式
```mermaid
graph LR
    subgraph "离线安装 Offline Installation"
        A[本地镜像包<br/>image.tar.gz] --> B[镜像仓库服务]
        C[本地源文件<br/>source.tar.gz] --> D[YUM仓库服务]
        E[本地Chart包<br/>charts.tar.gz] --> F[Chart仓库服务]
        G[本地NFS数据<br/>nfsshare.tar.gz] --> H[NFS服务]
    end
    
    style A fill:#e3f2fd
    style C fill:#e3f2fd
    style E fill:#e3f2fd
    style G fill:#e3f2fd
```
**特点**：
- 所有资源来自本地文件
- 无需外部网络访问
- 适用于隔离环境
### 2. 在线安装模式
```mermaid
graph LR
    subgraph "在线安装 Online Installation"
        A[hub.oepkgs.net/openfuyao] -->|拉取镜像| B[镜像仓库服务]
        C[HTTP源] -->|下载文件| D[YUM仓库服务]
        E[cr.openfuyao.cn] -->|拉取Chart| F[Chart仓库服务]
        G[版本配置URL] -->|下载配置| H[版本管理]
    end
    
    style A fill:#fff3e0
    style C fill:#fff3e0
    style E fill:#fff3e0
    style G fill:#fff3e0
```
**特点**：
- 从外部仓库拉取镜像
- 从HTTP服务器下载文件
- 需要外网访问
### 3. 私有仓库模式
```mermaid
graph LR
    subgraph "私有仓库安装 Private Repository Installation"
        A[私有镜像仓库<br/>--otherRepo] -->|拉取镜像| B[镜像仓库服务]
        C[私有HTTP源<br/>--otherSource] -->|下载文件| D[YUM仓库服务]
        E[私有Chart仓库<br/>--otherChart] -->|拉取Chart| F[Chart仓库服务]
    end
    
    style A fill:#e8f5e9
    style C fill:#e8f5e9
    style E fill:#e8f5e9
```
**特点**：
- 使用企业内部私有仓库
- 支持TLS证书认证
- 适用于企业内网环境
## 四、端口映射表
| 服务 | 外部端口 | 内部端口 | 用途 |
|-----|---------|---------|------|
| K3s API Server | 36443 | 6443 | Kubernetes API |
| 镜像仓库 | 40443 | 5000 | Docker Registry |
| YUM仓库 | 40080 | 80 | Nginx HTTP |
| Chart仓库 | 38080 | 8080 | ChartMuseum |
| NFS服务 | 2049 | 2049 | NFS Server |
| NTP服务 | 123 | 123 | NTP Server |
| Agent健康检查 | 58080 | 58080 | BKEAgent Health |
## 五、网络访问要求
### 离线安装
- ✅ 无需外网访问
- ✅ 所有资源本地提供
### 在线安装
- ❌ 需要访问 `hub.oepkgs.net`
- ❌ 需要访问 `cr.openfuyao.cn`
- ❌ 需要访问华为云OBS
- ❌ 需要访问 `cn.pool.ntp.org`
### 私有仓库安装
- ✅ 仅需访问内部私有仓库
- ✅ 支持TLS证书认证
- ✅ 支持用户名密码认证
## 六、安全配置
### TLS证书配置
```bash
bkeadm init \
  --otherRepo registry.internal.company.com/ \
  --imageRepoTLSVerify=true \
  --imageRepoCAFile=/path/to/ca.crt \
  --imageRepoUsername=admin \
  --imageRepoPassword=password
```
### 证书配置路径
```
/etc/containerd/certs.d/registry.internal.company.com:443/ca.crt
/etc/containerd/certs.d/registry.internal.company.com/ca.crt
```
这种设计确保了bkeadm能够灵活适应不同的网络环境和安全要求，同时保持清晰的依赖关系。
        
# bkeadm init 命令的设计思路
## 一、整体架构设计
### 1. 命令入口与参数解析
**文件**: [cmd/init.go](file:///d:/code/github/bkeadm/cmd/init.go)
```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize the boot node",
    Long:  `Initialize the boot node, including node check, warehouse start, cluster installation, and so on`,
}
```
**参数分类**：

| 类别 | 参数 | 说明 |
|------|------|------|
| **基础配置** | `--hostIP`, `--domain`, `--kubernetesPort` | 节点基础网络配置 |
| **仓库配置** | `--imageRepoPort`, `--yumRepoPort`, `--chartRepoPort` | 本地仓库端口配置 |
| **运行时配置** | `--runtime`, `--runtimeStorage` | 容器运行时选择 |
| **在线安装** | `--onlineImage`, `--otherRepo`, `--otherSource`, `--otherChart` | 在线安装相关参数 |
| **版本控制** | `--clusterAPI`, `--oFVersion`, `--versionUrl` | 组件版本控制 |
| **安全配置** | `--imageRepoTLSVerify`, `--imageRepoCAFile`, `--imageRepoUsername`, `--imageRepoPassword` | TLS认证配置 |
| **功能开关** | `--installConsole`, `--enableNTP` | 可选功能开关 |
## 二、初始化流程设计
### 核心流程（[initialize.go:103-138](file:///d:/code/github/bkeadm/pkg/initialize/initialize.go#L103-L138)）
```mermaid
flowchart TD
    A[开始初始化] --> B[节点信息收集]
    B --> C[环境验证]
    C --> D[设置时区/NTP]
    D --> E[准备环境]
    E --> F[容器运行时安装]
    F --> G[仓库服务启动]
    G --> H[Cluster API安装]
    H --> I{是否安装Console?}
    I -->|是| J[Console安装]
    I -->|否| K[生成集群配置]
    J --> K
    K --> L[部署集群]
    L --> M[初始化完成]
```
### 阶段详解
#### **阶段1: 节点信息收集** (`nodeInfo`)
```go
func (op *Options) nodeInfo() {
    h, _ := host.Info()
    c, _ := cpu.Counts(false)
    v, _ := mem.VirtualMemory()
    // 打印主机名、平台、内核、CPU、内存等信息
}
```
#### **阶段2: 环境验证** (`Validate`)
```go
func (op *Options) Validate() error {
    // 1. 解析在线配置
    oc, err = repository.ParseOnlineConfig(...)
    
    // 2. 验证磁盘空间
    if err = op.validateDiskSpace(); err != nil { return err }
    
    // 3. 验证端口占用
    if err = op.validatePorts(); err != nil { return err }
    
    return nil
}
```
#### **阶段3: 时区与NTP设置** (`setTimezone`)
```go
func (op *Options) setTimezone() error {
    // 1. 设置主机时区
    err := timezone.SetTimeZone()
    
    // 2. 配置NTP服务器（如果启用）
    if op.EnableNTP {
        newNTPServer, err := timezone.NTPServer(op.NtpServer, op.HostIP, len(oc.Repo) > 0)
    }
    return nil
}
```
#### **阶段4: 环境准备** (`prepareEnvironment`)
```go
func (op *Options) prepareEnvironment() error {
    // 1. 配置本地源
    op.configLocalSource()
    
    // 2. 设置hosts文件
    syscompat.SetHosts(hostIP, domain)
    
    // 3. 配置私有仓库CA证书
    op.configurePrivateRegistry(clientAuthConfig)
    
    // 4. 初始化仓库（下载源文件、解压等）
    op.initRepositories(clientAuthConfig)
    
    return nil
}
```
#### **阶段5: 容器运行时安装** (`ensureContainerServer`)
```go
func (op *Options) ensureContainerServer() error {
    // 1. 准备仓库依赖（解压镜像包、chart包等）
    repository.PrepareRepositoryDependOn(op.ImageFilePath)
    
    // 2. 验证containerd文件
    result, err := repository.VerifyContainerdFile(op.ImageFilePath)
    
    // 3. 安装运行时
    infrastructure.RuntimeInstall(infrastructure.RuntimeConfig{
        Runtime:        op.Runtime,        // docker 或 containerd
        RuntimeStorage: op.RuntimeStorage,
        Domain:         op.Domain,
        ContainerdFile: containerdFile,
        CniPluginFile:  cniPluginFile,
    })
    
    return nil
}
```
#### **阶段6: 仓库服务启动** (`ensureRepository`)
```go
func (op *Options) ensureRepository() error {
    // 1. 加载本地镜像
    repository.LoadLocalRepository()
    
    // 2. 启动镜像仓库服务
    repository.ContainerServer(op.ImageFilePath, op.ImageRepoPort, oc.Repo, oc.Image)
    
    // 3. 启动YUM仓库服务
    repository.YumServer(op.ImageFilePath, op.ImageRepoPort, op.YumRepoPort, oc.Repo, oc.Image)
    
    // 4. 启动Chart仓库服务
    repository.ChartServer(op.ImageFilePath, op.ImageRepoPort, op.ChartRepoPort, oc.Repo, oc.Image)
    
    // 5. 启动NFS服务（可选）
    repository.NFSServer(op.ImageRepoPort, oc.Repo, oc.Image)
    
    return nil
}
```
#### **阶段7: Cluster API安装** (`ensureClusterAPI`)
```go
func (op *Options) ensureClusterAPI() error {
    // 1. 启动本地Kubernetes（k3s）
    infrastructure.StartLocalKubernetes(k3s.Config{...})
    
    // 2. 生成部署ConfigMap（版本信息）
    op.generateDeployCM()
    
    // 3. 应用containerd配置
    containerd.ApplyContainerdCfg(fmt.Sprintf("%s:%s", op.Domain, op.ImageRepoPort))
    
    // 4. 应用kubelet配置
    kubelet.ApplyKubeletCfg()
    
    // 5. 安装BKEAgent CRD
    bkeagent.InstallBKEAgentCRD()
    
    // 6. 部署Cluster API组件
    clusterapi.DeployClusterAPI(repo, manifestsVersion, providerVersion)
    
    return nil
}
```
#### **阶段8: Console安装** (`ensureConsoleAll`)
```go
func (op *Options) ensureConsoleAll() error {
    if !op.InstallConsole {
        return nil
    }
    
    // 部署BKE Console所有组件
    bkeconsole.DeployConsoleAll(sRestartConfig, repo, op.OFVersion)
    
    return nil
}
```
#### **阶段9: 生成集群配置** (`generateClusterConfig`)
```go
func (op *Options) generateClusterConfig() {
    // 1. 准备配置数据
    data, repo, err := op.prepareClusterConfigData()
    
    // 2. 创建集群配置文件
    op.createClusterConfigFile(data, repo[0], repo[1], repo[2])
    
    // 输出提示信息
    log.BKEFormat(log.HINT, "Run `bke cluster create -f ...` command to deploy the cluster")
}
```
## 三、安装模式设计
### 1. 离线安装模式
```bash
bkeadm init
```
- 使用本地镜像包和源文件
- 启动本地镜像仓库、YUM仓库、Chart仓库
- 所有资源从本地获取
### 2. 在线安装模式
```bash
bkeadm init --onlineImage registry.example.com/openfuyao:v1.0.0
```
- 从远程镜像仓库拉取镜像
- 从远程HTTP服务器下载运行时文件
### 3. 私有仓库模式
```bash
bkeadm init \
  --otherRepo registry.internal.company.com/ \
  --otherSource http://repo.internal.company.com/openfuyao \
  --otherChart chart.internal.company.com/
```
- 使用企业内部私有仓库
- 支持TLS证书认证
### 4. 本地镜像文件模式
```bash
bkeadm init --imageFilePath /path/to/image.tar.gz --otherRepo registry.example.com/
```
- 使用本地镜像文件
- 适用于离线环境快速部署
## 四、参数优先级设计
### 镜像仓库优先级
```go
// 优先级：localImage > otherRepo > onlineImage > 默认值
if localImage != "" {
    image = utils.DefaultLocalImageRegistry
} else if otherRepo != "" {
    image = fmt.Sprintf("%s%s", otherRepo, utils.DefaultLocalImageRegistry)
} else if onlineImage == "" {
    image = utils.DefaultLocalImageRegistry
} else {
    image = fmt.Sprintf("%s/%s", utils.DefaultThirdMirror, utils.DefaultLocalImageRegistry)
}
```
### 仓库路径优先级
```go
// 优先级：ImageFilePath > oc.Repo > (oc.Image为空时使用本地) > 默认值
if op.ImageFilePath != "" {
    repo = localRepoPath
} else if oc.Repo != "" {
    repo = oc.Repo
} else if oc.Image == "" {
    repo = localRepoPath
}
```
## 五、版本管理设计
### 1. 版本配置来源
**离线模式**：从本地patches目录读取
```go
func (op *Options) offlineGenerateDeployCM(patchesDir string) error {
    // 从本地目录读取版本配置文件
    // 生成ConfigMap供后续安装使用
}
```
**在线模式**：从远程下载版本配置
```go
func (op *Options) onlineGenerateDeployCM() error {
    // 从versionUrl下载index.yaml
    // 根据oFVersion下载对应的版本配置文件
    // 生成ConfigMap
}
```
### 2. 版本信息存储
```go
// 存储在ConfigMap中
patchCmKey := fmt.Sprintf("cm.%s", openFuyaoVersion)
patchConfigMap, err := k8sClient.CoreV1().ConfigMaps("openfuyao-patch").Get(...)
```
## 六、安全设计
### 1. TLS证书配置
```go
type CertificateConfig struct {
    TLSVerify    bool
    CAFile       string
    Username     string
    Password     string
    RegistryHost string
    RegistryPort string
}
```
### 2. 证书配置流程
```go
func (op *Options) configurePrivateRegistry(cfg *CertificateConfig) error {
    // 1. 解析仓库地址
    registryHost, registryPort := repository.ParseRegistryHostPort(oc.Repo)
    
    // 2. 配置CA证书
    if cfg.TLSVerify && cfg.CAFile != "" {
        repository.SetupCACertificate(cfg)
    }
    
    return nil
}
```
## 七、设计亮点
### 1. **灵活的安装模式**
- 支持离线、在线、私有仓库、本地镜像文件四种模式
- 参数优先级清晰，易于理解和使用
### 2. **模块化设计**
- 每个阶段独立函数，职责清晰
- 易于测试和维护
### 3. **依赖注入**
```go
type Options struct {
    FS               afero.Fs
    DownloadFunc     func(url, dest string) error
    SetPatchConfigFn func(version, path, key string) error
    K8sClient        k8s.KubernetesClient
}
```
- 支持测试时注入mock对象
- 提高代码可测试性
### 4. **错误处理**
- 每个阶段都有明确的错误处理
- 提供清晰的错误日志
### 5. **版本管理**
- 支持多版本共存
- 通过ConfigMap管理版本配置
- 支持在线和离线版本获取
### 6. **安全默认配置**
- TLS验证默认启用
- 支持CA证书和用户名密码认证
- 私有仓库安全配置
## 八、流程图总结
```mermaid
graph TB
    subgraph "初始化阶段"
        A1[节点信息收集] --> A2[环境验证]
        A2 --> A3[时区/NTP设置]
        A3 --> A4[环境准备]
    end
    
    subgraph "基础设施阶段"
        B1[容器运行时安装] --> B2[仓库服务启动]
        B2 --> B3[Cluster API安装]
    end
    
    subgraph "应用部署阶段"
        C1[Console安装<br/>可选] --> C2[生成集群配置]
        C2 --> C3[部署集群]
    end
    
    A4 --> B1
    B3 --> C1
```

# `bkeadm bke build online-image` 详细设计
## 1. 功能概述
`bke build online-image` 命令用于构建一个**在线安装镜像**，将 BKE 安装所需的所有依赖包、二进制文件打包成一个 Docker 镜像并推送到镜像仓库。该镜像可用于在线安装场景。
## 2. 命令定义
**入口文件**: [cmd/build.go](file:///D:/code/github/bkeadm/cmd/build.go#L98-L120)
```go
var onlineCmd = &cobra.Command{
    Use:   "online-image",
    Short: "Compile an image installed online",
    Long:  `Compile an image installed online`,
    Example: `
# 编译在线安装镜像
bke build online-image -f bke.yaml -t cr.openfuyao.cn/openfuyao/bke-online-installed:latest

# 编译多架构镜像 (默认架构为 amd64)
bke build online-image -f bke.yaml --arch amd64,arm64 -t cr.openfuyao.cn/openfuyao/bke-online-installed:latest
`,
}
```
### 参数说明
| 参数 | 简写 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--file` | `-f` | 是 | - | 配置文件路径 |
| `--target` | `-t` | 是 | - | 目标镜像地址 |
| `--arch` | | 否 | 当前系统架构 | 支持多架构，如 `amd64,arm64` |
| `--strategy` | | 否 | `registry` | 镜像同步策略 (registry/docker) |

## 3. 核心实现
**实现文件**: [pkg/build/onlineimage.go](file:///D:/code/github/bkeadm/pkg/build/onlineimage.go)
### 3.1 执行流程
```
┌─────────────────────────────────────────────────────────────┐
│                    BuildOnlineImage()                        │
├─────────────────────────────────────────────────────────────┤
│  Step 1: 配置文件检查                                        │
│    └─ 读取并解析 YAML 配置文件                               │
├─────────────────────────────────────────────────────────────┤
│  Step 2: 创建工作空间                                        │
│    └─ 在当前目录创建 packages/ 目录结构                       │
├─────────────────────────────────────────────────────────────┤
│  Step 3: 收集依赖包和文件                                    │
│    ├─ 下载二进制文件                       │
│    ├─ 下载 Charts 包                                         │
│    ├─ 下载补丁文件                          │
│    └─ 打包为 source.tar.gz                                   │
├─────────────────────────────────────────────────────────────┤
│  Step 4: 构建镜像                                            │
│    ├─ 单架构: docker build + docker push                     │
│    └─ 多架构: docker buildx build --push                     │
├─────────────────────────────────────────────────────────────┤
│  Step 5: 清理临时文件                                        │
└─────────────────────────────────────────────────────────────┘
```
### 3.2 关键代码逻辑
#### 环境检查
```go
if !infrastructure.IsDocker() {
    log.BKEFormat(log.ERROR, "This build instruction only supports running in docker environment.")
    return
}
```
**限制**: 必须在 Docker 环境中运行。
#### 镜像构建策略
**单架构构建** (架构匹配且不含逗号):
```bash
docker build -t <image_name> .
docker push <image_name>
```

**多架构构建** (使用 buildx):
```bash
docker buildx build --platform=linux/amd64,linux/arm64 -t <image_name> . --push
```
### 3.3 Dockerfile 模板
```dockerfile
FROM scratch
COPY source.tar.gz /bkesource/source.tar.gz
```
生成的镜像极其精简，仅包含一个 `source.tar.gz` 文件。
## 4. 配置文件结构
**配置定义**: [pkg/build/config.go](file:///D:/code/github/bkeadm/pkg/build/config.go#L24-L42)
```yaml
registry:
  imageAddress: cr.openfuyao.cn/openfuyao/registry:2.8.1
  architecture:
    - amd64
    - arm64

repos:          # 镜像仓库配置
  - architecture:
      - amd64
      - arm64
    needDownload: true
    subImages:
      - sourceRepo: cr.openfuyao.cn/openfuyao
        targetRepo: kubernetes
        images:
          - name: registry
            tag: ["2.8.1"]

rpms: []        # RPM 包配置
debs: []        # DEB 包配置

files:          # 二进制文件下载配置
  - address: https://openfuyao.obs.cn-north-4.myhuaweicloud.com/...
    files:
      - fileName: kubectl-v1.33.1-amd64
        fileAlias: ""

charts: []      # Helm Charts 配置
patches: []     # 补丁文件配置
```
### 配置项说明
| 字段 | 类型 | 说明 |
|------|------|------|
| `registry` | struct | 镜像仓库配置 |
| `repos` | []Repo | 需要同步的镜像列表 |
| `rpms` | []Rpm | RPM 依赖包配置 |
| `debs` | []Deb | DEB 依赖包配置 |
| `files` | []File | 需要下载的二进制文件 |
| `charts` | []File | Helm Charts 包 |
| `patches` | []File | 补丁文件 |
## 5. 工作空间目录结构
**定义文件**: [pkg/build/prepare.go](file:///D:/code/github/bkeadm/pkg/build/prepare.go#L35-L53)
```
packages/
├── bke/
│   ├── manifests.yaml
│   └── volumes/
│       └── source.tar.gz      # 最终打包的内容
├── usr/
│   └── bin/
│       └── bke                # bke 二进制文件
└── tmp/
    ├── registry/
    └── packages/
        ├── files/             # 下载的二进制文件
        │   └── patches/       # 补丁文件
        └── charts/            # Charts 包
```
## 6. 文件下载流程
**实现文件**: [pkg/build/sources.go](file:///D:/code/github/bkeadm/pkg/build/sources.go)
```go
func buildRpms(cfg *BuildConfig, stopChan <-chan struct{}) error {
    // 1. 下载文件
    err := downloadFile(cfg, stopChan)
    
    // 2. 文件版本适配
    err = fileVersionAdaptation()
    
    // 3. 重构 charts.tar.gz
    err = buildFileChart()
    
    // 4. 解压 rpm.tar.gz
    err = buildFileRpm()
    
    // 5. 同步 RPM 包
    for _, rpm := range cfg.Rpms {
        err := syncPackage(url, rpm.System, rpm.SystemVersion, ...)
    }
    
    // 6. 打包为 source.tar.gz
    err = global.TarGZ(tmpPackages, fmt.Sprintf("%s/%s", bke, utils.SourceDataFile))
}
```
## 7. 与 `bke build` 的区别
| 特性 | `bke build` | `bke build online-image` |
|------|-------------|--------------------------|
| 输出产物 | tar.gz 压缩包 | Docker 镜像 |
| 镜像处理 | 打包到 `image.tar.gz` | 不包含镜像 |
| 使用场景 | 离线安装 | 在线安装 |
| 推送目标 | 本地文件 | 镜像仓库 |
| 执行步骤 | 8 步 | 5 步 |
## 8. 示例配置
参考 [assets/online-artifacts.yaml](file:///D:/code/github/bkeadm/assets/online-artifacts.yaml):
```yaml
files:
  - address: https://openfuyao.obs.cn-north-4.myhuaweicloud.com/kubernetes/kubernetes/releases/download/of-v1.33.1/bin/linux/amd64/
    files:
      - kubectl-v1.33.1-amd64
      - kubelet-v1.33.1-amd64
  - address: https://openfuyao.obs.cn-north-4.myhuaweicloud.com/containerd/containerd/releases/download/v2.1.1-origin/
    files:
      - containerd-v2.1.1-linux-amd64.tar.gz
      - containerd-v2.1.1-linux-arm64.tar.gz
```
## 9. 使用流程
```bash
# 1. 准备配置文件
bke build config > bke.yaml

# 2. 编辑配置文件 (添加需要的文件、镜像等)

# 3. 构建并推送镜像
bke build online-image -f bke.yaml -t cr.openfuyao.cn/openfuyao/bke-online:v1.0.0

# 4. 多架构构建
bke build online-image -f bke.yaml --arch amd64,arm64 -t cr.openfuyao.cn/openfuyao/bke-online:v1.0.0
```

# `bkeadm bke build` 详细设计
## 1. 功能概述
`bke build` 命令用于构建 **BKE 离线安装包**，将 Kubernetes 集群安装所需的所有依赖（镜像、二进制文件、RPM/DEB 包、Charts 等）打包成一个完整的 tar.gz 压缩包，支持在无网络环境下进行离线安装。
## 2. 命令定义
**入口文件**: [cmd/build.go](file:///D:/code/github/bkeadm/cmd/build.go#L30-L54)
```go
var buildCmd = &cobra.Command{
    Use:   "build",
    Short: "Build the BKE installation package.",
    Long:  `Build the BKE installation package.`,
    Example: `
# Build the BKE installation package.
bke build -f bke.yaml -t bke.tar.gz
`,
}
```
### 参数说明
| 参数 | 简写 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `--file` | `-f` | 是 | - | 配置文件路径 |
| `--target` | `-t` | 否 | 自动生成 | 输出文件路径 |
| `--strategy` | | 否 | `registry` | 镜像同步策略 |
| `--arch` | | 否 | 当前系统架构 | 目标架构 |
### 自动生成文件名规则
```go
bke-{version}-{configName}-{arch}-{timestamp}.tar.gz
// 示例: bke-v1.33.1-bke-amd64-arm64-20250401120000.tar.gz
```
## 3. 核心实现
**实现文件**: [pkg/build/build.go](file:///D:/code/github/bkeadm/pkg/build/build.go)
### 3.1 整体执行流程
```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Build() 主流程                                 │
├─────────────────────────────────────────────────────────────────────────┤
│  Step 1: 配置文件检查                                                    │
│    └─ loadAndVerifyBuildConfig() 读取并验证 YAML 配置                    │
├─────────────────────────────────────────────────────────────────────────┤
│  Step 2: 创建工作空间                                                    │
│    └─ prepareBuildWorkspace() 创建 packages/ 目录结构                    │
├─────────────────────────────────────────────────────────────────────────┤
│  Step 3-6: 并行收集依赖 (两个 goroutine)                                 │
│    ├─ Goroutine A: collectRpmsAndBinary()                               │
│    │   ├─ Step 3: 收集 RPM 包和二进制文件                                │
│    │   └─ Step 4: 收集 bke 二进制文件并获取版本号                         │
│    └─ Goroutine B: collectRegistryImages()                              │
│        ├─ Step 5: 构建本地镜像仓库                                       │
│        └─ Step 6: 同步源仓库镜像到目标仓库                                │
├─────────────────────────────────────────────────────────────────────────┤
│  Step 7: 打包最终产物                                                    │
│    └─ createFinalPackage() 生成 tar.gz 压缩包                            │
├─────────────────────────────────────────────────────────────────────────┤
│  Step 8: 完成输出                                                        │
└─────────────────────────────────────────────────────────────────────────┘
```
### 3.2 核心代码结构
```go
func (o *Options) Build() {
    // Step 1: 配置文件检查
    cfg, err := loadAndVerifyBuildConfig(o.File)
    
    // Step 2: 创建工作空间
    if err := prepareBuildWorkspace(); err != nil { return }
    
    // Step 3-6: 并行收集依赖和镜像
    version, err := o.collectDependenciesAndImages(cfg)
    
    // Step 7: 创建最终安装包
    if err := o.createFinalPackage(cfg, version); err != nil { return }
    
    // Step 8: 完成
    log.BKEFormat("step.8", fmt.Sprintf("Packaging complete %s", o.Target))
}
```
## 4. 配置文件结构
**配置定义**: [pkg/build/config.go](file:///D:/code/github/bkeadm/pkg/build/config.go#L24-L42)
```yaml
registry:
  imageAddress: hub.oepkgs.net/openfuyao/registry:2.8.1  # 本地镜像仓库镜像
  architecture:
    - amd64
    - arm64

openFuyaoVersion: "1.0.0"
kubernetesVersion: "v1.33.1"
etcdVersion: "v3.5.6"
containerdVersion: "v2.1.1"

repos:          # 镜像仓库配置
  - architecture:
      - amd64
      - arm64
    needDownload: true
    isKubernetes: false
    subImages:
      - sourceRepo: cr.openfuyao.cn/openfuyao
        targetRepo: kubernetes
        imageTrack: ""  # 可选：动态标签追踪
        images:
          - name: coredns
            tag: ["1.12.2-of.1"]
            usedPodInfo:
              - podPrefix: coredns
                namespace: kube-system

rpms:          # RPM 包配置
  - address: http://127.0.0.1:40080/
    system: ["CentOS"]
    systemVersion: ["7", "8"]
    systemArchitecture: ["amd64", "arm64"]
    directory: ["docker-ce", "kubectl"]

debs: []       # DEB 包配置

files:         # 二进制文件下载配置
  - address: https://openfuyao.obs.cn-north-4.myhuaweicloud.com/...
    files:
      - fileName: kubectl-v1.33.1-amd64
        fileAlias: ""  # 可选：重命名

charts: []     # Helm Charts 配置
patches: []    # 补丁文件配置
```
### 配置项详细说明
| 字段 | 类型 | 说明 |
|------|------|------|
| `registry` | struct | 本地镜像仓库配置 |
| `registry.imageAddress` | string | Registry 镜像地址 |
| `registry.architecture` | []string | 支持的架构列表 |
| `repos` | []Repo | 需要同步的镜像仓库列表 |
| `repos[].needDownload` | bool | 是否需要下载 |
| `repos[].isKubernetes` | bool | 是否为 Kubernetes 核心组件 |
| `repos[].subImages[].imageTrack` | string | 动态标签追踪 URL |
| `rpms` | []Rpm | RPM 包配置 |
| `debs` | []Deb | DEB 包配置 |
| `files` | []File | 需要下载的二进制文件 |
| `charts` | []File | Helm Charts 包 |
| `patches` | []File | 补丁文件 |
## 5. 工作空间目录结构
**定义文件**: [pkg/build/prepare.go](file:///D:/code/github/bkeadm/pkg/build/prepare.go#L35-L53)
```
packages/
├── bke/                          # BKE 安装包核心目录
│   ├── manifests.yaml            # 配置清单文件
│   └── volumes/
│       ├── image.tar.gz          # 镜像数据文件
│       ├── registry.image-amd64  # Registry 镜像文件
│       ├── registry.image-arm64  # Registry 镜像文件
│       └── source.tar.gz         # 源数据文件
├── usr/
│   └── bin/
│       ├── bke                   # bke 主程序
│       ├── bke_amd64             # 多架构二进制
│       └── bke_arm64
└── tmp/                          # 临时工作目录
    ├── registry/                 # 镜像同步临时目录
    │   └── oci-layout/           # OCI layout 格式目录
    └── packages/
        ├── files/                # 下载的二进制文件
        │   └── patches/          # 补丁文件
        └── charts/               # Charts 包
            ├── CentOS/
            │   ├── 7/
            │   │   ├── amd64/
            │   │   └── arm64/
            │   └── 8/
            │       ├── amd64/
            │       └── arm64/
            └── Ubuntu/
                └── 22/
                    ├── amd64/
                    └── arm64/
```
## 6. 镜像同步策略
`bke build` 支持三种镜像同步策略，通过 `--strategy` 参数指定：
### 6.1 Registry 策略 (默认)
**实现文件**: [pkg/build/registrysyncimage.go](file:///D:/code/github/bkeadm/pkg/build/registrysyncimage.go)

**原理**: 启动本地 Registry 容器，使用 `skopeo` 或 `crane` 工具同步镜像。
```go
func syncRepo(cfg *BuildConfig, stopChan chan struct{}) error {
    // 1. 启动本地 Registry
    server.StartImageRegistry(utils.LocalImageRegistryName, cfg.Registry.ImageAddress, "5000", tmpRegistry)
    
    // 2. 遍历所有镜像仓库配置
    for _, cr := range cfg.Repos {
        if !cr.NeedDownload { continue }
        processRepoImages(cr, stopChan)
    }
    
    // 3. 打包镜像并清理
    return packImageAndCleanup()
}
```
**特点**:
- 需要启动 Docker 容器
- 支持多架构镜像同步
- 使用 `reg.CopyRegistry()` 进行镜像复制
### 6.2 Docker 策略
**实现文件**: [pkg/build/transfersyncimage.go](file:///D:/code/github/bkeadm/pkg/build/transfersyncimage.go)

**原理**: 使用 Docker 命令 拉取、标记、推送镜像。
```go
func collectRepo(cfg *BuildConfig, stopChan <-chan struct{}) error {
    // 1. 启动本地 Registry
    server.StartImageRegistry(...)
    
    // 2. 创建镜像处理通道
    imageChan := make(chan docker.ImageRef, 100)
    
    // 3. 启动推送协程
    go pushImage(imageChan, pullCompleteChan, pushCompleteChan, internalStopChan)
    
    // 4. 同步所有镜像
    syncAllRepoImages(cfg, channels)
    
    // 5. 打包镜像
    return packImageAndCleanup()
}
```
**特点**:
- 依赖 Docker 环境
- 使用生产者-消费者模式并发处理
- 支持重试机制
### 6.3 OCI 策略
**实现文件**: [pkg/build/ocisyncimage.go](file:///D:/code/github/bkeadm/pkg/build/ocisyncimage.go)

**原理**: 使用 OCI Layout 格式存储镜像，无需 Docker/Containerd。
```go
func syncRepoOCI(cfg *BuildConfig, stopChan chan struct{}) error {
    // 1. 创建 OCI Layout 目录结构
    ociDir, err := createOCILayoutStructure()
    
    // 2. 统计总镜像数
    totalImages := countTotalImages(cfg)
    
    // 3. 同步所有镜像到 OCI Layout
    syncAllImagesToOCI(cfg, ociDir, stopChan, totalImages)
    
    // 4. 移动到 volumes 目录
    return moveOCILayoutToVolumes(ociDir)
}
```
**OCI Layout 目录结构**:
```
oci-layout/
├── oci-layout           # {"imageLayoutVersion":"1.0.0"}
├── index.json           # 镜像索引
├── blobs/
│   └── sha256/          # 镜像层数据
│       ├── manifest/
│       ├── config/
│       └── layer/
└── refs/                # 镜像引用
```
**特点**:
- 无需 Docker 环境
- 更轻量级
- 支持标准 OCI 格式
## 7. 镜像同步详细流程
### 7.1 单架构镜像同步
```go
func syncSingleArchImage(source, target, arch string, srcTLSVerify bool) error {
    imageAddress := source
    
    // 1. 尝试直接拉取
    op := reg.Options{
        MultiArch:     false,
        SrcTLSVerify:  srcTLSVerify,
        DestTLSVerify: false,
        Arch:          arch,
        Source:        imageAddress,
        Target:        target,
    }
    
    if err := reg.CopyRegistry(op); err != nil {
        // 2. 失败则尝试添加架构后缀
        imageAddress = imageAddress + "-" + arch
        op.Source = imageAddress
        return reg.CopyRegistry(op)
    }
    return nil
}
```
### 7.2 多架构镜像同步
```go
func syncMultiArchImage(source, target string, arch []string, srcTLSVerify bool) error {
    // 逐个架构拉取并创建多架构 manifest
    return syncArchImagesAndCreateManifest(source, target, arch, srcTLSVerify)
}

func syncArchImagesAndCreateManifest(source, target string, arch []string, srcTLSVerify bool) error {
    var img []reg.ImageArch
    
    for _, ar := range arch {
        // 1. 同步单架构镜像
        archImg, err := syncSingleArchVariant(source, target, ar, op)
        img = append(img, archImg)
    }
    
    // 2. 创建多架构 manifest
    return reg.CreateMultiArchImage(img, target)
}
```
### 7.3 镜像标签格式支持
支持多种镜像标签格式：

| 格式 | 示例 | 说明 |
|------|------|------|
| 标准格式 | `alpine:3.15` | 多架构镜像 |
| 架构后缀 | `alpine:3.15-amd64` | 单架构镜像 |
| 动态标签 | `alpine:v4.0-*-202502051112` | 使用 `-*-` 占位符 |

```go
// 动态标签处理
if strings.Contains(tag, cut) {  // cut = "-*-"
    imageAddress = strings.ReplaceAll(imageAddress, cut, fmt.Sprintf("-%s-", arch))
}
```
## 8. 动态标签追踪
**实现文件**: [pkg/build/repotrack.go](file:///D:/code/github/bkeadm/pkg/build/repotrack.go)

支持从不同镜像仓库 API 获取最新标签：
### 8.1 支持的仓库类型
| 类型 | 标识 | API 格式 |
|------|------|----------|
| DockerHub | `dockerhub` | Docker Hub API |
| Nexus | `nexus@http://user:pass@nexus.com/` | Nexus REST API |
| Harbor | `harbor@http://user:pass@harbor.com/` | Harbor API v2 |
| Registry | `registry@http://registry.com/` | Registry v2 API |
### 8.2 使用示例
```yaml
subImages:
  - sourceRepo: cr.openfuyao.cn/openfuyao
    targetRepo: kubernetes
    imageTrack: "harbor@http://admin:password@harbor.example.com/"
    images:
      - name: my-image
        tag: ["latest"]  # 将被替换为实际最新标签
```

```go
func imageTrack(sourceRepo, imageTrack, imageName, imageTag string, arch []string) (string, error) {
    if len(imageTrack) == 0 || strings.Contains(imageTag, cut) {
        // 直接使用配置的标签
        return fmt.Sprintf("%s/%s:%s", sourceRepo, imageName, imageTag), nil
    }
    
    repo, url := splitRepo1(imageTrack)
    switch repo {
    case DockerHub:
        imageTagList, err = dockerHubTags(imageName)
    case Nexus:
        imageTagList, err = nexusTags(newUrl, imageName)
    case Harbor:
        imageTagList, err = harborTags(newUrl, projectName, imageName)
    case Registry:
        imageTagList, err = registryTags(newUrl, imageName)
    }
    
    // 返回最新标签
    return getLatestTag(imageTagList, arch)
}
```
## 9. 文件下载流程
**实现文件**: [pkg/build/sources.go](file:///D:/code/github/bkeadm/pkg/build/sources.go)
### 9.1 下载流程
```go
func buildRpms(cfg *BuildConfig, stopChan <-chan struct{}) error {
    // 1. 下载配置文件中指定的文件
    err := downloadFile(cfg, stopChan)
    
    // 2. 文件版本适配 (重命名)
    err = fileVersionAdaptation()
    
    // 3. 重构 charts.tar.gz
    err = buildFileChart()
    
    // 4. 解压 rpm.tar.gz
    err = buildFileRpm()
    
    // 5. 同步 RPM 包
    for _, rpm := range cfg.Rpms {
        syncPackage(url, rpm.System, rpm.SystemVersion, rpm.SystemArchitecture, rpm.Directory)
    }
    
    // 6. 打包为 source.tar.gz
    return global.TarGZ(tmpPackages, fmt.Sprintf("%s/%s", bke, utils.SourceDataFile))
}
```
### 9.2 RPM 包下载
```go
func syncPackage(url string, systems, versions, architectures, directories []string) error {
    for _, s := range systems {
        for _, v := range versions {
            for _, ar := range architectures {
                for _, d := range directories {
                    // 拼接下载 URL
                    downloadUrl := fmt.Sprintf("%s%s/%s/%s/%s/", url, system, version, arch, directory)
                    
                    // 下载目录下所有文件
                    utils.DownloadAllFiles(downloadUrl, downloadDirectory)
                }
                
                // 创建 yum 仓库元数据
                cmd := fmt.Sprintf("createrepo %s", path.Join(tmpPackages, system, version, arch))
                global.Command.ExecuteCommandWithOutput("sh", "-c", cmd)
            }
        }
    }
}
```
### 9.3 文件下载目录结构
```
tmp/packages/
├── files/
│   ├── kubectl-v1.33.1-amd64
│   ├── kubelet-v1.33.1-amd64
│   ├── containerd-v2.1.1-linux-amd64.tar.gz
│   ├── cni-plugins-linux-amd64-v1.4.1.tgz
│   ├── helm-v3.14.2-linux-amd64.tar.gz
│   └── patches/
│       └── patch-v1.0.0.tar.gz
├── charts/
│   └── charts.tar.gz
├── CentOS/
│   ├── 7/
│   │   ├── amd64/
│   │   │   ├── docker-ce/
│   │   │   │   └── *.rpm
│   │   │   └── repodata/
│   │   └── arm64/
│   └── 8/
│       └── ...
└── Ubuntu/
    └── 22/
        └── ...
```
## 10. BKE 二进制处理
**实现文件**: [pkg/build/sources.go](file:///D:/code/github/bkeadm/pkg/build/sources.go#L201-L262)
```go
func buildBkeBinary() (string, error) {
    // 1. 查找 bke 二进制文件
    bkeBinaryList, err := findBkeBinaries()
    // 支持的命名: bke, bke_amd64, bke_arm64, bkeadm_*
    
    if len(bkeBinaryList) == 0 {
        return "", errors.New("the files list must contain bke binary file")
    }
    
    if len(bkeBinaryList) == 1 {
        // 单一二进制
        return installSingleBkeBinary(bkeBinaryList[0])
    }
    
    // 多架构二进制
    return installMultipleBkeBinaries(bkeBinaryList)
}

func installSingleBkeBinary(bkeName string) (string, error) {
    // 1. 复制到 usr/bin/bke
    utils.CopyFile(sourceBKE, targetBKE)
    
    // 2. 设置可执行权限
    os.Chmod(targetBKE, utils.ExecutableFilePermission)
    
    // 3. 获取版本号
    version, _ := global.Command.ExecuteCommandWithOutput("sh", "-c", fmt.Sprintf("%s version only", targetBKE))
    
    return version, nil
}
```
## 11. 最终打包
```go
func (o *Options) createFinalPackage(cfg *BuildConfig, version string) error {
    // 1. 生成默认文件名 (如果未指定)
    if len(o.Target) == 0 {
        fileInfo, _ := os.Stat(o.File)
        o.Target = path.Join(pwd, fmt.Sprintf("bke-%s-%s-%s-%s.tar.gz", 
            version,
            strings.TrimSuffix(fileInfo.Name(), ".yaml"),
            strings.Join(cfg.Registry.Architecture, "-"),
            time.Now().Format("20060102150405")))
    }
    
    // 2. 压缩打包
    return compressedPackage(cfg, o.Target)
}

func compressedPackage(cfg *BuildConfig, target string) error {
    // 1. 写入 manifests.yaml
    writeManifestsFile(cfg, path.Join(bke, "manifests.yaml"))
    
    // 2. 删除临时目录
    os.RemoveAll(tmp)
    
    // 3. 压缩打包 (保留文件权限)
    return global.TaeGZWithoutChangeFile(packages, target)
}
```
## 12. 与 `bke build online-image` 对比

| 特性 | `bke build` | `bke build online-image` |
|------|-------------|--------------------------|
| **输出产物** | tar.gz 压缩包 | Docker 镜像 |
| **镜像处理** | 打包到 `image.tar.gz` | 不包含镜像 |
| **使用场景** | 离线安装 | 在线安装 |
| **推送目标** | 本地文件系统 | 镜像仓库 |
| **执行步骤** | 8 步 | 5 步 |
| **环境要求** | Docker 环境 | Docker 环境 |
| **多架构支持** | ✅ | ✅ (buildx) |
| **镜像仓库** | 包含完整镜像仓库 | 不包含 |

## 13. 使用示例
### 13.1 生成默认配置
```bash
bke build config > bke.yaml
```
### 13.2 编辑配置文件
```yaml
registry:
  imageAddress: hub.oepkgs.net/openfuyao/registry:2.8.1
  architecture: ["amd64", "arm64"]

repos:
  - architecture: ["amd64", "arm64"]
    needDownload: true
    subImages:
      - sourceRepo: cr.openfuyao.cn/openfuyao
        targetRepo: kubernetes
        images:
          - name: coredns
            tag: ["1.12.2-of.1"]

files:
  - address: https://openfuyao.obs.cn-north-4.myhuaweicloud.com/kubernetes/kubernetes/releases/download/of-v1.33.1/bin/linux/amd64/
    files:
      - fileName: kubectl-v1.33.1-amd64
```
### 13.3 构建安装包
```bash
# 基本构建
bke build -f bke.yaml -t bke-v1.0.0.tar.gz

# 使用 OCI 策略 (无需 Docker)
bke build -f bke.yaml --strategy oci

# 指定架构
bke build -f bke.yaml --arch amd64,arm64
```
## 14. 架构图
```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              bke build 架构                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────┐    ┌─────────────────────────────────────────────────┐    │
│  │  bke.yaml   │───▶│              Build Pipeline                     │    │
│  └─────────────┘    └─────────────────────────────────────────────────┘    │
│                                    │                                        │
│                                    ▼                                        │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                        Step 1-2: 初始化                               │  │
│  │  ┌────────────────┐    ┌────────────────┐    ┌────────────────┐     │  │
│  │  │ 配置文件验证    │───▶│ 工作空间创建    │───▶│ 目录结构初始化  │     │  │
│  │  └────────────────┘    └────────────────┘    └────────────────┘     │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                    │                                        │
│                    ┌───────────────┴───────────────┐                       │
│                    ▼                               ▼                       │
│  ┌─────────────────────────────┐   ┌─────────────────────────────┐        │
│  │   Step 3-4: 依赖收集        │   │   Step 5-6: 镜像同步        │        │
│  │  ┌─────────────────────┐   │   │  ┌─────────────────────┐   │        │
│  │  │ 下载二进制文件       │   │   │  │ 启动本地 Registry    │   │        │
│  │  └─────────────────────┘   │   │  └─────────────────────┘   │        │
│  │  ┌─────────────────────┐   │   │  ┌─────────────────────┐   │        │
│  │  │ 下载 Charts 包       │   │   │  │ 同步源仓库镜像       │   │        │
│  │  └─────────────────────┘   │   │  └─────────────────────┘   │        │
│  │  ┌─────────────────────┐   │   │  ┌─────────────────────┐   │        │
│  │  │ 同步 RPM/DEB 包      │   │   │  │ 打包镜像数据         │   │        │
│  │  └─────────────────────┘   │   │  └─────────────────────┘   │        │
│  │  ┌─────────────────────┐   │   │                            │        │
│  │  │ 打包 source.tar.gz   │   │   │  策略: registry/docker/oci │        │
│  │  └─────────────────────┘   │   │                            │        │
│  └─────────────────────────────┘   └─────────────────────────────┘        │
│                    └───────────────┬───────────────┘                       │
│                                    ▼                                        │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                        Step 7-8: 打包输出                             │  │
│  │  ┌────────────────┐    ┌────────────────┐    ┌────────────────┐     │  │
│  │  │ 生成 manifests  │───▶│ 清理临时文件    │───▶│ 压缩打包输出    │     │  │
│  │  └────────────────┘    └────────────────┘    └────────────────┘     │  │
│  └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│  输出: bke-{version}-{config}-{arch}-{timestamp}.tar.gz                    │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

# bke build与bke build online-image
## 代码复用分析
### 当前两个命令的流程对比
| 步骤 | `bke build` | `bke build online-image` | 是否复用 |
|------|-------------|--------------------------|----------|
| Step 1 | 配置文件检查 | 配置文件检查 | ❌ 重复代码 |
| Step 2 | 创建工作空间 | 创建工作空间 | ✅ 复用 `prepare()` |
| Step 3 | 收集 RPM/文件 | 收集 RPM/文件 | ✅ 复用 `buildRpms()` |
| Step 4 | 收集 bke 二进制 | ❌ 无 | - |
| Step 5 | 构建镜像仓库 | ❌ 无 | - |
| Step 6 | 同步镜像 | ❌ 无 | - |
| Step 7 | 打包 tar.gz | 构建镜像 | ❌ 不同 |
| Step 8 | 完成 | 完成 | - |

### 当前问题
**`online-image` 存在的问题**:
1. 只包含 `source.tar.gz`（二进制文件、RPM 包）
2. **不包含镜像数据** (`image.tar.gz`)
3. **不包含 bke 二进制文件**
4. 本质上只是一个"在线安装辅助镜像"，不是完整的离线安装包

**代码重复问题**:
```go
// onlineimage.go - 重复代码
cfg := &BuildConfig{}
yamlFile, err := os.ReadFile(o.File)
if err != nil { ... }
if err = yaml.Unmarshal(yamlFile, cfg); err != nil { ... }

// build.go - 同样的逻辑
cfg := &BuildConfig{}
yamlFile, err := os.ReadFile(file)
if err != nil { ... }
if err = yaml.Unmarshal(yamlFile, cfg); err != nil { ... }
```
### 优化方案
**你的建议完全可行**：先构建离线 tar 包，再封装成镜像。
#### 方案一：直接复用 Build() 函数
```go
func (o *Options) BuildOnlineImage() {
    // 1. 先构建离线包
    tarPath := fmt.Sprintf("bke-online-%d.tar.gz", time.Now().Unix())
    o.Target = tarPath
    o.Build()  // 复用完整的 Build 流程
    
    // 2. 将 tar 包封装成镜像
    err := o.buildImageFromTar(tarPath)
    if err != nil {
        log.BKEFormat(log.ERROR, fmt.Sprintf("Build image failures %s", err.Error()))
        return
    }
    
    // 3. 清理临时文件
    os.RemoveAll(tarPath)
    log.BKEFormat("step.6", "Push the image to the registry completed")
}

func (o *Options) buildImageFromTar(tarPath string) error {
    err := os.Mkdir(pwd+"/bkesource", utils.DefaultDirPermission)
    if err != nil {
        return err
    }
    defer os.RemoveAll(pwd + "/bkesource")
    
    dockerfile := `
FROM scratch
COPY bke.tar.gz /bke/bke.tar.gz
`
    err = os.WriteFile(pwd+"/bkesource/Dockerfile", []byte(dockerfile), utils.DefaultFilePermission)
    if err != nil {
        return err
    }
    
    // 复制 tar 包
    err = utils.CopyFile(tarPath, pwd+"/bkesource/bke.tar.gz")
    if err != nil {
        return err
    }
    
    // 构建并推送镜像
    return buildAndPushImage(o.Target, o.Arch)
}
```
#### 方案二：提取公共函数
```go
// 提取公共的配置加载函数
func loadBuildConfig(file string) (*BuildConfig, error) {
    cfg := &BuildConfig{}
    yamlFile, err := os.ReadFile(file)
    if err != nil {
        return nil, fmt.Errorf("failed to read file: %w", err)
    }
    if err = yaml.Unmarshal(yamlFile, cfg); err != nil {
        return nil, fmt.Errorf("unable to unmarshal file: %w", err)
    }
    if err = verifyConfigContent(cfg); err != nil {
        return nil, fmt.Errorf("config verification failed: %w", err)
    }
    return cfg, nil
}

// 提取公共的镜像构建函数
func buildAndPushImage(imageName, arch, sourceDir string) error {
    if strings.Contains(arch, runtime.GOARCH) && !strings.Contains(arch, ",") {
        // 单架构构建
        output, err := global.Command.ExecuteCommandWithOutput("sh", "-c",
            fmt.Sprintf("cd %s && docker build -t %s .", sourceDir, imageName))
        if err != nil {
            return errors.New(output + err.Error())
        }
        output, err = global.Command.ExecuteCommandWithOutput("sh", "-c",
            fmt.Sprintf("docker push %s", imageName))
        if err != nil {
            return errors.New(output + err.Error())
        }
    } else {
        // 多架构构建
        platform := buildPlatformString(arch)
        output, err := global.Command.ExecuteCommandWithOutput("sh", "-c",
            fmt.Sprintf("cd %s && docker buildx build --platform=%s -t %s . --push", 
                sourceDir, platform, imageName))
        if err != nil {
            return errors.New(output + err.Error())
        }
    }
    return nil
}
```
### 优化后的架构

```
┌─────────────────────────────────────────────────────────────────┐
│                      重构后的架构                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    公共函数层                            │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │loadConfig()  │  │ prepare()    │  │ buildRpms()  │  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐  │   │
│  │  │buildRegistry │  │ syncRepo()   │  │buildBkeBinary│  │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘  │   │
│  │  ┌──────────────┐  ┌──────────────┐                    │   │
│  │  │compressPkg() │  │buildDockerImg│                    │   │
│  │  └──────────────┘  └──────────────┘                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                              │                                  │
│              ┌───────────────┴───────────────┐                 │
│              ▼                               ▼                  │
│  ┌───────────────────────┐     ┌───────────────────────┐      │
│  │     bke build         │     │  bke build online-image│      │
│  │  ┌─────────────────┐  │     │  ┌─────────────────┐  │      │
│  │  │ 1. loadConfig   │  │     │  │ 1. loadConfig   │  │      │
│  │  │ 2. prepare      │  │     │  │ 2. prepare      │  │      │
│  │  │ 3. buildRpms    │  │     │  │ 3. buildRpms    │  │      │
│  │  │ 4. buildBkeBin  │  │     │  │ 4. buildBkeBin  │  │      │
│  │  │ 5. buildRegistry│  │     │  │ 5. buildRegistry│  │      │
│  │  │ 6. syncRepo     │  │     │  │ 6. syncRepo     │  │      │
│  │  │ 7. compressPkg  │  │     │  │ 7. compressPkg  │  │      │
│  │  │ 8. 输出 tar.gz  │  │     │  │ 8. buildDockerImg│  │      │
│  │  └─────────────────┘  │     │  │ 9. 输出镜像     │  │      │
│  │                        │     │  └─────────────────┘  │      │
│  │  输出: bke.tar.gz     │     │  输出: bke:v1.0 镜像  │      │
│  └───────────────────────┘     └───────────────────────┘      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```
### 优化收益
| 方面 | 优化前 | 优化后 |
|------|--------|--------|
| **代码复用** | 部分复用 | 完全复用 |
| **online-image 功能** | 不完整 | 完整离线包 |
| **维护成本** | 两套逻辑 | 一套逻辑 |
| **构建时间** | 分别构建 | 可复用缓存 |
| **产物一致性** | 不一致 | 完全一致 |

### 建议的实现
```go
// build.go
func (o *Options) Build() {
    if !infrastructure.IsDocker() {
        log.BKEFormat(log.ERROR, "This build instruction only supports running in docker environment.")
        return
    }

    cfg, err := loadBuildConfig(o.File)
    if err != nil {
        log.BKEFormat(log.ERROR, err.Error())
        return
    }

    if err := prepareBuildWorkspace(); err != nil {
        log.BKEFormat(log.ERROR, err.Error())
        return
    }

    version, err := o.collectDependenciesAndImages(cfg)
    if err != nil {
        log.BKEFormat(log.ERROR, err.Error())
        return
    }

    if err := o.createFinalPackage(cfg, version); err != nil {
        log.BKEFormat(log.ERROR, err.Error())
        return
    }

    log.BKEFormat("step.8", fmt.Sprintf("Packaging complete %s", o.Target))
}

// onlineimage.go - 重构后
func (o *Options) BuildOnlineImage() {
    if !infrastructure.IsDocker() {
        log.BKEFormat(log.ERROR, "This build instruction only supports running in docker environment.")
        return
    }

    // 1. 复用 Build() 生成离线包
    tarPath := o.generateTempTarPath()
    o.Target = tarPath
    o.Build()
    
    // 检查 tar 包是否生成成功
    if !utils.Exists(tarPath) {
        log.BKEFormat(log.ERROR, "Failed to create offline package")
        return
    }

    // 2. 将 tar 包封装成 Docker 镜像
    log.BKEFormat("step.9", fmt.Sprintf("Build the image %s ...", o.Target))
    if err := o.buildImageFromTar(tarPath); err != nil {
        log.BKEFormat(log.ERROR, fmt.Sprintf("Build image failures %s", err.Error()))
        return
    }

    // 3. 清理
    os.RemoveAll(tarPath)
    log.BKEFormat("step.10", "Push the image to the registry completed")
}

func (o *Options) buildImageFromTar(tarPath string) error {
    sourceDir := path.Join(pwd, "bkesource")
    if err := os.MkdirAll(sourceDir, utils.DefaultDirPermission); err != nil {
        return err
    }
    defer os.RemoveAll(sourceDir)

    // 创建 Dockerfile
    dockerfile := `FROM scratch
COPY bke.tar.gz /bke/bke.tar.gz
`
    if err := os.WriteFile(path.Join(sourceDir, "Dockerfile"), []byte(dockerfile), utils.DefaultFilePermission); err != nil {
        return err
    }

    // 复制 tar 包
    if err := utils.CopyFile(tarPath, path.Join(sourceDir, "bke.tar.gz")); err != nil {
        return err
    }

    // 构建并推送镜像
    return buildAndPushDockerImage(o.Target, o.Arch, sourceDir)
}
```
### 总结
**你的建议非常合理**，可以实现：
1. ✅ **代码完全复用** - `online-image` 直接调用 `Build()` 函数
2. ✅ **功能完整** - 生成的镜像包含完整的离线安装包
3. ✅ **维护简单** - 只需维护一套构建逻辑
4. ✅ **产物一致** - 两种方式生成的离线包内容完全一致

这种设计确保了初始化过程的**可靠性**、**灵活性**和**可维护性**，同时支持多种部署场景。

      # `bke build rpm` 详细设计
## 1. 功能概述
`bke build rpm` 命令用于构建**离线 RPM/DEB 软件包仓库**，支持为多种操作系统（CentOS、Ubuntu、Kylin）和架构创建本地 YUM/APT 软件源。生成的 `rpm.tar.gz` 可用于 BKE 离线安装场景。
## 2. 命令定义
**入口文件**: [cmd/build.go](file:///D:/code/github/bkeadm/cmd/build.go#L131-L156)
```go
var rpmCmd = &cobra.Command{
    Use:   "rpm",
    Short: "Build an offline rpm package",
    Long:  `Build an offline rpm package`,
    Example: `
# 初始化 rpm 包
# rpm 是一个空目录或包含多个包的源目录
bke build rpm --source rpm

# 为已存在的 rpm.tar.gz 添加新的 rpm 包
bke build rpm --source rpm.tar.gz --add centos/8/amd64 --package docker-ce

# 自定义镜像仓库
bke build rpm --source rpm.tar.gz --add centos/8/amd64 --package docker-ce --registry cr.openfuyao.cn/openfuyao

# 仅构建指定系统的 rpm
bke build rpm --add centos/8/amd64 --package docker-ce
`,
}
```
### 参数说明
| 参数 | 必填 | 默认值 | 说明 |
|------|------|--------|------|
| `--source` | 否 | - | 源 RPM 文件路径，可以是目录或 `rpm.tar.gz` 文件 |
| `--add` | 否 | - | 添加 RPM 包的目标系统路径，如 `centos/8/amd64` |
| `--package` | 否 | - | 要添加的包目录路径 |
| `--registry` | 否 | `registry.cn-hangzhou.aliyuncs.com/bocloud` | 构建镜像仓库地址 |
### 参数校验规则
```go
PreRunE: func(cmd *cobra.Command, args []string) error {
    // add 和 package 必须同时指定或同时为空
    if (rpmOption.Add == "") != (rpmOption.Package == "") {
        return errors.New("The parameter `add` or `package` is required. ")
    }
    return nil
}
```
## 3. 支持的操作系统和架构
**定义文件**: [pkg/build/buildrpm.go](file:///D:/code/github/bkeadm/pkg/build/buildrpm.go#L42-L52)
```go
var adds = map[string]string{
    "centos/7/amd64":  "CentOS/7/amd64",
    "centos/7/arm64":  "CentOS/7/arm64",
    "centos/8/amd64":  "CentOS/8/amd64",
    "centos/8/arm64":  "CentOS/8/arm64",
    "ubuntu/22/amd64": "Ubuntu/22/amd64",
    "ubuntu/22/arm64": "Ubuntu/22/arm64",
    "kylin/v10/arm64": "Kylin/V10/arm64",
    "kylin/v10/amd64": "Kylin/V10/amd64",
}
```

| 操作系统 | 版本 | 架构 | 包格式 | 构建工具 |
|----------|------|------|--------|----------|
| CentOS | 7 | amd64, arm64 | RPM | createrepo |
| CentOS | 8 | amd64, arm64 | RPM | createrepo + repo2module |
| Ubuntu | 22 | amd64, arm64 | DEB | dpkg-scanpackages |
| Kylin | V10 | amd64, arm64 | RPM | createrepo |
## 4. 核心实现
**实现文件**: [pkg/build/buildrpm.go](file:///D:/code/github/bkeadm/pkg/build/buildrpm.go)
### 4.1 执行流程
```
┌─────────────────────────────────────────────────────────────────────────┐
│                         RpmOptions.Build()                              │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  无参数时: 输出目录结构                                                   │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  consoleOutputStruct()  # 打印 rpm 目录结构                      │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  参数校验:                                                               │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  1. 验证 add 参数是否有效                                         │   │
│  │  2. 验证 package 目录结构                                         │   │
│  │  3. 获取 source/package 绝对路径                                  │   │
│  │  4. 检查 Docker 环境                                              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│  执行构建: executeBuild()                                               │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │  场景 1: --add + --package (无 source)                           │   │
│  │    └─ rmpBuild() → 构建单个系统的 rpm 包                          │   │
│  │                                                                   │   │
│  │  场景 2: --source (无 add/package)                               │   │
│  │    └─ rpmBuildPackage() → 构建完整的 rpm.tar.gz                  │   │
│  │                                                                   │   │
│  │  场景 3: --source + --add + --package                            │   │
│  │    └─ rpmPackageAddOne() → 向已有包添加新 rpm                     │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```
### 4.2 三种使用场景
#### 场景 1: 构建单个系统的 RPM 包
```bash
bke build rpm --add centos/8/amd64 --package ./docker-ce
```

```go
func rmpBuild(registry string, add string, absPath string) error {
    switch add {
    case "centos/7/amd64", "centos/7/arm64":
        return rpmCentos7Build(registry, absPath)
    case "centos/8/amd64", "centos/8/arm64":
        return rpmCentos8Build(registry, absPath)
    case "ubuntu/22/amd64", "ubuntu/22/arm64":
        return rpmUbuntu22Build(registry, absPath)
    case "kylin/v10/amd64", "kylin/v10/arm64":
        return rpmKylinV10Build(registry, absPath)
    }
}
```
#### 场景 2: 构建完整的 rpm.tar.gz
```bash
bke build rpm --source ./rpm
```

```go
func rpmBuildPackage(source string, registry string) {
    // 1. 准备工作空间
    prepareWorkspace()
    
    // 2. 复制源目录到 packages
    utils.CopyDir(source, packages)
    
    // 3. 为所有系统/架构构建 rpm 仓库
    rpmBuildAllArchitectures(registry)
    
    // 4. 打包为 rpm.tar.gz.1
    compressAndCleanupRpm(path.Join(pwd, "rpm.tar.gz.1"), "Build rpm success.")
}
```
#### 场景 3: 向已有包添加新 RPM
```bash
bke build rpm --source rpm.tar.gz --add centos/8/amd64 --package ./docker-ce
```

```go
func rpmPackageAddOne(source string, registry string, add string, pack string) {
    // 1. 解压已有的 rpm.tar.gz
    utils.UnTar(source, packages)
    
    // 2. 复制新包到目标目录
    addList := strings.Split(adds[add], "/")  // ["CentOS", "8", "amd64"]
    utils.CopyDir(pack, path.Join(packages, addList[0], addList[1], addList[2]))
    
    // 3. 重新构建该系统的 rpm 仓库
    rmpBuild(registry, add, path.Join(packages, addList[0], addList[1], addList[2]))
    
    // 4. 重新打包
    compressAndCleanupRpm(path.Join(pwd, "rpm.tar.gz.1"), "The rpm package has been successfully built")
}
```
## 5. 构建实现详解
### 5.1 CentOS 7 构建
```go
func rpmCentos7Build(registry string, mnt string) error {
    return executeGenericRpmBuild(rpmBuildConfig{
        registry:      registry,
        mnt:           mnt,
        image:         "centos:7-amd64-build",      // 构建镜像
        containerName: "build-centos7-rpm",
        cmd:           "createrepo ./",              // 构建命令
        osInfo:        "centos/7/amd64",
        checkFile:     "repodata",                   // 校验文件
    })
}
```
**构建流程**:
1. 拉取 `centos:7-amd64-build` 镜像
2. 清理旧的 `repodata` 目录
3. 启动容器执行 `createrepo ./`
4. 验证 `repodata` 目录是否存在
### 5.2 CentOS 8 构建
```go
func rpmCentos8Build(registry string, mnt string) error {
    // CentOS 8 需要额外的模块支持
    cmd := "createrepo ./ && repo2module -s stable . modules.yaml && " +
           "modifyrepo_c --mdtype=modules modules.yaml repodata/"
    
    return executeGenericRpmBuild(rpmBuildConfig{
        registry:      registry,
        mnt:           mnt,
        image:         "centos:8-amd64-build",
        containerName: "build-centos8-rpm",
        cmd:           cmd,
        osInfo:        "centos/8/amd64",
        checkFile:     "modules.yaml",  // 需要校验 modules.yaml
    })
}
```
**CentOS 8 特殊处理**:
```go
func cleanCentos8Modules(mnt string) error {
    // 清理旧的模块文件
    for _, f := range []string{"modules.yaml", "repodata", ".repodata"} {
        os.RemoveAll(path.Join(mnt, f))
    }
    // 清理子目录中的模块文件
    for _, entry := range entries {
        for _, f := range []string{"modules.yaml", "repodata"} {
            os.RemoveAll(path.Join(mnt, entry.Name(), f))
        }
    }
}
```
### 5.3 Ubuntu 22 构建
```go
func rpmUbuntu22Build(registry string, mnt string) error {
    image := registry + "/ubuntu:22-amd64-build"
    
    // 清理旧的 Packages.gz
    os.RemoveAll(path.Join(mnt, "Packages.gz"))
    os.RemoveAll(path.Join(mnt, "archives", "Packages.gz"))
    
    // 启动容器执行 dpkg-scanpackages
    runBuildContainer(image, mnt, "build-ubuntu22-rpm",
        "dpkg-scanpackages -m . /dev/null | gzip -9c > Packages.gz && cp Packages.gz ./archives")
    
    // 验证 Packages.gz 是否存在
    if !utils.Exists(path.Join(mnt, "Packages.gz")) {
        return errors.New("packages.gz not found")
    }
}
```
### 5.4 Kylin V10 构建
```go
func rpmKylinV10Build(registry string, mnt string) error {
    // Kylin V10 使用与 CentOS 7 相同的构建方式
    return executeGenericRpmBuild(rpmBuildConfig{
        registry:      registry,
        mnt:           mnt,
        image:         "centos:7-amd64-build",
        containerName: "build-kylin10-rpm",
        cmd:           "createrepo ./",
        osInfo:        "kylin/v10/amd64",
        checkFile:     "repodata",
    })
}
```
## 6. 通用构建流程
```go
func executeGenericRpmBuild(cfg rpmBuildConfig) error {
    // 1. 检查目录是否为空
    if utils.DirectoryIsEmpty(cfg.mnt) {
        return nil
    }

    // 2. 确保构建镜像存在
    image, err := ensureRpmBuildImage(cfg.registry, cfg.image)
    if err != nil {
        return err
    }

    // 3. 清理旧的仓库元数据
    if err := cleanRepodata(cfg.mnt); err != nil {
        return err
    }

    // 4. 执行构建容器
    if err := executeRpmBuildContainer(image, cfg.mnt, cfg.containerName, cfg.cmd); err != nil {
        return err
    }

    // 5. 验证构建结果
    return verifyRpmBuildResult(cfg.mnt, cfg.osInfo, cfg.checkFile)
}
```
### 6.1 容器执行流程
```go
func executeRpmBuildContainer(image, mnt, containerName, cmd string) error {
    // 1. 移除同名容器（如果存在）
    global.Docker.ContainerRemove(containerName)
    
    // 2. 启动构建容器
    runBuildContainer(image, mnt, containerName, cmd)
    
    // 3. 等待容器执行完成
    defer global.Docker.ContainerRemove(containerName)
    waitForContainerComplete(containerName)
}
```
### 6.2 容器运行配置
```go
func runBuildContainer(image, mnt, containerName, cmd string) error {
    return global.Docker.Run(
        &container.Config{
            Image:      image,
            WorkingDir: "/opt/mnt",
            Cmd: strslice.StrSlice{"sh", "-c", cmd},
        },
        &container.HostConfig{
            Mounts: []mount.Mount{
                {
                    Type:   mount.TypeBind,
                    Source: mnt,          // 宿主机目录
                    Target: "/opt/mnt",   // 容器内目录
                },
            },
        }, nil, nil, containerName)
}
```
## 7. 输出目录结构
### 7.1 标准目录结构
```
rpm/
├── CentOS/
│   ├── 7/
│   │   ├── amd64/
│   │   │   ├── docker-ce/
│   │   │   │   ├── docker-ce-20.10.7-3.el7.x86_64.rpm
│   │   │   │   ├── docker-ce-cli-20.10.7-3.el7.x86_64.rpm
│   │   │   │   └── ...
│   │   │   └── repodata/
│   │   │       ├── repomd.xml
│   │   │       └── ...
│   │   └── arm64/
│   │       └── ...
│   └── 8/
│       ├── amd64/
│       │   ├── docker-ce/
│       │   ├── repodata/
│       │   └── modules.yaml    # CentOS 8 特有
│       └── arm64/
│           └── ...
├── Ubuntu/
│   └── 22/
│       ├── amd64/
│       │   ├── docker-ce/
│       │   │   ├── containerd.io_1.4.11-1_amd64.deb
│       │   │   └── ...
│       │   ├── Packages.gz     # APT 仓库索引
│       │   └── archives/
│       │       └── Packages.gz
│       └── arm64/
│           └── ...
├── Kylin/
│   └── V10/
│       ├── amd64/
│       │   ├── docker-ce/
│       │   └── repodata/
│       └── arm64/
│           └── ...
└── files/                      # 其他文件
```
### 7.2 无参数时输出
```bash
$ bke build rpm

rpm
├── CentOS
│   ├── 7
│   │   ├── amd64
│   │   └── arm64
│   └── 8
│       ├── amd64
│       └── arm64
├── files
├── Kylin
│   └── V10
│       ├── amd64
│       └── arm64
└── Ubuntu
    └── 22
        ├── amd64
        └── arm64
```
## 8. 构建镜像说明
| 镜像 | 用途 | 包含工具 |
|------|------|----------|
| `centos:7-amd64-build` | CentOS 7 / Kylin V10 | createrepo |
| `centos:8-amd64-build` | CentOS 8 | createrepo, repo2module, modifyrepo_c |
| `ubuntu:22-amd64-build` | Ubuntu 22 | dpkg-scanpackages, gzip |
## 9. 使用示例
### 9.1 初始化新仓库
```bash
# 创建目录结构
mkdir -p rpm/CentOS/8/amd64/docker-ce

# 放入 RPM 包
cp *.rpm rpm/CentOS/8/amd64/docker-ce/

# 构建仓库
bke build rpm --source rpm
```
### 9.2 添加新包到已有仓库
```bash
# 准备新包目录
mkdir -p new-packages/docker-ce
cp docker-ce-*.rpm new-packages/docker-ce/

# 添加到已有 rpm.tar.gz
bke build rpm --source rpm.tar.gz --add centos/8/amd64 --package new-packages/docker-ce
```
### 9.3 仅构建指定系统
```bash
# 仅构建 Ubuntu 22 的 DEB 仓库
mkdir -p debs/docker-ce
cp *.deb debs/docker-ce/

bke build rpm --add ubuntu/22/amd64 --package debs/docker-ce
```
### 9.4 自定义镜像仓库
```bash
bke build rpm --source rpm --registry cr.openfuyao.cn/openfuyao
```
## 10. 架构图
```
┌───────────────────────────────────────────────────────────────────────────┐
│                         bke build rpm 架构                                │
├───────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  ┌────────────────────────────────────────────────────────────────────┐   │
│  │                        参数解析与校验                              │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │   │
│  │  │ 验证 add     │  │ 验证 package │  │ 检查 Docker  │              │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘              │   │
│  └────────────────────────────────────────────────────────────────────┘   │
│                                    │                                      │
│                ┌───────────────────┼───────────────────┐                  │
│                ▼                   ▼                   ▼                  │
│  ┌─────────────────────┐ ┌─────────────────────┐ ┌─────────────────────┐  │
│  │   场景 1: 新建      │ │   场景 2: 全量构建  │ │   场景 3: 增量添加  │  │
│  │   --add + --package │ │   --source          │ │   --source + --add  │  │
│  │                     │ │                     │ │   + --package       │  │
│  │  ┌───────────────┐  │ │  ┌───────────────┐  │ │  ┌───────────────┐  │  │
│  │  │ rmpBuild()    │  │ │  │ prepareWork   │  │ │  │ 解压 tar.gz   │  │  │
│  │  │ 构建单个系统  │  │ │  │ space()       │  │ │  └───────────────┘  │  │
│  │  └───────────────┘  │ │  └───────────────┘  │ │  ┌───────────────┐  │  │
│  │                     │ │  ┌───────────────┐  │ │  │ 复制新包      │  │  │
│  │                     │ │  │ CopyDir()     │  │ │  └───────────────┘  │  │
│  │                     │ │  └───────────────┘  │ │  ┌───────────────┐  │  │
│  │                     │ │  ┌───────────────┐  │ │  │ rmpBuild()    │  │  │
│  │                     │ │  │ 构建所有系统   │  │   └───────────────┘  │  │
│  │                     │ │  └───────────────┘  │ │                     │  │
│  └─────────────────────┘ └─────────────────────┘ └─────────────────────┘  │
│                │                   │                   │                  │
│                └───────────────────┼───────────────────┘                  │
│                                    ▼                                      │
│  ┌────────────────────────────────────────────────────────────────────┐   │
│  │                        Docker 容器构建                             │   │
│  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │   │
│  │  │ CentOS 7     │  │ CentOS 8     │  │ Ubuntu 22    │              │   │
│  │  │ createrepo   │  │ createrepo + │  │ dpkg-scan    │              │   │
│  │  │              │  │ repo2module  │  │ packages     │              │   │
│  │  └──────────────┘  └──────────────┘  └──────────────┘              │   │
│  │  ┌──────────────┐                                                  │   │
│  │  │ Kylin V10    │                                                  │   │
│  │  │ createrepo   │                                                  │   │
│  │  └──────────────┘                                                  │   │
│  └────────────────────────────────────────────────────────────────────┘   │
│                                    │                                        │
│                                    ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │                           输出产物                                   │   │
│  │                                                                       │   │
│  │   rpm.tar.gz.1 (或直接生成 repodata/Packages.gz)                     │   │
│  │                                                                       │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```
## 11. 与 `bke build` 的关系
`bke build rpm` 生成的 `rpm.tar.gz` 会被 `bke build` 命令使用：
```yaml
# bke.yaml 配置文件
files:
  - address: https://openfuyao.obs.cn-north-4.myhuaweicloud.com/rpm/releases/download/v0.0.1/
    files:
      - rpm.tar.gz    # 这里的 rpm.tar.gz 就是由 bke build rpm 生成的
```
在 `bke build` 执行过程中：
```go
func buildRpms(cfg *BuildConfig, stopChan <-chan struct{}) error {
    // ...
    // 解压 rpm.tar.gz
    if err = buildFileRpm(); err != nil {
        return err
    }
    // 同步 RPM 包到目标目录
    for _, rpm := range cfg.Rpms {
        syncPackage(url, rpm.System, rpm.SystemVersion, ...)
    }
    // ...
}
```

# `bke init` 设计思路与详细设计
## 1. 设计思路
### 1.1 核心目标
`bke init` 命令的核心目标是**初始化引导节点**，将一台普通服务器转变为 Kubernetes 集群的管理节点，为后续的集群创建和节点加入做好准备。
### 1.2 设计原则
```
┌─────────────────────────────────────────────────────────────────────────┐
│                           设计原则                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. 离线优先                                                             │
│     • 支持完全离线环境安装                                                │
│     • 所有依赖可预先打包到 bke.tar.gz                                     │
│     • 支持在线模式补充缺失资源                                            │
│                                                                         │
│  2. 声明式配置                                                           │
│     • 通过 YAML 文件定义集群配置                                          │
│     • 配置与执行分离                                                      │
│     • 支持配置模板生成                                                    │
│                                                                         │
│  3. 幂等性                                                               │
│     • 重复执行不会产生副作用                                              │
│     • 已存在的服务跳过安装                                                │
│     • 支持增量更新                                                        │
│                                                                         │
│  4. 可扩展性                                                             │
│     • 支持外部镜像仓库                                                    │
│     • 支持外部软件源                                                      │
│     • 支持自定义 Console 安装                                            │
│                                                                         │
│  5. 安全性                                                               │
│     • 支持 TLS 证书认证                                                  │
│     • 支持用户名密码认证                                                  │
│     • 自动生成并管理证书                                                  │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```
### 1.3 两种安装模式
| 模式 | 触发条件 | 数据来源 | 适用场景 |
|------|----------|----------|----------|
| **离线模式** | `--otherRepo` 和 `--onlineImage` 都为空 | 本地 `bke.tar.gz` | 内网隔离环境 |
| **在线模式** | 指定 `--otherRepo` 或 `--onlineImage` | 远程镜像仓库 | 有网络环境 |
## 2. 命令定义
**入口文件**: [cmd/init.go](file:///D:/code/github/bkeadm/cmd/init.go)
```go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Initialize the boot node",
    Long:  `Initialize the boot node, including node check, warehouse start, cluster installation, and so on`,
    Example: `
# 初始化引导节点
bke init 

# 启用 Console 安装
bke init --installConsole=true

# 禁用 Console 安装
bke init --installConsole=false

# 使用集群配置文件初始化
bke init --file bkecluster.yaml

# 在线模式：从指定镜像下载文件
bke init --otherRepo cr.openfuyao.cn/openfuyao/bke-online-installed:latest

# 使用外部镜像仓库和软件源
bke init --otherRepo cr.openfuyao.cn/openfuyao --otherSource http://192.168.1.120:40080
`,
}
```
### 参数说明
| 参数 | 默认值 | 说明 |
|------|--------|------|
| `--file` | - | 集群配置文件路径 |
| `--hostIP` | 自动获取 | 引导节点 IP 地址 |
| `--domain` | `bke.registry.io` | 镜像仓库域名 |
| `--kubernetesPort` | `36443` | Kubernetes API 端口 |
| `--imageRepoPort` | `443` | 镜像仓库端口 |
| `--yumRepoPort` | `40080` | YUM 仓库端口 |
| `--chartRepoPort` | `38080` | Chart 仓库端口 |
| `--runtime` | `docker` | 容器运行时 |
| `--installConsole` | `true` | 是否安装 Console |
| `--enableNTP` | `true` | 是否启用 NTP 服务 |
| `--otherRepo` | - | 外部镜像仓库地址 |
| `--otherSource` | - | 外部软件源地址 |
| `--onlineImage` | - | 在线安装镜像地址 |
| `--imageFilePath` | - | 本地镜像文件路径 |
| `--imageRepoCAFile` | - | 镜像仓库 CA 证书 |
| `--imageRepoUsername` | - | 镜像仓库用户名 |
| `--imageRepoPassword` | - | 镜像仓库密码 |
## 3. 整体架构
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              bke init 整体架构                                           │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                              用户输入层                                           │   │
│  │                                                                                   │   │
│  │  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐       │   │
│  │  │ bke.tar.gz  │    │bkecluster.  │    │ --otherRepo │    │ --onlineImage│      │   │
│  │  │ (离线包)    │    │    yaml     │    │ (外部仓库)  │    │ (在线镜像)  │       │   │
│  │  └──────┬──────┘    └──────┬──────┘    └──────┬──────┘    └──────┬──────┘       │   │
│  └─────────┼──────────────────┼──────────────────┼──────────────────┼───────────────┘   │
│            │                  │                  │                  │                    │
│            └──────────────────┼──────────────────┼──────────────────┘                    │
│                               ▼                  ▼                                       │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                           参数解析与校验                                          │   │
│  │                                                                                   │   │
│  │  ┌───────────────────────────────────────────────────────────────────────────┐  │   │
│  │  │                      ParseOnlineConfig()                                   │  │   │
│  │  │                                                                            │  │   │
│  │  │   解析模式:                                                                │  │   │
│  │  │   • 离线模式: otherRepo="" && onlineImage=""                              │  │   │
│  │  │   • 在线模式: otherRepo!="" || onlineImage!=""                            │  │   │
│  │  │                                                                            │  │   │
│  │  │   解析内容:                                                                │  │   │
│  │  │   • Repo: 镜像仓库地址                                                    │  │   │
│  │  │   • RepoIP: 仓库 IP 地址                                                  │  │   │
│  │  │   • Source: 软件源地址                                                    │  │   │
│  │  │   • ChartRepo: Chart 仓库地址                                             │  │   │
│  │  └───────────────────────────────────────────────────────────────────────────┘  │   │
│  │                                                                                   │   │
│  │  ┌───────────────────────────────────────────────────────────────────────────┐  │   │
│  │  │                         Validate()                                        │  │   │
│  │  │                                                                            │  │   │
│  │  │   校验内容:                                                                │  │   │
│  │  │   • 磁盘空间检查 (≥20GB 或 ≥3GB 已有数据)                                  │  │   │
│  │  │   • 端口占用检查 (36443, 443, 40080, 38080, 2049)                         │  │   │
│  │  │   • 过滤已存在容器的端口                                                   │  │   │
│  │  └───────────────────────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                           │                                             │
│                                           ▼                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                           初始化执行流程                                          │   │
│  │                                                                                   │   │
│  │  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐     ┌─────────┐          │   │
│  │  │Step 1   │──▶│Step 2   │──▶│Step 3   │──▶│Step 4   │──▶│Step 5   │          │   │
│  │  │节点信息  │   │环境校验 │   │时区设置  │   │环境准备  │     │容器服务 │          │   │
│  │  │收集     │   │         │   │         │   │         │      │         │          │   │
│  │  └─────────┘   └─────────┘   └─────────┘   └─────────┘      └─────────┘          │   │
│  │       │                                                        │               │   │
│  │       ▼                                                        ▼               │   │
│  │  ┌─────────┐    ┌─────────┐   ┌─────────┐     ┌─────────┐   ┌─────────┐          │   │
│  │  │Step 6   │──▶ │Step 7   │──▶│Step 8   │──▶│Step 9   │──▶│Step 10  │          │   │
│  │  │仓库服务  │    │Cluster  │──▶│Console  │──▶│生成配置 │──▶│部署集群 │          │   │
│  │  │         │    │API      │   │(可选)   │     │文件     │   │         │          │   │
│  │  └─────────┘    └─────────┘   └─────────┘     └─────────┘   └─────────┘          │   │
│  │                                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                           │                                             │
│                                           ▼                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                              输出产物                                             │   │
│  │                                                                                   │   │
│  │  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐                 │   │
│  │  │ 运行中的服务     │  │ 配置文件        │  │ 管理集群        │                 │   │
│  │  │                 │  │                 │  │                 │                 │   │
│  │  │ • K3s (K8s)     │  │ • bkecluster.yaml│ │ • Cluster API   │                 │   │
│  │  │ • Registry      │  │ • bkenodes.yaml │  │ • Cert Manager  │                 │   │
│  │  │ • YUM Repo      │  │ • kubeconfig    │  │ • BKE Provider  │                 │   │
│  │  │ • Chart Repo    │  │                 │  │ • Console (可选)│                 │   │
│  │  │ • NFS (可选)    │  │                 │  │                 │                 │   │
│  │  └─────────────────┘  └─────────────────┘  └─────────────────┘                 │   │
│  │                                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
## 4. 详细执行流程
### 4.1 主初始化流程
**实现文件**: [pkg/initialize/initialize.go](file:///D:/code/github/bkeadm/pkg/initialize/initialize.go#L95-L130)
```go
func (op *Options) Initialize() {
    log.BKEFormat(log.INFO, "BKE initialize ...")
    
    // Step 1: 节点信息收集
    op.nodeInfo()
    
    // Step 2: 环境校验
    err := op.Validate()
    
    // Step 3: 时区和 NTP 设置
    err = op.setTimezone()
    
    // Step 4: 环境准备 (配置源、hosts、证书)
    err = op.prepareEnvironment()
    
    // Step 5: 容器服务启动
    err = op.ensureContainerServer()
    
    // Step 6: 仓库服务启动
    err = op.ensureRepository()
    
    // Step 7: Cluster API 安装
    err = op.ensureClusterAPI()
    
    // Step 8: Console 安装 (可选)
    if op.InstallConsole {
        err = op.ensureConsoleAll()
    }
    
    // Step 9: 生成配置文件
    op.generateClusterConfig()
    
    // Step 10: 部署集群
    op.deployCluster()
}
```
### 4.2 各步骤详解
#### Step 1: 节点信息收集
```go
func (op *Options) nodeInfo() {
    h, _ := host.Info()
    c, _ := cpu.Counts(false)
    v, _ := mem.VirtualMemory()

    log.BKEFormat(log.INFO, fmt.Sprintf("HOSTNAME: %s", h.Hostname))
    log.BKEFormat(log.INFO, fmt.Sprintf("PLATFORM: %s", h.Platform))
    log.BKEFormat(log.INFO, fmt.Sprintf("Version:  %s", h.PlatformVersion))
    log.BKEFormat(log.INFO, fmt.Sprintf("KERNEL:   %s", h.KernelVersion))
    log.BKEFormat(log.INFO, fmt.Sprintf("GOOS:     %s", runtime.GOOS))
    log.BKEFormat(log.INFO, fmt.Sprintf("ARCH:     %s", runtime.GOARCH))
    log.BKEFormat(log.INFO, fmt.Sprintf("CPU:      %d", c))
    log.BKEFormat(log.INFO, fmt.Sprintf("MEMORY:   %dG", v.Total/1024/1024/1024+1))
}
```
#### Step 2: 环境校验
```go
func (op *Options) Validate() error {
    // 1. 解析在线配置
    oc, err = repository.ParseOnlineConfig(
        op.Domain, op.OnlineImage, op.OtherRepo, op.OtherSource, op.OtherChart)
    
    // 2. 磁盘空间检查
    if err = op.validateDiskSpace(); err != nil { return err }
    
    // 3. 端口占用检查
    if err = op.validatePorts(); err != nil { return err }
    
    // 4. 设置全局配置
    op.setGlobalCustomExtra()
    return nil
}
```
**磁盘空间要求**:
- 新安装: ≥ 20GB
- 已有数据: ≥ 3GB

**端口检查**:
| 端口 | 服务 |
|------|------|
| 36443 | Kubernetes API |
| 443 | 镜像仓库 |
| 40080 | YUM 仓库 |
| 38080 | Chart 仓库 |
| 2049 | NFS |
#### Step 3: 时区和 NTP 设置
```go
func (op *Options) setTimezone() error {
    // 设置时区
    err := timezone.SetTimeZone()
    
    // NTP 服务 (可选)
    if !op.EnableNTP {
        return nil
    }
    
    // 设置 NTP 服务器
    newNTPServer, err := timezone.NTPServer(op.NtpServer, op.HostIP, len(oc.Repo) > 0)
    op.NtpServer = newNTPServer
    return nil
}
```
#### Step 4: 环境准备
```go
func (op *Options) prepareEnvironment() error {
    // 1. 配置本地软件源
    op.configLocalSource()
    
    // 2. 设置 hosts 文件
    hostIP, domain := op.resolveHostIPAndDomain()
    syscompat.SetHosts(hostIP, domain)
    
    // 3. 配置私有仓库证书
    clientAuthConfig := op.buildClientAuthConfig()
    op.configurePrivateRegistry(clientAuthConfig)
    
    // 4. 初始化仓库
    return op.initRepositories(clientAuthConfig)
}
```
**配置本地源**:
```go
func (op *Options) configLocalSource() {
    // 离线模式: 使用本地文件
    if op.OtherRepo == "" && op.OnlineImage == "" {
        baseurl := "file://" + path.Join(global.Workspace, utils.SourceDataDirectory)
        if len(oc.Source) > 0 {
            baseurl = oc.Source  // 在线模式: 使用外部源
        }
        configsource.SetSource(baseurl)
    }
}
```
#### Step 5: 容器服务启动
```go
func (op *Options) ensureContainerServer() error {
    // 启动 Docker 或 Containerd
    return infrastructure.EnsureContainerServer(op.Runtime, op.RuntimeStorage)
}
```
#### Step 6: 仓库服务启动
```go
func (op *Options) ensureRepository() error {
    // 1. 解压数据包
    PrepareRepositoryDependOn(op.ImageFilePath)
    
    // 2. 启动镜像仓库
    repository.ImageServer(op.ImageFilePath, op.ImageRepoPort, oc.Repo, oc.Image)
    
    // 3. 启动 YUM 仓库
    repository.YumServer(op.ImageFilePath, op.YumRepoPort, oc.Repo, oc.Image)
    
    // 4. 启动 Chart 仓库
    repository.ChartServer(op.ImageFilePath, op.ImageRepoPort, oc.ChartRepoPort, oc.Repo, oc.Image)
    
    // 5. 启动 NFS 服务 (可选)
    if op.ImageFilePath == "" {
        repository.NFSServer(op.ImageRepoPort, oc.Repo, oc.Image)
    }
    return nil
}
```
**仓库服务架构**:
```
┌─────────────────────────────────────────────────────────────────┐
│                      仓库服务架构                                │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Image Registry                         │   │
│  │                   (镜像仓库)                              │   │
│  │                                                           │   │
│  │   端口: 443                                               │   │
│  │   镜像: registry:2.8.1                                    │   │
│  │   数据: /bke/mount/image_registry/                       │   │
│  │   功能: 存储 Kubernetes 集群所需的所有容器镜像             │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    YUM Registry                          │   │
│  │                   (软件包仓库)                            │   │
│  │                                                           │   │
│  │   端口: 40080                                             │   │
│  │   镜像: nginx:1.23.0-alpine                               │   │
│  │   数据: /bke/mount/source_registry/files/                │   │
│  │   功能: 提供 RPM/DEB 软件包                               │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   Chart Registry                         │   │
│  │                   (Chart 仓库)                            │   │
│  │                                                           │   │
│  │   端口: 38080                                             │   │
│  │   镜像: helm/chartmuseum:v0.16.2                          │   │
│  │   数据: /bke/mount/charts/                                │   │
│  │   功能: 存储 Helm Charts                                  │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                    NFS Server                            │   │
│  │                   (文件共享)                              │   │
│  │                                                           │   │
│  │   端口: 2049                                              │   │
│  │   镜像: openebs/nfs-server-alpine:0.9.0                   │   │
│  │   数据: /bke/mount/nfsshare/                              │   │
│  │   功能: 提供 NFS 共享存储                                 │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```
#### Step 7: Cluster API 安装
```go
func (op *Options) ensureClusterAPI() error {
    // 1. 启动 K3s (轻量级 Kubernetes)
    infrastructure.StartLocalKubernetes(k3s.Config{
        OnlineImage:    oc.Image,
        OtherRepo:      oc.Repo,
        OtherRepoIP:    oc.RepoIP,
        HostIP:         op.HostIP,
        ImageRepo:      op.Domain,
        ImageRepoPort:  op.ImageRepoPort,
        KubernetesPort: op.KubernetesPort,
    }, op.ImageFilePath)
    
    // 2. 生成版本 ConfigMap
    if op.OFVersion != "" {
        op.generateDeployCM()
    }
    
    // 3. 配置 Containerd
    containerd.ApplyContainerdCfg(fmt.Sprintf("%s:%s", op.Domain, op.ImageRepoPort))
    
    // 4. 部署 Cluster API 组件
    clusterapi.DeployClusterAPI(repo, manifestsVersion, providerVersion)
    
    // 5. 部署 BKE Agent
    bkeagent.DeployBKEAgent(repo, agentVersion)
}
```
**Cluster API 组件**:
```
┌─────────────────────────────────────────────────────────────────┐
│                    Cluster API 组件架构                          │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   cert-manager                           │   │
│  │                   (证书管理)                              │   │
│  │                                                           │   │
│  │   Namespace: cert-manager                                 │   │
│  │   功能: 为 Webhook 自动生成证书                           │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                   cluster-api                            │   │
│  │                   (核心组件)                              │   │
│  │                                                           │   │
│  │   Namespace: cluster-system                               │   │
│  │   功能: Cluster API 核心控制器                            │   │
│  │   • Cluster Controller                                    │   │
│  │   • Machine Controller                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │               cluster-api-provider-bke                   │   │
│  │                   (BKE Provider)                          │   │
│  │                                                           │   │
│  │   Namespace: cluster-system                               │   │
│  │   功能: BKE 集群 Provider                                 │   │
│  │   • BKECluster Controller                                 │   │
│  │   • BKENode Controller                                    │   │
│  │   • Command Controller                                    │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │                     bkeagent                             │   │
│  │                   (节点代理)                              │   │
│  │                                                           │   │
│  │   Namespace: bke-system                                   │   │
│  │   部署方式: DaemonSet                                     │   │
│  │   功能: 在每个节点执行命令                                │   │
│  └─────────────────────────────────────────────────────────┘   │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```
#### Step 8: Console 安装 (可选)
```go
func (op *Options) ensureConsoleAll() error {
    // 安装 BKE Console 组件
    return bkeconsole.InstallConsole(repo, consoleVersion)
}
```
**Console 组件**:
- OAuth Server: 认证服务
- OAuth Webhook: 认证 Webhook
- User Manager: 用户管理
- Console Service: 控制台后端
- Console Website: 控制台前端

#### Step 9: 生成配置文件
```go
func (op *Options) generateClusterConfig() {
    // 生成 bkecluster.yaml
    // 生成 bkenodes.yaml
    // 生成 kubeconfig
}
```
#### Step 10: 部署集群
```go
func (op *Options) deployCluster() {
    // 如果提供了集群配置文件，自动创建集群
    if op.File != "" {
        cluster.CreateCluster(op.File)
    }
}
```
## 5. 在线/离线模式对比
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                           在线模式 vs 离线模式                                            │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌─────────────────────────────────────┐    ┌─────────────────────────────────────┐    │
│  │           离线模式                   │    │           在线模式                   │    │
│  │                                     │    │                                     │    │
│  │  触发条件:                          │    │  触发条件:                          │    │
│  │  --otherRepo="" &&                  │    │  --otherRepo!="" ||                 │    │
│  │  --onlineImage=""                   │    │  --onlineImage!=""                  │    │
│  │                                     │    │                                     │    │
│  │  数据来源:                          │    │  数据来源:                          │    │
│  │  ┌─────────────────────────────┐   │    │  ┌─────────────────────────────┐   │    │
│  │  │ 本地 bke.tar.gz             │   │    │  │ 远程镜像仓库                 │   │    │
│  │  │                             │   │    │  │                             │   │    │
│  │  │ • /bke/volumes/image.tar.gz │   │    │  │ • cr.openfuyao.cn/...       │   │    │
│  │  │ • /bke/volumes/source.tar.gz│   │    │  │ • 外部 YUM 源               │   │    │
│  │  │ • /bke/manifests.yaml       │   │    │  │ • 外部 Chart 源             │   │    │
│  │  └─────────────────────────────┘   │    │  └─────────────────────────────┘   │    │
│  │                                     │    │                                     │    │
│  │  仓库服务:                          │    │  仓库服务:                          │    │
│  │  • 本地 Registry                   │    │  • 本地 Registry (代理外部)         │    │
│  │  • 本地 YUM Repo                   │    │  • 外部 YUM Repo                    │    │
│  │  • 本地 Chart Repo                 │    │  • 外部 Chart Repo                  │    │
│  │  • 本地 NFS                        │    │  • 无需 NFS                         │    │
│  │                                     │    │                                     │    │
│  │  适用场景:                          │    │  适用场景:                          │    │
│  │  • 内网隔离环境                    │    │  • 有网络环境                       │    │
│  │  • 安全要求高                      │    │  • 快速部署                         │    │
│  │  • 大规模部署                      │    │  • 测试环境                         │    │
│  │                                     │    │                                     │    │
│  └─────────────────────────────────────┘    └─────────────────────────────────────┘    │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
## 6. 目录结构
初始化完成后的目录结构：
```
/bke/
├── bke                          # bke 二进制文件
├── bkecluster.yaml              # 集群配置模板
├── bkenodes.yaml                # 节点配置模板
├── kubeconfig                   # kubeconfig 文件
├── manifests.yaml               # 安装包清单
├── mount/                       # 挂载数据目录
│   ├── image_registry/          # 镜像仓库数据
│   │   └── docker/registry/v2/
│   ├── source_registry/         # 软件源数据
│   │   └── files/
│   │       ├── CentOS/
│   │       ├── Ubuntu/
│   │       ├── Kylin/
│   │       ├── kubectl-*
│   │       ├── kubelet-*
│   │       └── charts.tar.gz
│   ├── charts/                  # Charts 数据
│   └── nfsshare/                # NFS 共享数据
├── volumes/                     # 原始数据包
│   ├── image.tar.gz             # 镜像数据包
│   └── source.tar.gz            # 源数据包
└── tmpl/                        # 模板文件
    ├── cert-manager.yaml
    ├── cluster-api.yaml
    └── cluster-api-bke.yaml
```
## 7. 典型使用场景
### 7.1 离线安装
```bash
# 1. 准备离线包
# 从构建服务器获取 bke.tar.gz

# 2. 解压离线包
tar -xzf bke.tar.gz -C /

# 3. 初始化引导节点
cd /bke
./bke init

# 4. 创建集群
./bke cluster create -f bkecluster.yaml
```
### 7.2 在线安装
```bash
# 1. 下载 bke 二进制
curl -LO https://openfuyao.cn/downloads/bke

# 2. 初始化 (指定外部仓库)
./bke init --otherRepo cr.openfuyao.cn/openfuyao \
           --otherSource https://openfuyao.cn/sources
# 3. 创建集群
./bke cluster create -f bkecluster.yaml
```
### 7.3 使用在线镜像安装
```bash
# 从在线镜像快速安装
./bke init --onlineImage cr.openfuyao.cn/openfuyao/bke-online-installed:latest
```
### 7.4 使用外部镜像仓库
```bash
# 使用企业内部镜像仓库
./bke init --otherRepo registry.company.com/bke \
           --imageRepoCAFile /path/to/ca.crt \
           --imageRepoUsername admin \
           --imageRepoPassword password
``
## 8. 状态流转图
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              bke init 状态流转                                           │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌───────────┐                                                                         │
│  │   开始    │                                                                         │
│  └─────┬─────┘                                                                         │
│        │                                                                               │
│        ▼                                                                               │
│  ┌───────────┐     ┌───────────┐                                                      │
│  │ 节点信息  │────▶│ 环境校验  │                                                      │
│  │   收集    │     │           │                                                      │
│  └───────────┘     └─────┬─────┘                                                      │
│                          │                                                            │
│                    ┌─────┴─────┐                                                      │
│                    │ 校验通过? │                                                      │
│                    └─────┬─────┘                                                      │
│                          │                                                            │
│              ┌───────────┼───────────┐                                                │
│              ▼                       ▼                                                │
│        ┌───────────┐           ┌───────────┐                                         │
│        │    是     │           │    否     │                                         │
│        └─────┬─────┘           └─────┬─────┘                                         │
│              │                       │                                                │
│              ▼                       ▼                                                │
│        ┌───────────┐           ┌───────────┐                                         │
│        │ 时区设置  │           │  报错退出  │                                         │
│        └─────┬─────┘           └───────────┘                                         │
│              │                                                                          │
│              ▼                                                                          │
│        ┌───────────┐                                                                  │
│        │ 环境准备  │                                                                  │
│        │ (源/hosts)│                                                                  │
│        └─────┬─────┘                                                                  │
│              │                                                                          │
│              ▼                                                                          │
│        ┌───────────┐     ┌───────────┐     ┌───────────┐                             │
│        │ 容器服务  │────▶│ 仓库服务  │────▶│Cluster API│                             │
│        │  启动     │     │   启动    │     │   安装    │                             │
│        └───────────┘     └───────────┘     └─────┬─────┘                             │
│                                                  │                                     │
│                                        ┌─────────┴─────────┐                           │
│                                        │ installConsole?   │                           │
│                                        └─────────┬─────────┘                           │
│                                                  │                                     │
│                              ┌───────────────────┼───────────────────┐                 │
│                              ▼                                       ▼                 │
│                        ┌───────────┐                           ┌───────────┐           │
│                        │    是     │                           │    否     │           │
│                        └─────┬─────┘                           └─────┬─────┘           │
│                              │                                       │                 │
│                              ▼                                       │                 │
│                        ┌───────────┐                                 │                 │
│                        │ Console   │                                 │                 │
│                        │   安装    │                                 │                 │
│                        └─────┬─────┘                                 │                 │
│                              │                                       │                 │
│                              └───────────────────┬───────────────────┘                 │
│                                                  │                                     │
│                                                  ▼                                     │
│                                            ┌───────────┐                              │
│                                            │ 生成配置  │                              │
│                                            │   文件    │                              │
│                                            └─────┬─────┘                              │
│                                                  │                                     │
│                                                  ▼                                     │
│                                            ┌───────────┐                              │
│                                            │ 部署集群  │                              │
│                                            │  (可选)   │                              │
│                                            └─────┬─────┘                              │
│                                                  │                                     │
│                                                  ▼                                     │
│                                            ┌───────────┐                              │
│                                            │   完成    │                              │
│                                            └───────────┘                              │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
这个设计文档展示了 `bke init` 的完整设计思路，包括离线/在线两种模式、详细的执行流程和各组件的职责。

# `bke init --file` 设计思路与详细设计
## 1. 设计思路
### 1.1 核心目标
`bke init --file bkecluster.yaml` 的核心目标是实现**声明式集群初始化**：
- 从集群配置文件中自动提取仓库信息
- 自动判断并设置在线/离线安装模式
- 初始化引导节点后自动部署集群
### 1.2 设计原则
```
┌─────────────────────────────────────────────────────────────────────────┐
│                           设计原则                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. 配置驱动                                                             │
│     • 从 bkecluster.yaml 自动提取 ImageRepo/HTTPRepo/ChartRepo          │
│     • 自动设置 --otherRepo/--otherSource/--otherChart 参数              │
│     • 减少用户手动输入参数的工作量                                        │
│                                                                         │
│  2. 智能模式切换                                                         │
│     • 根据配置文件中的仓库地址自动判断在线/离线模式                        │
│     • 本地 IP → 离线模式                                                 │
│     • 远程 IP/域名 → 在线模式                                            │
│                                                                         │
│  3. 一体化流程                                                           │
│     • 初始化引导节点 + 部署集群 一步完成                                  │
│     • 简化操作流程，减少人为错误                                          │
│                                                                         │
│  4. 参数优先级                                                           │
│     • 命令行参数 > 配置文件参数 > 默认值                                  │
│     • 用户显式指定的参数优先级最高                                        │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```
### 1.3 与普通 `bke init` 的区别
| 特性 | `bke init` | `bke init --file bkecluster.yaml` |
|------|------------|-----------------------------------|
| 参数来源 | 命令行参数 | 配置文件 + 命令行参数 |
| 仓库配置 | 手动指定 `--otherRepo` 等 | 自动从配置文件提取 |
| 集群部署 | 需要单独执行 `bke cluster create` | 初始化后自动部署 |
| 适用场景 | 快速初始化、测试环境 | 生产环境、自动化部署 |
## 2. 整体架构
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        bke init --file 整体架构                                          │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                              用户输入                                             │   │
│  │                                                                                   │   │
│  │           bkecluster.yaml                                                        │   │
│  │           ┌─────────────────────────────────────────────────────────────┐        │   │
│  │           │ apiVersion: bke.bocloud.com/v1beta1                        │        │   │
│  │           │ kind: BKECluster                                           │        │   │
│  │           │ metadata:                                                  │        │   │
│  │           │   name: my-cluster                                         │        │   │
│  │           │   namespace: bke-my-cluster                                │        │   │
│  │           │ spec:                                                      │        │   │
│  │           │   clusterConfig:                                           │        │   │
│  │           │     cluster:                                               │        │   │
│  │           │       imageRepo:        ← 提取镜像仓库信息                  │        │   │
│  │           │         domain: cr.openfuyao.cn                            │        │   │
│  │           │         ip: 192.168.1.100                                  │        │   │
│  │           │         port: "443"                                        │        │   │
│  │           │         prefix: openfuyao                                  │        │   │
│  │           │       httpRepo:        ← 提取软件源信息                    │        │   │
│  │           │         domain: openfuyao.cn                               │        │   │
│  │           │         ip: 192.168.1.100                                  │        │   │
│  │           │         port: "40080"                                      │        │   │
│  │           │       chartRepo:       ← 提取 Chart 仓库信息               │        │   │
│  │           │         domain: charts.openfuyao.cn                        │        │   │
│  │           │         port: "443"                                        │        │   │
│  │           │         prefix: charts                                     │        │   │
│  │           │       ntpServer: ntp.openfuyao.cn                          │        │   │
│  │           └─────────────────────────────────────────────────────────────┘        │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                           │                                             │
│                                           ▼                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                        PreRunE: 配置解析与参数提取                                │   │
│  │                                                                                   │   │
│  │  ┌───────────────────────────────────────────────────────────────────────────┐  │   │
│  │  │                    解析 bkecluster.yaml                                    │  │   │
│  │  │                                                                            │  │   │
│  │  │   1. loadBKEClusterFromFile(file)                                         │  │   │
│  │  │   2. 提取 ImageRepo 配置:                                                 │  │   │
│  │  │      • 如果 IP != 本机 IP → 设置 otherRepo                                │  │   │
│  │  │      • 格式: IP:Port/Prefix 或 Domain:Port/Prefix                         │  │   │
│  │  │   3. 提取 HTTPRepo 配置:                                                  │  │   │
│  │  │      • 如果 IP != 本机 IP → 设置 otherSource                              │  │   │
│  │  │      • 格式: http://IP:Port 或 http://Domain:Port                         │  │   │
│  │  │   4. 提取 ChartRepo 配置:                                                 │  │   │
│  │  │      • 设置 otherChart                                                    │  │   │
│  │  │      • 格式: oci://Domain/Prefix 或 IP:Port/Prefix                        │  │   │
│  │  │   5. 提取 NTPServer 配置                                                  │  │   │
│  │  └───────────────────────────────────────────────────────────────────────────┘  │   │
│  │                                                                                   │   │
│  │  ┌───────────────────────────────────────────────────────────────────────────┐  │   │
│  │  │                    模式判断逻辑                                            │  │   │
│  │  │                                                                            │  │   │
│  │  │   if imageRepo.IP == 本机IP:                                              │  │   │
│  │  │       → 离线模式 (otherRepo = 空)                                         │  │   │
│  │  │   else:                                                                    │  │   │
│  │  │       → 在线模式 (otherRepo = 提取的仓库地址)                              │  │   │
│  │  └───────────────────────────────────────────────────────────────────────────┘  │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                           │                                             │
│                                           ▼                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                        Initialize(): 标准初始化流程                               │   │
│  │                                                                                   │   │
│  │   Step 1: 节点信息收集                                                            │   │
│  │   Step 2: 环境校验                                                                │   │
│  │   Step 3: 时区设置                                                                │   │
│  │   Step 4: 环境准备                                                                │   │
│  │   Step 5: 容器服务启动                                                            │   │
│  │   Step 6: 仓库服务启动                                                            │   │
│  │   Step 7: Cluster API 安装                                                        │   │
│  │   Step 8: Console 安装 (可选)                                                     │   │
│  │   Step 9: 生成配置文件                                                            │   │
│  │   Step 10: deployCluster() ← 自动部署集群                                         │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                           │                                             │
│                                           ▼                                             │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                        deployCluster(): 集群部署                                  │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │                    cluster.Options.Cluster()                             │   │   │
│  │   │                                                                           │   │   │
│  │   │   1. NewClusterResourcesFromFiles(file, nodesFile)                       │   │   │
│  │   │   2. 创建 Namespace                                                       │   │   │
│  │   │   3. 安装 BKENodes YAML                                                   │   │   │
│  │   │   4. 安装 BKECluster YAML                                                 │   │   │
│  │   │   5. 等待集群启动                                                          │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
## 3. 详细代码分析
### 3.1 参数解析入口
**文件**: [cmd/init.go](file:///D:/code/github/bkeadm/cmd/init.go#L128-L196)
```go
PreRunE: func(cmd *cobra.Command, args []string) error {
    // 自动获取本机 IP
    if initOption.HostIP == "" {
        ip, _ := utils.GetOutBoundIP()
        initOption.HostIP = ip
    }
    
    // 如果指定了 --file 参数
    if initOption.File != "" {
        // 1. 解析 BKECluster 配置文件
        bkeCluster, err := cluster.NewBKEClusterFromFile(initOption.File)
        if err != nil {
            return err
        }
        
        // 2. 提取镜像仓库配置
        imageRepo := bkeCluster.Spec.ClusterConfig.Cluster.ImageRepo
        imgPort := ":443"
        if len(imageRepo.Port) > 0 {
            imgPort = ":" + imageRepo.Port
        }
        
        // 优先使用 IP 地址判断
        if len(imageRepo.Ip) > 0 {
            // 如果 IP 和本机 IP 不同，说明使用外部仓库
            if imageRepo.Ip != initOption.HostIP {
                if imageRepo.Domain == "" || imageRepo.Domain == configinit.DefaultImageRepo {
                    initOption.OtherRepo = imageRepo.Ip + imgPort + "/" + imageRepo.Prefix
                } else {
                    initOption.OtherRepo = imageRepo.Domain + imgPort + "/" + imageRepo.Prefix
                }
            }
        } else if len(imageRepo.Domain) > 0 {
            _, err := utils.LoopIP(imageRepo.Domain)
            if err != nil {
                return err
            }
            initOption.OtherRepo = imageRepo.Domain + imgPort + "/" + imageRepo.Prefix
        }
        
        // 3. 提取 HTTP 仓库配置
        httpRepo := bkeCluster.Spec.ClusterConfig.Cluster.HTTPRepo
        // ... 类似逻辑设置 otherSource
        
        // 4. 提取 Chart 仓库配置
        chartRepo := bkeCluster.Spec.ClusterConfig.Cluster.ChartRepo
        // ... 类似逻辑设置 otherChart
        
        // 5. 提取 NTP 服务器配置
        if len(bkeCluster.Spec.ClusterConfig.Cluster.NTPServer) > 0 {
            initOption.NtpServer = bkeCluster.Spec.ClusterConfig.Cluster.NTPServer
        }
    }
    // ...
}
```
### 3.2 模式判断逻辑
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              模式判断流程                                                │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                         ImageRepo.IP 判断                                        │   │
│  │                                                                                   │   │
│  │   imageRepo.Ip != "" ?                                                           │   │
│  │   ├── YES: imageRepo.Ip == 本机IP ?                                              │   │
│  │   │   ├── YES: 离线模式 (不设置 otherRepo)                                        │   │
│  │   │   │           使用本地镜像仓库                                                │   │
│  │   │   │                                                                           │   │
│  │   │   └── NO:  在线模式                                                           │   │
│  │   │               otherRepo = IP:Port/Prefix                                      │   │
│  │   │               或 Domain:Port/Prefix                                           │   │
│  │   │                                                                               │   │
│  │   └── NO:  imageRepo.Domain != "" ?                                              │   │
│  │       ├── YES: 在线模式                                                           │   │
│  │       │           otherRepo = Domain:Port/Prefix                                  │   │
│  │       │           (需要 DNS 解析)                                                 │   │
│  │       │                                                                           │   │
│  │       └── NO:  离线模式                                                           │   │
│  │                   使用默认本地仓库                                                 │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                         HTTPRepo 判断                                            │   │
│  │                                                                                   │   │
│  │   httpRepo.Ip != "" ?                                                            │   │
│  │   ├── YES: httpRepo.Ip == 本机IP ?                                               │   │
│  │   │   ├── YES: 不设置 otherSource (使用本地 YUM 源)                               │   │
│  │   │   └── NO:  otherSource = http://IP:Port                                      │   │
│  │   │                                                                               │   │
│  │   └── NO:  httpRepo.Domain != "" ?                                               │   │
│  │       ├── YES: otherSource = http://Domain:Port                                  │   │
│  │       └── NO:  不设置 otherSource                                                │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
### 3.3 集群部署逻辑
**文件**: [pkg/initialize/initialize.go](file:///D:/code/github/bkeadm/pkg/initialize/initialize.go#L1188-L1198)
```go
func (op *Options) deployCluster() {
    // 如果没有指定 --file，跳过部署
    if op.File == "" {
        return
    }
    
    log.BKEFormat(log.INFO, "Starting to deploy the cluster...")
    
    // 创建集群部署选项
    c := cluster.Options{
        File:      op.File,
        NtpServer: op.NtpServer,
        // 注意: NodesFile 未设置
    }
    
    // 执行集群部署
    c.Cluster()
}
```
**文件**: [pkg/cluster/deploy.go](file:///D:/code/github/bkeadm/pkg/cluster/deploy.go#L31-L80)
```go
func (op *Options) Cluster() {
    // 从文件加载 BKECluster 和 BKENode 资源
    resources, err := NewClusterResourcesFromFiles(op.File, op.NodesFile)
    if err != nil {
        log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to load the configuration file. %v", err))
        return
    }

    bkeCluster := resources.BKECluster
    bkeNodes := resources.BKENodes

    // 覆盖 NTP 服务器配置
    if len(op.NtpServer) > 0 {
        bkeCluster.Spec.ClusterConfig.Cluster.NTPServer = op.NtpServer
    }

    // 规范化 namespace
    if !strings.HasPrefix(bkeCluster.Namespace, "bke") {
        bkeCluster.Namespace = "bke-" + bkeCluster.Namespace
    }

    // 写入 BKECluster YAML
    bkefile, err := marshalAndWriteClusterYAML(bkeCluster)
    
    // 写入 BKENodes YAML
    nodeFiles, err := marshalAndWriteNodeYAMLs(bkeCluster.Namespace, bkeNodes)

    // 创建 Namespace
    namespace := corev1.Namespace{...}
    err = global.K8s.CreateNamespace(&namespace)

    // 先安装 BKENodes (webhook 需要节点来设置默认 ControlPlaneEndpoint)
    for _, nodeFile := range nodeFiles {
        err = global.K8s.InstallYaml(nodeFile, map[string]string{}, "")
    }

    // 后安装 BKECluster
    err = global.K8s.InstallYaml(bkefile, map[string]string{}, "")

    // 等待集群启动
    global.K8s.WatchEventByAnnotation(bkeCluster.Namespace)
}
```
### 3.4 资源加载逻辑
**文件**: [pkg/cluster/bkecluster.go](file:///D:/code/github/bkeadm/pkg/cluster/bkecluster.go#L35-L48)
```go
func NewClusterResourcesFromFiles(clusterFile, nodesFile string) (*ClusterResources, error) {
    resources := &ClusterResources{}

    // 加载 BKECluster
    bkeCluster, err := loadBKEClusterFromFile(clusterFile)
    if err != nil {
        return nil, err
    }
    resources.BKECluster = bkeCluster

    // 加载 BKENodes
    bkeNodes, err := loadBKENodesFromFile(nodesFile)
    if err != nil {
        return nil, err
    }
    resources.BKENodes = bkeNodes

    // 处理和验证资源
    if err := processClusterResources(resources); err != nil {
        return nil, err
    }

    return resources, nil
}
```
## 4. 配置文件结构
### 4.1 BKECluster 配置文件示例
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKECluster
metadata:
  name: my-cluster
  namespace: bke-my-cluster
spec:
  pause: false
  dryRun: false
  reset: false
  clusterConfig:
    cluster:
      # 镜像仓库配置
      imageRepo:
        domain: cr.openfuyao.cn      # 仓库域名
        ip: 192.168.1.100            # 仓库 IP
        port: "443"                  # 端口
        prefix: openfuyao            # 镜像前缀
      
      # 软件源配置
      httpRepo:
        domain: openfuyao.cn
        ip: 192.168.1.100
        port: "40080"
      
      # Chart 仓库配置
      chartRepo:
        domain: charts.openfuyao.cn
        port: "443"
        prefix: charts
      
      # NTP 服务器
      ntpServer: ntp.openfuyao.cn
      
      # Kubernetes 版本
      kubernetesVersion: v1.28.0
      etcdVersion: v3.5.9
      containerdVersion: v1.7.0
      
      # Agent 健康检查端口
      agentHealthPort: "10256"
    
    # 插件配置
    addons:
      - name: calico
        version: v3.26.0
        param:
          calicoMode: vxlan
      - name: coredns
        version: v1.11.1
```
### 4.2 BKENodes 配置文件示例
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKENode
metadata:
  name: my-cluster-master-1
  namespace: bke-my-cluster
  labels:
    cluster.x-k8s.io/cluster-name: my-cluster
spec:
  hostname: master-1
  ip: 192.168.1.101
  username: root
  password: <encrypted-password>  # AES 加密
  port: "22"
  role:
    - master/node
    - etcd
---
apiVersion: bke.bocloud.com/v1beta1
kind: BKENode
metadata:
  name: my-cluster-worker-1
  namespace: bke-my-cluster
  labels:
    cluster.x-k8s.io/cluster-name: my-cluster
spec:
  hostname: worker-1
  ip: 192.168.1.102
  username: root
  password: <encrypted-password>
  port: "22"
  role:
    - worker
```
## 5. 执行流程图
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        bke init --file 执行流程                                          │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌───────────┐                                                                         │
│  │   开始    │                                                                         │
│  └─────┬─────┘                                                                         │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                        PreRunE: 参数解析                                         │   │
│  │                                                                                   │   │
│  │   1. 获取本机 IP                                                                  │   │
│  │   2. 解析 bkecluster.yaml                                                        │   │
│  │   3. 提取 ImageRepo → 设置 otherRepo                                             │   │
│  │   4. 提取 HTTPRepo → 设置 otherSource                                            │   │
│  │   5. 提取 ChartRepo → 设置 otherChart                                            │   │
│  │   6. 提取 NTPServer → 设置 ntpServer                                             │   │
│  │   7. 判断在线/离线模式                                                            │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                        Initialize(): 标准初始化                                   │   │
│  │                                                                                   │   │
│  │   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐          │   │
│  │   │节点信息 │──▶│环境校验 │──▶│时区设置 │──▶│环境准备 │──▶│容器服务 │          │   │
│  │   │  收集   │   │         │   │         │   │         │   │  启动   │          │   │
│  │   └─────────┘   └─────────┘   └─────────┘   └─────────┘   └─────┬───┘          │   │
│  │                                                                  │               │   │
│  │                                                                  ▼               │   │
│  │   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐          │   │
│  │   │仓库服务 │──▶│Cluster  │──▶│Console  │──▶│生成配置 │──▶│部署集群 │          │   │
│  │   │  启动   │   │  API    │   │ (可选)  │   │  文件   │   │         │          │   │
│  │   └─────────┘   └─────────┘   └─────────┘   └─────────┘   └─────┬───┘          │   │
│  │                                                                  │               │   │
│  └──────────────────────────────────────────────────────────────────┼───────────────┘   │
│                                                                     │                   │
│                                                        ┌────────────┴────────────┐      │
│                                                        │  op.File != "" ?        │      │
│                                                        └────────────┬────────────┘      │
│                                                                     │                   │
│                                              ┌──────────────────────┼──────────────────┐  │
│                                              ▼                                         ▼  │
│                                        ┌───────────┐                           ┌───────────┐
│                                        │    是     │                           │    否     │
│                                        └─────┬─────┘                           └─────┬─────┘
│                                              │                                       │     │
│                                              ▼                                       │     │
│  ┌───────────────────────────────────────────────────────────────────────────────┐   │     │
│  │                        deployCluster(): 集群部署                               │   │     │
│  │                                                                               │   │     │
│  │   ┌─────────────────────────────────────────────────────────────────────┐    │   │     │
│  │   │  cluster.Options{                                                    │    │   │     │
│  │   │      File:      op.File,         // bkecluster.yaml                 │    │   │     │
│  │   │      NtpServer: op.NtpServer,    // NTP 服务器                       │    │   │     │
│  │   │      NodesFile: ""               // ⚠️ 未设置                        │    │   │     │
│  │   │  }                                                                   │    │   │     │
│  │   └─────────────────────────────────────────────────────────────────────┘    │   │     │
│  │                                                                               │   │     │
│  │   ┌─────────────────────────────────────────────────────────────────────┐    │   │     │
│  │   │  c.Cluster()                                                         │    │   │     │
│  │   │                                                                       │    │   │     │
│  │   │  1. NewClusterResourcesFromFiles(File, NodesFile)                    │    │   │     │
│  │   │     • NodesFile = "" → loadBKENodesFromFile("") → ERROR!             │    │   │     │
│  │   │                                                                       │    │   │     │
│  │   │  问题: NodesFile 为空会导致加载节点失败                                │    │   │     │
│  │   └─────────────────────────────────────────────────────────────────────┘    │   │     │
│  └───────────────────────────────────────────────────────────────────────────────┘   │     │
│                                                                                       │     │
│                                              ┌────────────────────────────────────────┘     │
│                                              │                                           │
│                                              ▼                                           │
│                                        ┌───────────┐                                    │
│                                        │   完成    │                                    │
│                                        └───────────┘                                    │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
## 6. 当前设计问题分析
### 6.1 问题：NodesFile 未设置
**问题描述**:
在 `deployCluster()` 中创建 `cluster.Options` 时，只设置了 `File` 和 `NtpServer`，没有设置 `NodesFile`：
```go
func (op *Options) deployCluster() {
    c := cluster.Options{
        File:      op.File,
        NtpServer: op.NtpServer,
        // NodesFile 未设置，默认为空字符串
    }
    c.Cluster()
}
```
**影响**:
- `NewClusterResourcesFromFiles(file, "")` 会尝试读取空文件路径
- `loadBKENodesFromFile("")` 会失败，因为 `os.ReadFile("")` 返回错误
### 6.2 解决方案建议
**方案 1: 添加 --nodes 参数**
```go
// cmd/init.go
initCmd.Flags().StringVarP(&initOption.NodesFile, "nodes", "n", "", "BKENodes YAML file")

// pkg/initialize/initialize.go
func (op *Options) deployCluster() {
    if op.File == "" {
        return
    }
    c := cluster.Options{
        File:      op.File,
        NodesFile: op.NodesFile,  // 新增
        NtpServer: op.NtpServer,
    }
    c.Cluster()
}
```
**方案 2: 自动推断 Nodes 文件路径**
```go
func (op *Options) deployCluster() {
    if op.File == "" {
        return
    }
    
    // 自动推断 nodes 文件路径
    nodesFile := inferNodesFilePath(op.File)
    
    c := cluster.Options{
        File:      op.File,
        NodesFile: nodesFile,
        NtpServer: op.NtpServer,
    }
    c.Cluster()
}

func inferNodesFilePath(clusterFile string) string {
    // 规则: bkecluster.yaml → bkenodes.yaml
    dir := filepath.Dir(clusterFile)
    return filepath.Join(dir, "bkenodes.yaml")
}
```
**方案 3: 支持内嵌 Nodes 的配置文件**
修改 `loadBKENodesFromFile` 支持从 BKECluster 文件中读取内嵌的节点信息：
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKECluster
metadata:
  name: my-cluster
spec:
  clusterConfig: {...}
  nodes:  # 内嵌节点信息
    - hostname: master-1
      ip: 192.168.1.101
      role: [master/node, etcd]
```
## 7. 典型使用场景
### 7.1 在线模式部署
```bash
# 配置文件指定了外部仓库
bke init --file /path/to/bkecluster.yaml

# 等价于
bke init --otherRepo cr.openfuyao.cn/openfuyao \
         --otherSource http://openfuyao.cn:40080 \
         --otherChart oci://charts.openfuyao.cn/charts
```
### 7.2 离线模式部署
```bash
# 配置文件中的仓库 IP 为本机 IP
bke init --file /path/to/bkecluster.yaml

# 等价于
bke init  # 使用本地仓库
```
### 7.3 混合模式
```bash
# 镜像仓库使用外部，软件源使用本地
bke init --file /path/to/bkecluster.yaml --otherSource ""
```
## 8. 总结
`bke init --file` 的设计目标是实现声明式集群初始化，通过从配置文件中自动提取仓库信息来简化操作流程。当前实现存在 `NodesFile` 未设置的问题，需要进一步完善才能实现完整的自动化部署流程。

**核心价值**:
1. 减少手动参数输入
2. 自动判断在线/离线模式
3. 支持配置文件版本化管理
4. 适合自动化运维场景


# `bke cluster` 设计思路与详细设计
## 1. 设计思路
### 1.1 核心目标
`bke cluster` 是 BKE 集群生命周期管理的核心命令，提供集群的创建、扩缩容、查询、删除等全生命周期管理能力。
### 1.2 设计原则
```
┌─────────────────────────────────────────────────────────────────────────┐
│                           设计原则                                       │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  1. 声明式 API                                                           │
│     • 使用 YAML 文件定义集群和节点配置                                    │
│     • 配置与执行分离，支持版本化管理                                      │
│     • 通过 Kubernetes CRD 实现声明式管理                                 │
│                                                                         │
│  2. 分离配置                                                             │
│     • BKECluster: 集群配置（版本、网络、插件等）                          │
│     • BKENodes: 节点配置（IP、角色、密码等）                              │
│     • 配置分离便于复用和维护                                             │
│                                                                         │
│  3. Controller 模式                                                      │
│     • 命令只负责提交 CR 到管理集群                                        │
│     • 实际的集群创建由 cluster-api-provider-bke Controller 执行          │
│     • 支持异步操作和状态监控                                              │
│                                                                         │
│  4. 安全性                                                               │
│     • 节点密码自动 AES 加密                                              │
│     • 支持已加密密码的幂等处理                                            │
│                                                                         │
│  5. 可观测性                                                             │
│     • 实时事件监控                                                       │
│     • 支持查看集群部署日志                                                │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```
### 1.3 命令结构
```
bke cluster
├── list        # 列出所有集群
├── create      # 创建新集群
├── scale       # 扩缩容集群
├── remove      # 删除集群
├── logs        # 查看集群事件
└── exist       # 纳管已有集群
```
## 2. 整体架构
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              bke cluster 整体架构                                        │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                              用户层                                               │   │
│  │                                                                                   │   │
│  │   ┌───────────────┐  ┌───────────────┐  ┌───────────────┐  ┌───────────────┐    │   │
│  │   │ bkecluster.   │  │ bkenodes.     │  │ kubeconfig    │  │ 命令行参数    │    │   │
│  │   │    yaml       │  │    yaml       │  │    file       │  │               │    │   │
│  │   └───────┬───────┘  └───────┬───────┘  └───────┬───────┘  └───────┬───────┘    │   │
│  └───────────┼──────────────────┼──────────────────┼──────────────────┼─────────────┘   │
│              │                  │                  │                  │                   │
│              └──────────────────┼──────────────────┼──────────────────┘                   │
│                                 ▼                  ▼                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                           CLI 命令层                                             │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │                        cmd/cluster.go                                    │   │   │
│  │   │                                                                          │   │   │
│  │   │   bke cluster list      → List()                                        │   │   │
│  │   │   bke cluster create    → Cluster()                                     │   │   │
│  │   │   bke cluster scale     → Scale()                                       │   │   │
│  │   │   bke cluster remove    → Remove()                                      │   │   │
│  │   │   bke cluster logs      → Log()                                         │   │   │
│  │   │   bke cluster exist     → ExistsCluster()                               │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │                     PreRunE: ClusterPre()                                │   │   │
│  │   │                                                                          │   │   │
│  │   │   • 初始化 Kubernetes 客户端                                             │   │   │
│  │   │   • 连接管理集群 (K3s)                                                   │   │   │
│  │   │   • 设置 global.K8s                                                      │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                 │                                                       │
│                                 ▼                                                       │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                         业务逻辑层                                               │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │                     pkg/cluster/                                         │   │   │
│  │   │                                                                          │   │   │
│  │   │   cluster.go      → List(), Remove(), Scale(), Log(), ExistsCluster()   │   │   │
│  │   │   deploy.go       → Cluster() (创建集群)                                 │   │   │
│  │   │   bkecluster.go   → 资源加载、验证、处理                                 │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │                 NewClusterResourcesFromFiles()                           │   │   │
│  │   │                                                                          │   │   │
│  │   │   1. loadBKEClusterFromFile()     → 解析 BKECluster YAML                │   │   │
│  │   │   2. loadBKENodesFromFile()       → 解析 BKENodes YAML                  │   │   │
│  │   │   3. processClusterResources()    → 验证、加密、设置标签                 │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                 │                                                       │
│                                 ▼                                                       │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                         Kubernetes API 层                                        │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │                   pkg/executor/k8s/                                      │   │   │
│  │   │                                                                          │   │   │
│  │   │   InstallYaml()           → 创建 Kubernetes 资源                         │   │   │
│  │   │   PatchYaml()             → 更新 Kubernetes 资源                         │   │   │
│  │   │   WatchEventByAnnotation() → 监控事件                                    │   │   │
│  │   │   CreateNamespace()       → 创建命名空间                                 │   │   │
│  │   │   CreateSecret()          → 创建 Secret                                  │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                 │                                                       │
│                                 ▼                                                       │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                         管理集群 (K3s)                                           │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │                   Kubernetes CRD 资源                                    │   │   │
│  │   │                                                                          │   │   │
│  │   │   BKECluster CRD        → 集群配置                                       │   │   │
│  │   │   BKENode CRD           → 节点配置                                       │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │                   Cluster API Provider BKE                               │   │   │
│  │   │                                                                          │   │   │
│  │   │   BKECluster Controller → 监听 BKECluster CR，执行集群创建               │   │   │
│  │   │   BKENode Controller    → 监听 BKENode CR，执行节点管理                  │   │   │
│  │   │   Command Controller    → 执行远程命令                                   │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
## 3. 子命令详细设计
### 3.1 `bke cluster create` - 创建集群
**命令格式**:
```bash
bke cluster create -f /bke/cluster/bkecluster.yaml -n /bke/cluster/bkenodes.yaml
```
**参数说明**:
| 参数 | 简写 | 必填 | 说明 |
|------|------|------|------|
| `--file` | `-f` | 是 | BKECluster 配置文件 |
| `--nodes` | `-n` | 是 | BKENodes 配置文件 |

**执行流程**:
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        bke cluster create 执行流程                                       │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌───────────┐                                                                         │
│  │   开始    │                                                                         │
│  └─────┬─────┘                                                                         │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 1: ClusterPre() - 初始化 Kubernetes 客户端                                 │   │
│  │                                                                                   │   │
│  │   • 检查 global.K8s 是否已初始化                                                 │   │
│  │   • 未初始化则创建新的 KubernetesClient                                          │   │
│  │   • 连接管理集群的 kubeconfig                                                    │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 2: NewClusterResourcesFromFiles() - 加载配置文件                           │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │  loadBKEClusterFromFile(file)                                            │   │   │
│  │   │                                                                          │   │   │
│  │   │  • 读取 YAML 文件                                                        │   │   │
│  │   │  • 解析为 configv1beta1.BKECluster                                       │   │   │
│  │   │  • 验证 ClusterConfig 不为空                                             │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │  loadBKENodesFromFile(nodesFile)                                         │   │   │
│  │   │                                                                          │   │   │
│  │   │  • 支持 BKENodeList 格式 (items 数组)                                    │   │   │
│  │   │  • 支持多文档 YAML (--- 分隔符)                                          │   │   │
│  │   │  • 支持单个 BKENode 格式                                                 │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 3: processClusterResources() - 处理资源                                    │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │  验证配置                                                                │   │   │
│  │   │                                                                          │   │   │
│  │   │  • NewBkeConfigFromClusterConfig() → 转换配置                           │   │   │
│  │   │  • bc.Validate() → 验证配置合法性                                        │   │   │
│  │   │  • ValidateBKENodes() → 验证节点配置                                     │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │  密码加密                                                                │   │   │
│  │   │                                                                          │   │   │
│  │   │  for each node:                                                          │   │   │
│  │   │      _, err := security.AesDecrypt(node.Spec.Password)                  │   │   │
│  │   │      if err != nil:                                                      │   │   │
│  │   │          // 未加密，执行加密                                              │   │   │
│  │   │          p, _ := security.AesEncrypt(node.Spec.Password)                │   │   │
│  │   │          node.Spec.Password = p                                          │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                   │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │  设置标签和注解                                                          │   │   │
│  │   │                                                                          │   │   │
│  │   │  • node.Labels["cluster.x-k8s.io/cluster-name"] = cluster.Name          │   │   │
│  │   │  • node.Namespace = cluster.Namespace                                   │   │   │
│  │   │  • CustomExtra[BKEAgentListenerAnnotationKey] = Current                 │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 4: 写入 YAML 文件到本地                                                    │   │
│  │                                                                                   │   │
│  │   • marshalAndWriteClusterYAML() → /bke/cluster/{ns}-{name}.yaml               │   │
│  │   • marshalAndWriteNodeYAMLs()  → /bke/cluster/{ns}-{node}-node.yaml           │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 5: 创建 Namespace                                                          │   │
│  │                                                                                   │   │
│  │   • global.K8s.CreateNamespace(&namespace)                                      │   │
│  │   • Namespace 名称: bke-{cluster-name}                                          │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 6: 安装 BKENodes (先于 BKECluster)                                         │   │
│  │                                                                                   │   │
│  │   for each nodeFile:                                                             │   │
│  │       global.K8s.InstallYaml(nodeFile, {}, "")                                  │   │
│  │                                                                                   │   │
│  │   原因: Webhook 需要节点来设置默认 ControlPlaneEndpoint                          │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 7: 安装 BKECluster                                                         │   │
│  │                                                                                   │   │
│  │   global.K8s.InstallYaml(bkefile, {}, "")                                       │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 8: 监控集群启动                                                            │   │
│  │                                                                                   │   │
│  │   global.K8s.WatchEventByAnnotation(namespace)                                  │   │
│  │   • 实时显示集群创建事件                                                         │   │
│  │   • 等待集群就绪                                                                 │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│        │                                                                               │
│        ▼                                                                               │
│  ┌───────────┐                                                                         │
│  │   完成    │                                                                         │
│  └───────────┘                                                                         │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
**核心代码**:

**文件**: [pkg/cluster/deploy.go](file:///D:/code/github/bkeadm/pkg/cluster/deploy.go#L31-L80)
```go
func (op *Options) Cluster() {
    // 1. 加载配置文件
    resources, err := NewClusterResourcesFromFiles(op.File, op.NodesFile)
    
    bkeCluster := resources.BKECluster
    bkeNodes := resources.BKENodes

    // 2. 覆盖 NTP 服务器
    if len(op.NtpServer) > 0 {
        bkeCluster.Spec.ClusterConfig.Cluster.NTPServer = op.NtpServer
    }

    // 3. 规范化 namespace
    if !strings.HasPrefix(bkeCluster.Namespace, "bke") {
        bkeCluster.Namespace = "bke-" + bkeCluster.Namespace
    }

    // 4. 写入 YAML 文件
    bkefile, _ := marshalAndWriteClusterYAML(bkeCluster)
    nodeFiles, _ := marshalAndWriteNodeYAMLs(bkeCluster.Namespace, bkeNodes)

    // 5. 创建 Namespace
    global.K8s.CreateNamespace(&namespace)

    // 6. 先安装 BKENodes
    for _, nodeFile := range nodeFiles {
        global.K8s.InstallYaml(nodeFile, map[string]string{}, "")
    }

    // 7. 后安装 BKECluster
    global.K8s.InstallYaml(bkefile, map[string]string{}, "")

    // 8. 监控事件
    global.K8s.WatchEventByAnnotation(bkeCluster.Namespace)
}
```
### 3.2 `bke cluster scale` - 扩缩容集群
**命令格式**:
```bash
bke cluster scale -f /bke/cluster/bkecluster.yaml -n /bke/cluster/bkenodes.yaml
```
**执行流程**:
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        bke cluster scale 执行流程                                        │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  与 create 类似，但使用 PatchYaml 而非 InstallYaml                                       │
│                                                                                         │
│  Step 1: 加载配置文件                                                                   │
│  Step 2: 处理资源 (验证、加密)                                                           │
│  Step 3: 写入 YAML 文件                                                                 │
│  Step 4: Patch BKECluster → 更新现有资源                                                │
│  Step 5: Patch BKENodes → 更新现有节点                                                  │
│  Step 6: 监控事件                                                                       │
│                                                                                         │
│  关键区别:                                                                              │
│  • create: InstallYaml() → 创建新资源                                                   │
│  • scale:  PatchYaml()   → 更新现有资源                                                 │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
**核心代码**:

**文件**: [pkg/cluster/cluster.go](file:///D:/code/github/bkeadm/pkg/cluster/cluster.go#L158-L190)
```go
func (op *Options) Scale() {
    resources, err := NewClusterResourcesFromFiles(op.File, op.NodesFile)
    
    bkeFile, _ := marshalAndWriteClusterYAML(resources.BKECluster)
    nodeFiles, _ := marshalAndWriteNodeYAMLs(resources.BKECluster.Namespace, resources.BKENodes)

    // 使用 Patch 更新
    global.K8s.PatchYaml(bkeFile, map[string]string{})
    
    for _, nodeFile := range nodeFiles {
        global.K8s.PatchYaml(nodeFile, map[string]string{})
    }

    global.K8s.WatchEventByAnnotation(resources.BKECluster.Namespace)
}
```
### 3.3 `bke cluster list` - 列出集群
**命令格式**:
```bash
bke cluster list
```
**输出示例**:
```
namespace       name        endpoint            master  worker  pause   dryRun  reset
bke-cluster-1   my-cluster  192.168.1.100:6443  0       0       false   false   false
bke-cluster-2   test        192.168.1.101:6443  0       0       false   false   false
```
**核心代码**:

**文件**: [pkg/cluster/cluster.go](file:///D:/code/github/bkeadm/pkg/cluster/cluster.go#L77-L110)
```go
func (op *Options) List() {
    bclusterlist := &configv1beta1.BKEClusterList{}
    global.ListK8sResources(gvr, bclusterlist)

    headers := []string{"namespace", "name", "endpoint", "master", "worker", "pause", "dryRun", "reset"}
    
    // 使用 tabwriter 格式化输出
    w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
    fmt.Fprintln(w, strings.Join(headers, "\t"))
    for _, row := range rows {
        fmt.Fprintln(w, strings.Join(row, "\t"))
    }
    w.Flush()
}
```
### 3.4 `bke cluster remove` - 删除集群
**命令格式**:
```bash
bke cluster remove bke-my-cluster/my-cluster
```
**参数格式**: `namespace/name`

**执行流程**:
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        bke cluster remove 执行流程                                       │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  Step 1: 解析 namespace/name 参数                                                       │
│  Step 2: 获取 BKECluster CR                                                            │
│  Step 3: 设置 Spec.Reset = true                                                        │
│  Step 4: 更新 CR                                                                        │
│  Step 5: Controller 检测到 Reset=true，执行清理                                         │
│                                                                                         │
│  注意:                                                                                  │
│  • 不是直接删除 CR，而是设置 Reset 标志                                                 │
│  • Controller 负责实际的清理工作                                                        │
│  • 清理完成后 Controller 会删除 CR                                                      │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
**核心代码**:

**文件**: [pkg/cluster/cluster.go](file:///D:/code/github/bkeadm/pkg/cluster/cluster.go#L112-L145)
```go
func (op *Options) Remove() {
    ns := strings.Split(op.Args[0], "/")
    
    // 获取 BKECluster
    dynamicClient := global.K8s.GetDynamicClient()
    workloadUnstructured, _ := dynamicClient.Resource(gvr).Namespace(ns[0]).Get(context.TODO(), ns[1], metav1.GetOptions{})
    
    bcluster := &configv1beta1.BKECluster{}
    runtime.DefaultUnstructuredConverter.FromUnstructured(workloadUnstructured.UnstructuredContent(), bcluster)
    
    // 设置 Reset 标志
    bcluster.Spec.Reset = true
    
    // 更新 CR
    t1, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(bcluster)
    dynamicClient.Resource(gvr).Namespace(bcluster.Namespace).Update(context.TODO(), &unstructured.Unstructured{Object: t1}, metav1.UpdateOptions{})
}
```
### 3.5 `bke cluster logs` - 查看集群事件
**命令格式**:
```bash
bke cluster logs bke-my-cluster/my-cluster
```
**核心代码**:
```go
func (op *Options) Log() {
    ns := strings.Split(op.Args[0], "/")
    global.K8s.WatchEventByAnnotation(ns[0])
}
```
### 3.6 `bke cluster exist` - 纳管已有集群
**命令格式**:
```bash
bke cluster exist --conf /path/to/kubeconfig -f /path/to/bkecluster.yaml
```
**参数说明**:
| 参数 | 简写 | 必填 | 说明 |
|------|------|------|------|
| `--file` | `-f` | 是 | BKECluster 配置文件 |
| `--conf` | `-c` | 是 | 目标集群的 kubeconfig 文件 |

**执行流程**:
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        bke cluster exist 执行流程                                        │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  Step 1: 加载 BKECluster 配置                                                           │
│  Step 2: 创建 Namespace                                                                 │
│  Step 3: 创建 kubeconfig Secret                                                         │
│          • Secret 名称: {cluster-name}-kubeconfig                                       │
│          • Data: kubeconfig 文件内容                                                    │
│  Step 4: 安装 BKECluster CR                                                             │
│                                                                                         │
│  用途:                                                                                  │
│  • 纳管已存在的 Kubernetes 集群                                                          │
│  • 升级旧版本 K8s 集群                                                                   │
│  • 管理其他类型集群                                                                      │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
**核心代码**:

**文件**: [pkg/cluster/cluster.go](file:///D:/code/github/bkeadm/pkg/cluster/cluster.go#L228-L264)
```go
func (op *Options) ExistsCluster() {
    conf, _ := loadClusterConfig(op.File)
    
    // 创建 Namespace
    k8s.CreateNamespace(global.K8s, conf.Namespace)
    
    // 创建 kubeconfig Secret
    createKubeconfigSecret(conf.Namespace, conf.Name, op.Conf)
    
    // 安装 BKECluster
    global.K8s.InstallYaml(op.File, map[string]string{}, "")
}

func createKubeconfigSecret(namespace, name, confPath string) error {
    kubeConfig, _ := os.ReadFile(confPath)
    
    secret := corev1.Secret{
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
```
## 4. 配置文件结构
### 4.1 BKECluster 配置
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKECluster
metadata:
  name: my-cluster
  namespace: bke-my-cluster
spec:
  pause: false          # 是否暂停集群
  dryRun: false         # 是否试运行
  reset: false          # 是否重置（删除）
  
  controlPlaneEndpoint:
    host: 192.168.1.100
    port: 6443
  
  clusterConfig:
    cluster:
      kubernetesVersion: v1.28.0
      etcdVersion: v3.5.9
      containerdVersion: v1.7.0
      
      imageRepo:
        domain: cr.openfuyao.cn
        ip: 192.168.1.100
        port: "443"
        prefix: openfuyao
      
      httpRepo:
        domain: openfuyao.cn
        ip: 192.168.1.100
        port: "40080"
      
      chartRepo:
        domain: charts.openfuyao.cn
        port: "443"
        prefix: charts
      
      ntpServer: ntp.openfuyao.cn
      agentHealthPort: "10256"
    
    addons:
      - name: calico
        version: v3.26.0
        param:
          calicoMode: vxlan
      - name: coredns
        version: v1.11.1
```
### 4.2 BKENodes 配置
**方式 1: BKENodeList 格式**
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKENodeList
items:
  - metadata:
      name: my-cluster-master-1
      namespace: bke-my-cluster
    spec:
      hostname: master-1
      ip: 192.168.1.101
      username: root
      password: <encrypted>
      port: "22"
      role:
        - master/node
        - etcd
  - metadata:
      name: my-cluster-worker-1
    spec:
      hostname: worker-1
      ip: 192.168.1.102
      username: root
      password: <encrypted>
      port: "22"
      role:
        - worker
```
**方式 2: 多文档 YAML 格式**
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKENode
metadata:
  name: my-cluster-master-1
spec:
  hostname: master-1
  ip: 192.168.1.101
  role: [master/node, etcd]
---
apiVersion: bke.bocloud.com/v1beta1
kind: BKENode
metadata:
  name: my-cluster-worker-1
spec:
  hostname: worker-1
  ip: 192.168.1.102
  role: [worker]
```

**方式 3: 单个 BKENode 格式**
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKENode
metadata:
  name: my-cluster-master-1
spec:
  hostname: master-1
  ip: 192.168.1.101
  username: root
  password: password123
  port: "22"
  role:
    - master/node
    - etcd
```
## 5. 资源处理流程
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        processClusterResources() 详细流程                                │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  输入: ClusterResources { BKECluster, BKENodes }                                        │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 1: 配置转换与验证                                                          │   │
│  │                                                                                   │   │
│  │   bc, err := configinit.NewBkeConfigFromClusterConfig(clusterConfig)            │   │
│  │   if err != nil { return err }                                                   │   │
│  │                                                                                   │   │
│  │   if err := bc.Validate(); err != nil {                                         │   │
│  │       return err  // 配置验证失败                                                │   │
│  │   }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 2: 节点验证                                                                │   │
│  │                                                                                   │   │
│  │   if len(BKENodes) == 0 {                                                        │   │
│  │       return errors.New("at least one BKENode resource is required")            │   │
│  │   }                                                                               │   │
│  │                                                                                   │   │
│  │   if err := validation.ValidateBKENodes(BKENodes); err != nil {                  │   │
│  │       return err  // 节点验证失败                                                │   │
│  │   }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 3: 密码加密处理                                                            │   │
│  │                                                                                   │   │
│  │   for i := range BKENodes {                                                      │   │
│  │       node := &BKENodes[i]                                                       │   │
│  │                                                                                   │   │
│  │       // 尝试解密                                                                │   │
│  │       _, err := security.AesDecrypt(node.Spec.Password)                         │   │
│  │       if err != nil {                                                            │   │
│  │           // 解密失败 = 未加密，执行加密                                          │   │
│  │           p, err := security.AesEncrypt(node.Spec.Password)                     │   │
│  │           if err != nil { return err }                                           │   │
│  │           node.Spec.Password = p                                                 │   │
│  │       }                                                                           │   │
│  │       // 解密成功 = 已加密，跳过                                                  │   │
│  │   }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 4: 设置标签和命名空间                                                      │   │
│  │                                                                                   │   │
│  │   for i := range BKENodes {                                                      │   │
│  │       node := &BKENodes[i]                                                       │   │
│  │                                                                                   │   │
│  │       // 设置集群关联标签                                                        │   │
│  │       if node.Labels == nil {                                                    │   │
│  │           node.Labels = make(map[string]string)                                 │   │
│  │       }                                                                           │   │
│  │       node.Labels["cluster.x-k8s.io/cluster-name"] = BKECluster.Name            │   │
│  │                                                                                   │   │
│  │       // 设置命名空间                                                            │   │
│  │       if node.Namespace == "" {                                                  │   │
│  │           node.Namespace = BKECluster.Namespace                                 │   │
│  │       }                                                                           │   │
│  │   }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  Step 5: 设置 Agent 监听器注解                                                   │   │
│  │                                                                                   │   │
│  │   if BKECluster.Spec.ClusterConfig.CustomExtra == nil {                         │   │
│  │       BKECluster.Spec.ClusterConfig.CustomExtra = map[string]string{            │   │
│  │           bkecommon.BKEAgentListenerAnnotationKey: bkecommon.BKEAgentListenerCurrent │
│  │       }                                                                           │   │
│  │   } else {                                                                        │   │
│  │       // 确保注解存在                                                            │   │
│  │       if len(CustomExtra[BKEAgentListenerAnnotationKey]) == 0 {                  │   │
│  │           CustomExtra[BKEAgentListenerAnnotationKey] = BKEAgentListenerCurrent  │   │
│  │       }                                                                           │   │
│  │   }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  输出: 处理后的 ClusterResources                                                        │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
## 6. Kubernetes 客户端接口
**文件**: [pkg/executor/k8s/k8s.go](file:///D:/code/github/bkeadm/pkg/executor/k8s/k8s.go)
```go
type KubernetesClient interface {
    GetClient() kubernetes.Interface
    GetDynamicClient() dynamic.Interface
    
    // 资源操作
    InstallYaml(filename string, variable map[string]string, ns string) error
    PatchYaml(filename string, variable map[string]string) error
    UninstallYaml(filename string, ns string) error
    
    // 监控
    WatchEventByAnnotation(namespace string)
    
    // Namespace 和 Secret
    CreateNamespace(namespace *corev1.Namespace) error
    CreateSecret(secret *corev1.Secret) error
    GetNamespace(filename string) (string, error)
}
```
**关键方法说明**:

| 方法 | 用途 | create | scale | remove | exist |
|------|------|--------|-------|--------|-------|
| `InstallYaml` | 创建资源 | ✓ | - | - | ✓ |
| `PatchYaml` | 更新资源 | - | ✓ | - | - |
| `CreateNamespace` | 创建命名空间 | ✓ | - | - | ✓ |
| `CreateSecret` | 创建 Secret | - | - | - | ✓ |
| `WatchEventByAnnotation` | 监控事件 | ✓ | ✓ | - | - |
## 7. 典型使用场景
### 7.1 创建新集群
```bash
# 1. 准备配置文件
# /bke/cluster/bkecluster.yaml
# /bke/cluster/bkenodes.yaml

# 2. 创建集群
bke cluster create -f /bke/cluster/bkecluster.yaml -n /bke/cluster/bkenodes.yaml

# 3. 查看集群状态
bke cluster list

# 4. 查看部署日志
bke cluster logs bke-my-cluster/my-cluster
```
### 7.2 扩容集群
```bash
# 1. 修改 bkenodes.yaml，添加新节点

# 2. 执行扩容
bke cluster scale -f /bke/cluster/bkecluster.yaml -n /bke/cluster/bkenodes.yaml
```
### 7.3 删除集群
```bash
# 删除集群
bke cluster remove bke-my-cluster/my-cluster
```
### 7.4 纳管已有集群
```bash
# 纳管已存在的集群
bke cluster exist --conf /path/to/kubeconfig -f /bke/cluster/bkecluster.yaml
```
## 8. 总结
`bke cluster` 命令采用声明式 API 设计，通过 CRD 和 Controller 模式实现 Kubernetes 集群的全生命周期管理。核心特点包括：

1. **配置分离**: BKECluster 和 BKENodes 分离，便于复用和维护
2. **声明式管理**: 通过 YAML 文件定义期望状态，Controller 负责实现
3. **安全加密**: 节点密码自动 AES 加密，支持幂等处理
4. **实时监控**: 支持查看集群部署事件和状态
5. **纳管能力**: 支持纳管已存在的集群

# InstallYaml 与 PatchYaml 的区别与设计思路
## 1. 核心实现对比
### 1.1 InstallYaml 实现
**文件**: [pkg/executor/k8s/k8s.go](file:///D:/code/github/bkeadm/pkg/executor/k8s/k8s.go#L201-L225)
```go
func (c *Client) InstallYaml(filename string, variable map[string]string, ns string) error {
    tmpFile, cleanup, err := renderTemplateToTempFile(filename, variable)
    if err != nil {
        return err
    }
    defer cleanup()

    return c.processYamlResources(tmpFile, func(unstruct *unstructured.Unstructured, mapping *meta.RESTMapping) error {
        return c.createResource(unstruct, mapping, ns)  // 使用 Create
    })
}

func (c *Client) createResource(unstruct *unstructured.Unstructured, mapping *meta.RESTMapping, ns string) error {
    var obj *unstructured.Unstructured
    var err error

    targetNs := c.determineNamespace(unstruct, ns)
    if targetNs == "" {
        obj, err = c.DynamicClient.Resource(mapping.Resource).Create(context.Background(), unstruct, metav1.CreateOptions{})
    } else {
        obj, err = c.DynamicClient.Resource(mapping.Resource).Namespace(targetNs).Create(
            context.Background(), unstruct, metav1.CreateOptions{})
    }

    if err != nil {
        // 关键: 如果资源已存在，忽略错误
        if apierrors.IsAlreadyExists(err) {
            log.Debugf("%s/%s already exists", unstruct.GetKind(), unstruct.GetName())
            return nil  // 不报错，继续
        }
        return err
    }
    
    log.Debugf("%s/%s created", obj.GetKind(), obj.GetName())
    return nil
}
```
### 1.2 PatchYaml 实现
**文件**: [pkg/executor/k8s/k8s.go](file:///D:/code/github/bkeadm/pkg/executor/k8s/k8s.go#L247-L263)
```go
func (c *Client) PatchYaml(filename string, variable map[string]string) error {
    tmpFile, cleanup, err := renderTemplateToTempFile(filename, variable)
    if err != nil {
        return err
    }
    defer cleanup()

    return c.processYamlResources(tmpFile, func(unstruct *unstructured.Unstructured, mapping *meta.RESTMapping) error {
        jsonData, err := unstruct.MarshalJSON()
        if err != nil {
            return err
        }
        
        // 关键: 使用 Patch 更新现有资源
        _, err = c.DynamicClient.Resource(mapping.Resource).Namespace(unstruct.GetNamespace()).Patch(
            context.Background(), unstruct.GetName(), types.MergePatchType, jsonData, metav1.PatchOptions{})
        if err != nil {
            log.Errorf("failed patch %s/%s", unstruct.GetKind(), unstruct.GetName())
            return err  // 资源不存在会报错
        }
        return nil
    })
}
```
## 2. 详细对比分析
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        InstallYaml vs PatchYaml 对比                                     │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                           InstallYaml                                            │   │
│  │                                                                                   │   │
│  │   Kubernetes API: Create()                                                       │   │
│  │                                                                                   │   │
│  │   行为:                                                                          │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │  资源不存在 → 创建资源 ✓                                                 │   │   │
│  │   │  资源已存在 → 忽略错误，继续执行 ✓                                       │   │   │
│  │   │  其他错误   → 返回错误 ✗                                                 │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                   │   │
│  │   特点:                                                                          │   │
│  │   • 幂等性: 多次执行不会报错                                                     │   │
│  │   • 容错性: 资源已存在时静默忽略                                                 │   │
│  │   • 适用场景: 创建新资源、初始化部署                                             │   │
│  │                                                                                   │   │
│  │   HTTP 请求: POST /apis/{group}/{version}/namespaces/{ns}/{resource}            │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                           PatchYaml                                              │   │
│  │                                                                                   │   │
│  │   Kubernetes API: Patch() with MergePatchType                                   │   │
│  │                                                                                   │   │
│  │   行为:                                                                          │   │
│  │   ┌─────────────────────────────────────────────────────────────────────────┐   │   │
│  │   │  资源不存在 → 返回错误 ✗                                                 │   │   │
│  │   │  资源已存在 → 合并更新 ✓                                                 │   │   │
│  │   │  其他错误   → 返回错误 ✗                                                 │   │   │
│  │   └─────────────────────────────────────────────────────────────────────────┘   │   │
│  │                                                                                   │   │
│  │   特点:                                                                          │   │
│  │   • 非幂等: 资源不存在时会报错                                                   │   │
│  │   • 精确更新: 只更新 YAML 中指定的字段                                           │   │
│  │   • 适用场景: 更新现有资源、扩缩容                                               │   │
│  │                                                                                   │   │
│  │   HTTP 请求: PATCH /apis/{group}/{version}/namespaces/{ns}/{resource}/{name}    │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
## 3. 设计思路
### 3.1 为什么需要两个方法？
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              设计思路                                                    │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  场景 1: 创建集群                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                                                                                   │   │
│  │   用户期望: 创建一个全新的集群                                                    │   │
│  │                                                                                   │   │
│  │   操作:                                                                          │   │
│  │   1. 创建 Namespace (可能已存在)                                                 │   │
│  │   2. 创建 BKENodes (新资源)                                                      │   │
│  │   3. 创建 BKECluster (新资源)                                                    │   │
│  │                                                                                   │   │
│  │   需求:                                                                          │   │
│  │   • 如果资源已存在，不应该报错（幂等性）                                          │   │
│  │   • 支持重复执行                                                                 │   │
│  │                                                                                   │   │
│  │   解决方案: InstallYaml                                                          │   │
│  │   • Create() + IsAlreadyExists 容错                                              │   │
│  │                                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  场景 2: 扩缩容集群                                                  │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                                                                                   │   │
│  │   用户期望: 修改现有集群配置                                                      │   │
│  │                                                                                   │   │
│  │   操作:                                                                          │   │
│  │   1. 更新 BKECluster (修改副本数、配置等)                                        │   │
│  │   2. 更新 BKENodes (添加/删除节点)                                               │   │
│  │                                                                                   │   │
│  │   需求:                                                                          │   │
│  │   • 资源必须存在，否则是错误                                                     │   │
│  │   • 只更新变化的字段                                                             │   │
│  │                                                                                   │   │
│  │   解决方案: PatchYaml                                                            │   │
│  │   • Patch() with MergePatchType                                                  │   │
│  │                                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
### 3.2 MergePatchType 的工作原理
```yaml
# 原有资源
apiVersion: bke.bocloud.com/v1beta1
kind: BKENode
metadata:
  name: node-1
spec:
  hostname: node-1
  ip: 192.168.1.101
  role: [worker]

# Patch YAML (只包含要更新的字段)
apiVersion: bke.bocloud.com/v1beta1
kind: BKENode
metadata:
  name: node-1
spec:
  role: [master/node, etcd]  # 只更新 role

# 合并后的结果
apiVersion: bke.bocloud.com/v1beta1
kind: BKENode
metadata:
  name: node-1
spec:
  hostname: node-1          # 保留
  ip: 192.168.1.101         # 保留
  role: [master/node, etcd] # 更新
```
## 4. 新增节点可以使用 PatchYaml 吗？
### 4.1 答案：**不可以**
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        新增节点使用 PatchYaml 的问题                                     │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  场景: 扩容集群，添加新节点 node-2                                                       │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  bkenodes.yaml                                                                   │   │
│  │                                                                                   │   │
│  │  ---                                                                              │   │
│  │  apiVersion: bke.bocloud.com/v1beta1                                            │   │
│  │  kind: BKENode                                                                   │   │
│  │  metadata:                                                                       │   │
│  │    name: node-1        # 已存在的节点                                            │   │
│  │  spec:                                                                           │   │
│  │    ip: 192.168.1.101                                                            │   │
│  │  ---                                                                              │   │
│  │  apiVersion: bke.bocloud.com/v1beta1                                            │   │
│  │  kind: BKENode                                                                   │   │
│  │  metadata:                                                                       │   │
│  │    name: node-2        # 新增节点                                                │   │
│  │  spec:                                                                           │   │
│  │    ip: 192.168.1.102                                                            │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  使用 PatchYaml 的结果:                                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                                                                                   │   │
│  │  Patch node-1 → 成功 (资源存在，更新)                                            │   │
│  │  Patch node-2 → 失败 (资源不存在，报错!)                                         │   │
│  │                                                                                   │   │
│  │  错误信息:                                                                        │   │
│  │  "bkenodes.bke.bocloud.com \"node-2\" not found"                                │   │
│  │                                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
### 4.2 当前 scale 命令的问题
**文件**: [pkg/cluster/cluster.go](file:///D:/code/github/bkeadm/pkg/cluster/cluster.go#L158-L190)
```go
func (op *Options) Scale() {
    resources, _ := NewClusterResourcesFromFiles(op.File, op.NodesFile)
    
    bkeFile, _ := marshalAndWriteClusterYAML(resources.BKECluster)
    nodeFiles, _ := marshalAndWriteNodeYAMLs(resources.BKECluster.Namespace, resources.BKENodes)

    // 问题: 使用 PatchYaml，新增节点会失败
    global.K8s.PatchYaml(bkeFile, map[string]string{})
    
    for _, nodeFile := range nodeFiles {
        global.K8s.PatchYaml(nodeFile, map[string]string{})  // 新节点会报错
    }
}
```
### 4.3 正确的实现方案
**方案 1: Server-Side Apply (推荐)**
```go
func (c *Client) ApplyYaml(filename string, variable map[string]string, ns string) error {
    tmpFile, cleanup, err := renderTemplateToTempFile(filename, variable)
    if err != nil {
        return err
    }
    defer cleanup()

    return c.processYamlResources(tmpFile, func(unstruct *unstructured.Unstructured, mapping *meta.RESTMapping) error {
        jsonData, err := unstruct.MarshalJSON()
        if err != nil {
            return err
        }
        
        // 使用 Server-Side Apply
        // 如果资源不存在则创建，存在则更新
        _, err = c.DynamicClient.Resource(mapping.Resource).Namespace(unstruct.GetNamespace()).Apply(
            context.Background(),
            unstruct.GetName(),
            &apply.ApplyOptions{  // 需要导入 applyconfigurations
                FieldManager: "bkeadm",
                Force:        pointer.Bool(true),
            },
        )
        return err
    })
}
```
**方案 2: 智能判断 (Create or Patch)**
```go
func (c *Client) CreateOrUpdateYaml(filename string, variable map[string]string, ns string) error {
    tmpFile, cleanup, err := renderTemplateToTempFile(filename, variable)
    if err != nil {
        return err
    }
    defer cleanup()

    return c.processYamlResources(tmpFile, func(unstruct *unstructured.Unstructured, mapping *meta.RESTMapping) error {
        // 先尝试获取资源
        existing, err := c.DynamicClient.Resource(mapping.Resource).Namespace(unstruct.GetNamespace()).Get(
            context.Background(), unstruct.GetName(), metav1.GetOptions{})
        
        if apierrors.IsNotFound(err) {
            // 资源不存在，创建
            return c.createResource(unstruct, mapping, ns)
        } else if err != nil {
            return err
        }
        
        // 资源存在，更新
        jsonData, err := unstruct.MarshalJSON()
        if err != nil {
            return err
        }
        _, err = c.DynamicClient.Resource(mapping.Resource).Namespace(unstruct.GetNamespace()).Patch(
            context.Background(), unstruct.GetName(), types.MergePatchType, jsonData, metav1.PatchOptions{})
        return err
    })
}
```
**方案 3: 修改 scale 逻辑，区分新增和更新**
```go
func (op *Options) Scale() {
    resources, _ := NewClusterResourcesFromFiles(op.File, op.NodesFile)
    
    // 获取现有节点
    existingNodes := op.getExistingNodes(resources.BKECluster.Namespace)
    
    // 区分新增和更新
    var newNodes, updateNodes []configv1beta1.BKENode
    for _, node := range resources.BKENodes {
        if _, exists := existingNodes[node.Name]; exists {
            updateNodes = append(updateNodes, node)
        } else {
            newNodes = append(newNodes, node)
        }
    }
    
    // 新增节点使用 InstallYaml
    for _, node := range newNodes {
        nodeFile := writeNodeYAML(node)
        global.K8s.InstallYaml(nodeFile, map[string]string{}, "")
    }
    
    // 更新节点使用 PatchYaml
    for _, node := range updateNodes {
        nodeFile := writeNodeYAML(node)
        global.K8s.PatchYaml(nodeFile, map[string]string{})
    }
    
    // 更新 BKECluster
    bkeFile, _ := marshalAndWriteClusterYAML(resources.BKECluster)
    global.K8s.PatchYaml(bkeFile, map[string]string{})
}
```
## 5. 总结对比表
| 特性 | InstallYaml | PatchYaml | 推荐方案 |
|------|-------------|-----------|-------------------|
| **资源不存在** | 创建 ✓ | 报错 ✗ | 创建 ✓ |
| **资源已存在** | 忽略，不更新 | 更新 ✓ | 更新 ✓ |
| **幂等性** | 是 | 否 | 是 |
| **新增节点** | 支持 ✓ | 不支持 ✗ | 支持 ✓ |
| **更新节点** | 不更新 | 支持 ✓ | 支持 ✓ |
| **适用场景** | 创建集群 | 更新现有资源 | 扩缩容、通用场景 |

## 6. 建议改进
对于 `bke cluster scale` 命令，建议：
1. **使用 Server-Side Apply** 或 **CreateOrUpdate 模式**，而不是单纯的 PatchYaml
2. **或者区分新增和更新节点**，分别使用 InstallYaml 和 PatchYaml

当前实现的问题：
- `scale` 命令使用 `PatchYaml`，如果添加新节点会失败
- 需要改进为支持"存在则更新，不存在则创建"的语义

# `bke init --file` 没有 nodes 参数的设计问题分析
## 1. 当前实现的问题
### 1.1 代码流程分析
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                        bke init --file 当前实现流程                                      │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  Step 1: deployCluster()                                                                │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  func (op *Options) deployCluster() {                                            │   │
│  │      if op.File == "" {                                                          │   │
│  │          return                                                                  │   │
│  │      }                                                                           │   │
│  │      c := cluster.Options{                                                       │   │
│  │          File:      op.File,        // ✓ 设置了                                  │   │
│  │          NtpServer: op.NtpServer,   // ✓ 设置了                                  │   │
│  │          // NodesFile: ???         // ✗ 没有设置，默认为空字符串 ""              │   │
│  │      }                                                                           │   │
│  │      c.Cluster()                                                                 │   │
│  │  }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                        │                                                │
│                                        ▼                                                │
│  Step 2: Cluster()                                                                      │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  func (op *Options) Cluster() {                                                  │   │
│  │      resources, err := NewClusterResourcesFromFiles(op.File, op.NodesFile)      │   │
│  │      // op.NodesFile == ""                                                       │   │
│  │  }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                        │                                                │
│                                        ▼                                                │
│  Step 3: NewClusterResourcesFromFiles()                                                 │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  func NewClusterResourcesFromFiles(clusterFile, nodesFile string) {             │   │
│  │      // clusterFile = "bkecluster.yaml"                                          │   │
│  │      // nodesFile = ""  ← 空字符串                                               │   │
│  │                                                                                   │   │
│  │      bkeNodes, err := loadBKENodesFromFile(nodesFile)                           │   │
│  │      // nodesFile = "" → 会报错!                                                │   │
│  │  }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                        │                                                │
│                                        ▼                                                │
│  Step 4: loadBKENodesFromFile("")                                                       │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  func loadBKENodesFromFile(file string) ([]configv1beta1.BKENode, error) {       │   │
│  │      yamlFile, err := os.ReadFile(file)                                         │   │
│  │      // file = "" → os.ReadFile("") 返回错误!                                    │   │
│  │      // open : no such file or directory                                        │   │
│  │  }                                                                               │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  结果: 执行失败，报错 "open : no such file or directory"                                │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
### 1.2 问题根源
**文件**: [pkg/initialize/initialize.go](file:///D:/code/github/bkeadm/pkg/initialize/initialize.go#L1188-L1198)
```go
func (op *Options) deployCluster() {
    if op.File == "" {
        return
    }
    log.BKEFormat(log.INFO, "Starting to deploy the cluster...")
    c := cluster.Options{
        File:      op.File,
        NtpServer: op.NtpServer,
        // 问题: NodesFile 未设置，默认为空字符串
    }
    c.Cluster()
}
```
**文件**: [pkg/cluster/bkecluster.go](file:///D:/code/github/bkeadm/pkg/cluster/bkecluster.go#L73-L77)
```go
func loadBKENodesFromFile(file string) ([]configv1beta1.BKENode, error) {
    yamlFile, err := os.ReadFile(file)  // file = "" 会报错
    if err != nil {
        return nil, err
    }
    // ...
}
```
## 2. 设计意图分析
### 2.1 原始设计思路
```
┌─────────────────────────────────────────────────────────────────────────────────────────┐
│                              原始设计意图                                                │
├─────────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                         │
│  设计目标:                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │                                                                                   │   │
│  │  bke init --file bkecluster.yaml                                                 │   │
│  │                                                                                   │   │
│  │  期望行为:                                                                        │   │
│  │  1. 从 bkecluster.yaml 提取仓库信息                                              │   │
│  │  2. 初始化引导节点                                                                │   │
│  │  3. 自动部署集群 (使用 bkecluster.yaml 中定义的配置)                              │   │
│  │                                                                                   │   │
│  │  问题:                                                                            │   │
│  │  • BKECluster 和 BKENode 是分离的 CRD                                            │   │
│  │  • bkecluster.yaml 只包含集群配置，不包含节点信息                                 │   │
│  │  • 没有指定 nodes 文件，无法获取节点信息                                          │   │
│  │                                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  两种可能的设计意图:                                                                     │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  意图 A: 单文件模式 (未实现)                                                      │   │
│  │                                                                                   │   │
│  │  bkecluster.yaml 内嵌节点信息:                                                   │   │
│  │  ┌─────────────────────────────────────────────────────────────────────────┐    │   │
│  │  │  apiVersion: bke.bocloud.com/v1beta1                                    │    │   │
│  │  │  kind: BKECluster                                                       │    │   │
│  │  │  spec:                                                                  │    │   │
│  │  │    clusterConfig: {...}                                                 │    │   │
│  │  │    nodes:                        ← 内嵌节点                              │    │   │
│  │  │      - hostname: master-1                                              │    │   │
│  │  │        ip: 192.168.1.101                                               │    │   │
│  │  │        role: [master/node, etcd]                                       │    │   │
│  │  └─────────────────────────────────────────────────────────────────────────┘    │   │
│  │                                                                                   │   │
│  │  问题: 当前 BKECluster CRD 不支持内嵌 nodes 字段                                 │   │
│  │                                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  ┌─────────────────────────────────────────────────────────────────────────────────┐   │
│  │  意图 B: 自动推断 nodes 文件 (未实现)                                            │   │
│  │                                                                                   │   │
│  │  约定: bkecluster.yaml → bkenodes.yaml                                          │   │
│  │                                                                                   │   │
│  │  规则:                                                                            │   │
│  │  • bkecluster.yaml → 同目录下的 bkenodes.yaml                                   │   │
│  │  • 或 my-cluster.yaml → my-nodes.yaml                                           │   │
│  │                                                                                   │   │
│  │  问题: 当前代码没有实现这个自动推断逻辑                                           │   │
│  │                                                                                   │   │
│  └─────────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                         │
│  结论: 当前实现是一个 BUG，设计不完整                                                   │
│                                                                                         │
└─────────────────────────────────────────────────────────────────────────────────────────┘
```
### 2.2 对比 `bke init` (无 --file) 的行为
**文件**: [pkg/initialize/initialize.go](file:///D:/code/github/bkeadm/pkg/initialize/initialize.go#L1020-L1048)

```gofunc (op *Options) createClusterConfigFile(...) {
    // ...

    if op.File != "" {
        return  // 如果指定了 --file，跳过自动生成配置
    }

    // 没有指定 --file 时，自动生成配置文件
    c := config.Options{
        Directory:       fmt.Sprintf("%s/cluster", global.Workspace),
        Product:         "fuyao-allinone",
        Domain:          op.Domain,
        ImageRepoPort:   op.ImageRepoPort,
        AgentHealthPort: op.AgentHealthPort,
    }
    c.Config(global.CustomExtra, imageRepo, yumRepo, chartRepo, op.NtpServer)

    // 提示用户手动执行
    log.BKEFormat(log.HINT, fmt.Sprintf("Run `bke cluster create -f %s/cluster/1master-cluster.yaml -n %s/cluster/1master-node.yaml`", ...))
}
```
**关键发现**:
- `bke init` (无 `--file`) 会自动生成 `1master-cluster.yaml` 和 `1master-node.yaml`
- `bke init --file` 跳过自动生成，但 `deployCluster()` 没有正确处理 nodes 文件
## 3. 完整的设计方案
### 3.1 方案 A: 添加 `--nodes` 参数
```go
// cmd/init.go
initCmd.Flags().StringVarP(&initOption.NodesFile, "nodes", "n", "", "BKENodes YAML file")

// pkg/initialize/initialize.go
type Options struct {
    // ...
    File      string `json:"file"`
    NodesFile string `json:"nodesFile"` // 新增
    // ...
}

func (op *Options) deployCluster() {
    if op.File == "" {
        return
    }
    
    // 验证 nodes 文件
    if op.NodesFile == "" {
        log.BKEFormat(log.ERROR, "The --nodes parameter is required when using --file")
        return
    }
    
    c := cluster.Options{
        File:      op.File,
        NodesFile: op.NodesFile, // 设置 nodes 文件
        NtpServer: op.NtpServer,
    }
    c.Cluster()
}
```
**使用方式**:
```bash
bke init --file bkecluster.yaml --nodes bkenodes.yaml
```
### 3.2 方案 B: 自动推断 nodes 文件
```go
func (op *Options) deployCluster() {
    if op.File == "" {
        return
    }
    
    // 自动推断 nodes 文件路径
    nodesFile := op.NodesFile
    if nodesFile == "" {
        nodesFile = inferNodesFilePath(op.File)
    }
    
    // 验证文件存在
    if !utils.Exists(nodesFile) {
        log.BKEFormat(log.ERROR, "Nodes file not found: %s", nodesFile)
        log.BKEFormat(log.HINT, "Please specify --nodes parameter or create the nodes file")
        return
    }
    
    c := cluster.Options{
        File:      op.File,
        NodesFile: nodesFile,
        NtpServer: op.NtpServer,
    }
    c.Cluster()
}

func inferNodesFilePath(clusterFile string) string {
    dir := filepath.Dir(clusterFile)
    base := filepath.Base(clusterFile)
    
    // 规则 1: bkecluster.yaml → bkenodes.yaml
    if base == "bkecluster.yaml" {
        return filepath.Join(dir, "bkenodes.yaml")
    }
    
    // 规则 2: *-cluster.yaml → *-nodes.yaml
    if strings.HasSuffix(base, "-cluster.yaml") {
        return filepath.Join(dir, strings.TrimSuffix(base, "-cluster.yaml")+"-nodes.yaml")
    }
    
    // 规则 3: *.yaml → *-nodes.yaml
    ext := filepath.Ext(base)
    name := strings.TrimSuffix(base, ext)
    return filepath.Join(dir, name+"-nodes"+ext)
}
```
### 3.3 方案 C: 支持单文件模式 (修改 CRD)
**修改 BKECluster CRD**:
```yaml
apiVersion: bke.bocloud.com/v1beta1
kind: BKECluster
metadata:
  name: my-cluster
spec:
  clusterConfig:
    cluster:
      kubernetesVersion: v1.28.0
      # ...
  
  # 新增: 内嵌节点定义
  nodes:
    - hostname: master-1
      ip: 192.168.1.101
      username: root
      password: <encrypted>
      port: "22"
      role:
        - master/node
        - etcd
    - hostname: worker-1
      ip: 192.168.1.102
      role:
        - worker
```
**修改 loadBKENodesFromFile**:
```go
func loadBKENodesFromFile(file string, cluster *configv1beta1.BKECluster) ([]configv1beta1.BKENode, error) {
    // 优先从 cluster.spec.nodes 获取
    if cluster != nil && len(cluster.Spec.Nodes) > 0 {
        return cluster.Spec.Nodes, nil
    }
    
    // 如果没有内嵌节点，从文件加载
    if file == "" {
        return nil, errors.New("no nodes defined in cluster spec and no nodes file provided")
    }
    
    yamlFile, err := os.ReadFile(file)
    // ...
}
```
## 4. 推荐方案
### 4.1 短期修复: 方案 A + B 组合
```go
func (op *Options) deployCluster() {
    if op.File == "" {
        return
    }
    
    log.BKEFormat(log.INFO, "Starting to deploy the cluster...")
    
    // 确定 nodes 文件
    nodesFile := op.NodesFile
    if nodesFile == "" {
        // 尝试自动推断
        nodesFile = inferNodesFilePath(op.File)
        if utils.Exists(nodesFile) {
            log.BKEFormat(log.INFO, "Auto-detected nodes file: %s", nodesFile)
        } else {
            log.BKEFormat(log.ERROR, "Nodes file not specified and auto-detection failed")
            log.BKEFormat(log.HINT, "Please specify --nodes parameter")
            return
        }
    }
    
    c := cluster.Options{
        File:      op.File,
        NodesFile: nodesFile,
        NtpServer: op.NtpServer,
    }
    c.Cluster()
}
```
### 4.2 完整使用示例
```bash
# 方式 1: 显式指定 nodes 文件
bke init --file bkecluster.yaml --nodes bkenodes.yaml

# 方式 2: 自动推断 (约定: bkecluster.yaml → bkenodes.yaml)
bke init --file bkecluster.yaml

# 方式 3: 使用其他命名 (自动推断: my-cluster.yaml → my-nodes.yaml)
bke init --file my-cluster.yaml
```
## 5. 总结
| 问题 | 原因 | 解决方案 |
|------|------|----------|
| `NodesFile` 未设置 | `deployCluster()` 没有传递该参数 | 添加参数或自动推断 |
| `loadBKENodesFromFile("")` 报错 | 空字符串导致文件读取失败 | 增加空值检查 |
| 设计不完整 | 缺少单文件模式或自动推断逻辑 | 实现自动推断或修改 CRD |

**当前状态**: `bke init --file` 存在 BUG，无法正常工作，需要添加 `--nodes` 参数或实现自动推断逻辑。

  
# `bke command` 的设计思路与详细设计。
## bke command 设计思路与详细设计
### 一、设计思路
`bke command` 是一个基于 Kubernetes CRD 的远程命令执行框架，其核心设计思路是：
#### 1. 声明式命令管理
- 通过 Kubernetes CRD（Custom Resource Definition）定义命令资源
- 采用声明式 API，用户提交命令意图，系统负责实现
- 命令状态通过 CR 的 Status 字段反馈
#### 2. Agent 执行模式
- BKEAgent 运行在集群节点上，监听 Command CR
- Agent 根据节点选择器匹配并执行相应命令
- 执行结果回写到 CR 的 Status 字段
#### 3. 命令生命周期管理
- **创建**：通过 `exec` 子命令创建 Command CR
- **查看**：通过 `list` 和 `info` 子命令查看命令状态和结果
- **删除**：通过 `remove` 子命令删除命令资源
### 二、命令结构
```
bke command
├── exec       # 执行指定命令
├── list       # 列出所有命令
├── info       # 查看命令详情
├── remove     # 删除命令
└── syncTime   # 同步时间
```
### 三、详细设计
#### 3.1 Command CRD 定义
Command CRD 来自 `cluster-api-provider-bke` 仓库，其核心结构如下：
```go
type Command struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    
    Spec   CommandSpec   `json:"spec,omitempty"`
    Status CommandStatus `json:"status,omitempty"`
}

type CommandSpec struct {
    Suspend      bool             `json:"suspend"`       // 是否暂停执行
    NodeSelector *LabelSelector   `json:"nodeSelector"`  // 节点选择器
    Commands     []ExecCommand    `json:"commands"`      // 执行命令列表
}

type ExecCommand struct {
    ID            string        `json:"id"`            // 命令ID
    Command       []string      `json:"command"`       // 命令内容
    Type          CommandType   `json:"type"`          // 命令类型
    BackoffIgnore bool          `json:"backoffIgnore"` // 是否忽略重试
    BackoffDelay  int           `json:"backoffDelay"`  // 重试延迟
}

type CommandStatus map[string]NodeCommandStatus  // key: node name

type NodeCommandStatus struct {
    Phase          CommandPhase     `json:"phase"`          // 执行阶段
    Status         CommandStatus    `json:"status"`         // 执行状态
    LastStartTime  *metav1.Time     `json:"lastStartTime"`  // 最后开始时间
    CompletionTime *metav1.Time     `json:"completionTime"` // 完成时间
    Conditions     []Condition      `json:"conditions"`     // 执行条件
}
```
#### 3.2 exec 子命令详细设计
**功能**：在指定节点上执行命令

**参数**：
- `--name, -n`：命令名称（可选，默认为时间戳格式 `200601021504`）
- `--file, -f`：Shell 脚本文件路径
- `--command, -c`：直接执行的 Shell 命令
- `--nodes`：目标节点列表（必填，逗号分隔）

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│                    bke command exec                      │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   参数校验                         │
        │   - nodes 必填                     │
        │   - file 或 command 必须存在一个    │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   初始化 Kubernetes Client        │
        │   (ClusterPre)                   │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   构建 Command CR                 │
        │   - 设置 name, namespace          │
        │   - 构建节点选择器                 │
        │   - 添加命令到 Spec.Commands       │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   处理命令内容                     │
        │                                   │
        │   如果是 --command:               │
        │   └─> 添加到 Commands 列表         │
        │       Type: CommandShell          │
        │                                   │
        │   如果是 --file:                  │
        │   └─> 创建 ConfigMap              │
        │   └─> 添加到 Commands 列表         │
        │       Type: CommandKubernetes     │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   创建 Command CR 到 Kubernetes   │
        │   - 序列化为 YAML                 │
        │   - 写入临时文件                   │
        │   - 调用 InstallYaml 创建资源      │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   输出提示信息                     │
        │   bke command info ns/name        │
        └──────────────────────────────────┘
```
**关键代码实现**：
[agent.go:57-78](file:///D:\code\github\bkeadm\pkg\agent\agent.go#L57-L78) - Exec 函数实现：
```go
func (op *Options) Exec() {
    cmd := op.buildCommand()
    if err := op.applyCommand(&cmd); err != nil {
        log.Error(err.Error())
        return
    }
    log.BKEFormat(log.NIL, fmt.Sprintf("The execution command has been sent to the cluster, Please run the `bke command info "+
        "%s/%s` to optain the execution result", metav1.NamespaceDefault, op.Name))
}
```
[agent.go:80-97](file:///D:\code\github\bkeadm\pkg\agent\agent.go#L80-L97) - 构建 Command CR：
```go
func (op *Options) buildCommand() agentv1beta1.Command {
    cmd := agentv1beta1.Command{
        ObjectMeta: metav1.ObjectMeta{
            Annotations: map[string]string{
                annotationKey: annotationValue,
            },
        },
        Spec: agentv1beta1.CommandSpec{
            Suspend:  false,
            Commands: []agentv1beta1.ExecCommand{},
        },
    }
    cmd.SetGroupVersionKind(schema.GroupVersionKind{...})
    cmd.SetName(op.Name)
    cmd.SetNamespace(metav1.NamespaceDefault)
    cmd.Spec.NodeSelector = op.buildNodeSelector()
    return cmd
}
```
[agent.go:99-104](file:///D:\code\github\bkeadm\pkg\agent\agent.go#L99-L104) - 构建节点选择器：
```go
func (op *Options) buildNodeSelector() *metav1.LabelSelector {
    nodeMap := map[string]string{}
    for _, value := range strings.Split(op.Nodes, ",") {
        nodeMap[value] = value
    }
    return &metav1.LabelSelector{MatchLabels: nodeMap}
}
```
#### 3.3 list 子命令详细设计
**功能**：列出所有命令及其执行状态

**参数**：无

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│                    bke command list                      │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   初始化 Kubernetes Client        │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   列出所有 Command CR             │
        │   GET /apis/.../commands          │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   格式化输出表格                   │
        │   name | suspend | node | ...     │
        └──────────────────────────────────┘
```
**输出格式**：
```
name            suspend  node           LastStartTime  CompletionTime  Phase   Status
default/cmd001  false    192.168.1.100  2025-04-01...  2025-04-01...   Succeeded  Success
```
**关键代码实现**：
[agent.go:150-173](file:///D:\code\github\bkeadm\pkg\agent\agent.go#L150-L173) - List 函数实现：
```go
func (op *Options) List() {
    commandList := &agentv1beta1.CommandList{}
    err := global.ListK8sResources(gvr, commandList)
    if err != nil {
        return
    }
    const tabPadding = 2
    w := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
    fmt.Fprintln(w, "name\tsuspend\tnode\tLastStartTime\tCompletionTime\tPhase\tStatus")
    for _, bc := range commandList.Items {
        for node, v := range bc.Status {
            completionTime := ""
            if v.CompletionTime != nil {
                completionTime = v.CompletionTime.String()
            }
            fmt.Fprintf(w, "%s/%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
                bc.Namespace, bc.Name,
                strconv.FormatBool(bc.Spec.Suspend),
                node,
                v.LastStartTime.String(),
                completionTime,
                string(v.Phase),
                string(v.Status),
            )
        }
    }
    err = w.Flush()
}
```
#### 3.4 info 子命令详细设计
**功能**：查看指定命令的详细执行结果

**参数**：
- 位置参数：`namespace/name`（命令的命名空间和名称）

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│              bke command info ns/name                    │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   解析 namespace/name 参数        │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   获取 Command CR                │
        │   GET /apis/.../commands/name     │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   遍历 Status 字段                │
        │   - 每个节点一个状态               │
        │   - 包含执行条件和输出             │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   输出命令详情和执行结果           │
        │   - 命令基本信息表格               │
        │   - 标准输出/标准错误              │
        └──────────────────────────────────┘
```
**关键代码实现**：
[agent.go:175-209](file:///D:\code\github\bkeadm\pkg\agent\agent.go#L175-L209) - Info 函数实现：
```go
func (op *Options) Info() {
    ns := strings.Split(op.Args[0], "/")
    if len(ns) < nsNamePartsCount {
        log.Error("invalid argument format, expected namespace/name")
        return
    }
    var workloadUnstructured *unstructured.Unstructured
    var err error
    dynamicClient := global.K8s.GetDynamicClient()

    workloadUnstructured, err = dynamicClient.Resource(gvr).Namespace(ns[0]).Get(context.TODO(), ns[1], metav1.GetOptions{})
    if err != nil {
        log.Error(err.Error())
        return
    }
    commands := &agentv1beta1.Command{}
    err = runtime.DefaultUnstructuredConverter.FromUnstructured(workloadUnstructured.UnstructuredContent(), commands)
    if err != nil {
        log.Error(err.Error())
        return
    }

    const tabPadding = 2
    for node, v := range commands.Status {
        for _, c := range v.Conditions {
            w := tabwriter.NewWriter(os.Stdout, 0, 0, tabPadding, ' ', 0)
            fmt.Fprintln(w, "name\tsuspend\tnode\tID\tSTATUS")
            fmt.Fprintf(w, "%s/%s\t%s\t%s\t%s\t%s\n",
                commands.Namespace, commands.Name,
                strconv.FormatBool(commands.Spec.Suspend),
                node,
                c.ID,
                string(c.Status),
            )
            err = w.Flush()

            output := ""
            if len(c.StdOut) > 0 {
                output = strings.Join(c.StdOut, "")
            }
            if len(c.StdErr) > 0 {
                output = strings.Join(c.StdErr, "")
            }
            fmt.Println(output)
        }
    }
}
```
#### 3.5 remove 子命令详细设计
**功能**：删除指定的命令资源

**参数**：
- 位置参数：`namespace/name`（命令的命名空间和名称）

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│            bke command remove ns/name                    │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   解析 namespace/name 参数        │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   删除 Command CR                │
        │   DELETE /apis/.../commands/name  │
        └──────────────────────────────────┘
```
**关键代码实现**：
[agent.go:211-223](file:///D:\code\github\bkeadm\pkg\agent\agent.go#L211-L223) - Remove 函数实现：
```go
func (op *Options) Remove() {
    ns := strings.Split(op.Args[0], "/")
    if len(ns) < nsNamePartsCount {
        log.Error("invalid argument format, expected namespace/name")
        return
    }
    var err error
    dynamicClient := global.K8s.GetDynamicClient()
    err = dynamicClient.Resource(gvr).Namespace(ns[0]).Delete(context.TODO(), ns[1], metav1.DeleteOptions{})
    if err != nil {
        log.Error(err.Error())
        return
    }
}
```
#### 3.6 syncTime 子命令详细设计
**功能**：同步节点时间

**参数**：
- 位置参数：NTP 服务器地址（可选，默认使用配置的 NTP 服务器）

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│          bke command syncTime [ntp-server]               │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   获取 NTP 服务器地址             │
        │   - 参数指定                      │
        │   - 或使用默认值                  │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   执行 ntp.Date() 同步时间        │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   失败重试（最多3次）             │
        │   每次间隔 10 秒                  │
        └──────────────────────────────────┘
```
**关键代码实现**：
[agent.go:225-243](file:///D:\code\github\bkeadm\pkg\agent\agent.go#L225-L243) - SyncTime 函数实现：
```go
func (op *Options) SyncTime() {
    ntpServer := configinit.DefaultNTPServer
    if len(op.Args) >= 1 {
        ntpServer = op.Args[0]
    }
    err := ntp.Date(ntpServer)
    if err != nil {
        for i := 0; i < utils.MinManifestsImageArgs; i++ {
            time.Sleep(utils.ContainerWaitSeconds * time.Second)
            err = ntp.Date(ntpServer)
            if err == nil {
                return
            }
        }
        fmt.Println(fmt.Sprintf("Failed to connect to the ntp server %s", ntpServer))
    }
    return
}
```
### 四、架构设计
#### 4.1 整体架构图
```
┌────────────────────────────────────────────────────────────────┐
│                         bkeadm CLI                              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    bke command                            │  │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ │  │
│  │  │  exec  │ │  list  │ │  info  │ │ remove │ │syncTime│ │  │
│  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘ │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
                              │
                              │ Kubernetes API
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                     Kubernetes Cluster                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                  Command CRD                              │  │
│  │  ┌────────────────────────────────────────────────────┐  │  │
│  │  │  Command CR                                         │  │  │
│  │  │  - metadata: name, namespace                        │  │  │
│  │  │  - spec: suspend, nodeSelector, commands            │  │  │
│  │  │  - status: map[node]NodeCommandStatus               │  │  │
│  │  └────────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────────┘  │
│                                                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    ConfigMap (可选)                       │  │
│  │  - 存储 Shell 脚本文件内容                                 │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
                              │
                              │ Watch & Execute
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                      BKEAgent (每个节点)                         │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  1. 监听 Command CR                                       │  │
│  │  2. 匹配节点选择器                                         │  │
│  │  3. 执行命令                                              │  │
│  │  4. 回写执行结果到 Status                                  │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```
#### 4.2 命令执行流程时序图
```
┌─────────┐     ┌─────────┐     ┌──────────┐     ┌──────────┐
│  User   │     │ bkeadm  │     │   K8s    │     │ BKEAgent │
└────┬────┘     └────┬────┘     └────┬─────┘     └────┬─────┘
     │               │               │                │
     │ bke command   │               │                │
     │ exec --nodes  │               │                │
     │──────────────>│               │                │
     │               │               │                │
     │               │ Create Command│                │
     │               │ CR            │                │
     │               │──────────────>│                │
     │               │               │                │
     │               │               │ Watch Command  │
     │               │               │ CR             │
     │               │               │<───────────────│
     │               │               │                │
     │               │               │ Match Node     │
     │               │               │ Selector       │
     │               │               │────────────────>│
     │               │               │                │
     │               │               │                │ Execute
     │               │               │                │ Command
     │               │               │                │
     │               │               │ Update Status  │
     │               │               │<───────────────│
     │               │               │                │
     │ Command sent  │               │                │
     │<──────────────│               │                │
     │               │               │                │
     │ bke command   │               │                │
     │ info ns/name  │               │                │
     │──────────────>│               │                │
     │               │               │                │
     │               │ Get Command   │                │
     │               │ CR            │                │
     │               │──────────────>│                │
     │               │               │                │
     │               │ Return Status │                │
     │               │<──────────────│                │
     │               │               │                │
     │ Show Result   │               │                │
     │<──────────────│               │                │
     │               │               │                │
```
### 五、核心特性
#### 5.1 命令类型支持
**1. Shell 命令类型**
- 直接执行 Shell 命令
- 通过 `--command` 参数指定
- 存储在 `Command.Spec.Commands[].Command` 字段

**2. Kubernetes 命令类型**
- 从 ConfigMap 读取脚本并执行
- 通过 `--file` 参数指定脚本文件
- 自动创建 ConfigMap 存储脚本内容
- 命令格式：`configmap:namespace/name:rx:shell`
#### 5.2 节点选择器
节点选择器采用 Label Selector 机制：
- 将节点 IP 或主机名作为 Label
- 支持多个节点（逗号分隔）
- Agent 通过匹配 Label 决定是否执行
#### 5.3 命令状态管理
命令状态通过 CR 的 Status 字段管理：
- **Phase**：执行阶段
- **Status**：执行状态
- **LastStartTime**：最后开始时间
- **CompletionTime**：完成时间
- **Conditions**：执行条件，包含标准输出和标准错误
### 六、使用示例
#### 6.1 执行简单命令
```bash
# 在单个节点上执行命令
bke command exec --nodes 192.168.1.100 --command "touch /tmp/test.txt"

# 在多个节点上执行命令
bke command exec --nodes 192.168.1.100,192.168.1.101 --command "systemctl restart docker"
```
#### 6.2 执行脚本文件
```bash
# 在节点上执行脚本文件
bke command exec --nodes 192.168.1.100 -f /path/to/script.sh
```
#### 6.3 查看命令状态
```bash
# 列出所有命令
bke command list

# 查看特定命令的执行结果
bke command info default/202504011234
```
#### 6.4 删除命令
```bash
# 删除命令资源
bke command remove default/202504011234
```
#### 6.5 同步时间
```bash
# 使用默认 NTP 服务器同步时间
bke command syncTime

# 使用指定 NTP 服务器
bke command syncTime 192.168.24.25:123
```
### 七、设计优势
#### 7.1 声明式管理
- 命令作为 Kubernetes 资源管理
- 支持版本控制和审计
- 状态可追溯
#### 7.2 分布式执行
- 通过节点选择器实现多节点并行执行
- Agent 模式降低中心节点负载
- 执行状态实时反馈
#### 7.3 灵活性
- 支持单行命令和脚本文件
- 支持多种命令类型
- 可扩展的命令框架
#### 7.4 可靠性
- 命令执行状态持久化
- 支持失败重试
- 执行结果可查询
### 八、总结
`bke command` 是一个基于 Kubernetes CRD 的远程命令执行框架，通过声明式 API 管理命令的生命周期。其核心设计思想是将命令抽象为 Kubernetes 资源，利用 BKEAgent 在节点上执行命令，并通过 Status 字段反馈执行结果。这种设计实现了命令的分布式执行、状态可追溯和灵活管理，是 BKE 集群运维的重要工具。

      
# `bke registry` 的设计思路与详细设计
## bke registry 设计思路与详细设计
### 一、设计思路
`bke registry` 是一个功能完善的容器镜像仓库管理工具，其核心设计思路是：
#### 1. 多种镜像同步方式
- **sync**：基于数据块的镜像同步（无需本地存储）
- **transfer**：基于 docker pull/push 的镜像迁移（需要本地存储）
- **patch**：增量包镜像同步
#### 2. 多架构镜像支持
- 自动识别和处理多架构镜像
- 支持创建多架构 manifest
- 支持按架构同步镜像
#### 3. 镜像仓库管理
- 查看仓库镜像列表
- 检查镜像详细信息
- 删除镜像
- 下载镜像内文件
#### 4. 底层技术栈
- 使用 `containers/image` 库进行镜像操作
- 支持多种传输协议
- 支持多种压缩格式
### 二、命令结构
```
bke registry
├── sync        # 基于数据块的镜像同步
├── transfer    # 基于 docker pull/push 的镜像迁移
├── list-tags   # 列出镜像所有标签
├── inspect     # 检查镜像详细信息
├── manifests   # 创建多架构镜像
├── delete      # 删除镜像
├── view        # 查看仓库视图
├── patch       # 增量包镜像同步
└── download    # 下载镜像内文件
```
### 三、详细设计
#### 3.1 sync 子命令详细设计
**功能**：在两个镜像仓库之间通过数据块复制方式同步镜像

**参数**：
- `--source`：源镜像地址（必填）
- `--target`：目标镜像地址（必填）
- `--file, -f`：镜像列表文件
- `--multi-arch`：同步多架构镜像
- `--arch`：指定同步的架构（amd64/arm64）
- `--src-tls-verify`：验证源仓库 TLS 证书
- `--dest-tls-verify`：验证目标仓库 TLS 证书
- `--sync-repo`：同步整个仓库

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│                    bke registry sync                    │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   判断同步模式                    │
        │   - sync-repo: 整库同步           │
        │   - file: 批量同步                │
        │   - 单镜像同步                    │
        └──────────────────────────────────┘
                           │
           ┌───────────────┼───────────────┐
           ▼               ▼               ▼
    ┌────────────┐  ┌────────────┐  ┌────────────┐
    │ syncRepo   │  │ 批量同步    │  │ 单镜像同步  │
    └────────────┘  └────────────┘  └────────────┘
           │               │               │
           └───────────────┼───────────────┘
                           ▼
        ┌──────────────────────────────────┐
        │   判断架构模式                     │
        │   - multi-arch: 多架构同步        │
        │   - 单架构同步                    │
        └──────────────────────────────────┘
                           │
           ┌───────────────┴───────────────┐
           ▼                               ▼
    ┌─────────────────┐          ┌─────────────────┐
    │ syncMultiArch   │          │ syncSingleArch  │
    │ RepoImage       │          │ RepoImage       │
    └─────────────────┘          └─────────────────┘
           │                               │
           │                               ▼
           │                    ┌─────────────────────┐
           │                    │ CopyRegistry        │
           │                    │ (containers/image)  │
           │                    └─────────────────────┘
           ▼
    ┌──────────────────────────────────────┐
    │ 1. 尝试直接同步多架构镜像              │
    │ 2. 失败则同步各架构镜像                │
    │ 3. 创建多架构 manifest                │
    └──────────────────────────────────────┘
```
**关键代码实现**：
[registry.go:118-147](file:///D:\code\github\bkeadm\pkg\registry\registry.go#L118-L147) - Sync 函数：
```go
func (op *Options) Sync() {
    if op.SyncRepo {
        newOp := *op
        if err := syncRepo(newOp); err != nil {
            log.BKEFormat(log.ERROR, fmt.Sprintf("Sync repo %s to %s error %s", op.Source, op.Target, err.Error()))
            return
        }
        log.BKEFormat(log.INFO, fmt.Sprintf("Sync repo %s to %s success", op.Source, op.Target))
        return
    }

    archs := op.getArchList()

    if len(op.File) == 0 {
        if err := syncRepoImage(*op, archs); err != nil {
            log.Error(err.Error())
        }
        return
    }

    imageList, err := readImageListFromFile(op.File)
    if err != nil {
        log.Error(err.Error())
        return
    }

    sourceAddress := ensureTrailingSlash(op.Source)
    targetAddress := ensureTrailingSlash(op.Target)

    for _, image := range imageList {
        newOp := *op
        newOp.Source = sourceAddress + image
        newOp.Target = targetAddress + image
        if err := syncRepoImage(newOp, archs); err != nil {
            log.Error(err.Error())
            return
        }
    }
}
```
[registry.go:326-368](file:///D:\code\github\bkeadm\pkg\registry\registry.go#L326-L368) - CopyRegistry 函数：
```go
func CopyRegistry(op Options) error {
    imageAddress, targetAddress := normalizeImageAddresses(op.Source, op.Target)
    srcRef, destRef, err := parseImageReferences(imageAddress, targetAddress, op.Source)
    if err != nil {
        return err
    }

    policyContext, err := createPolicyContext()
    if err != nil {
        return err
    }

    sourceCtx, destinationCtx, err := createSystemContexts(op)
    if err != nil {
        return err
    }

    return executeCopyWithRetry(copyParams{
        srcRef:         srcRef,
        destRef:        destRef,
        policyContext:  policyContext,
        sourceCtx:      sourceCtx,
        destinationCtx: destinationCtx,
        imageAddress:   imageAddress,
        targetAddress:  targetAddress,
        multiArch:      op.MultiArch,
    })
}
```
#### 3.2 transfer 子命令详细设计
**功能**：通过 docker pull/push 方式迁移镜像

**参数**：
- `--source`：源镜像仓库地址（必填）
- `--target`：目标镜像仓库地址（必填）
- `--image`：镜像名称
- `--file, -f`：镜像列表文件
- `--arch`：架构列表（默认当前系统架构）

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│                  bke registry transfer                   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   检查 Docker 环境                │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   读取镜像列表                     │
        │   - 单镜像                        │
        │   - 文件批量                      │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   判断架构模式                     │
        │   - 单架构                        │
        │   - 多架构                        │
        └──────────────────────────────────┘
                           │
           ┌───────────────┴───────────────┐
           ▼                               ▼
    ┌─────────────────┐          ┌─────────────────┐
    │ syncSingleArch  │          │ syncMultiArch   │
    │ Image           │          │ Image           │
    └─────────────────┘          └─────────────────┘
           │                               │
           ▼                               ▼
    ┌─────────────────┐          ┌─────────────────┐
    │ 1. Pull 镜像    │          │ 1. Pull 各架构  │
    │ 2. Tag 镜像     │          │    镜像         │
    │ 3. Push 镜像    │          │ 2. Tag 镜像     │
    │ 4. 清理本地镜像  │          │ 3. Push 镜像    │
    └─────────────────┘          │ 4. 创建 Manifest│
                                  │ 5. Push Manifest│
                                  │ 6. 清理本地镜像  │
                                  └─────────────────┘
```
**关键代码实现**：
[migrate.go:23-56](file:///D:\code\github\bkeadm\pkg\registry\migrate.go#L23-L56) - MigrateImage 函数：
```go
func (op *Options) MigrateImage() {
    if !infrastructure.IsDocker() {
        log.BKEFormat(log.ERROR, "This migrate instruction only supports running in docker environment.")
        return
    }

    var imageList []string
    if len(op.Image) > 0 {
        imageList = append(imageList, op.Image)
    }
    if len(op.File) > 0 {
        fi, err := os.Open(op.File)
        if err != nil {
            log.Error(err)
            return
        }
        buf := bufio.NewScanner(fi)
        for {
            if !buf.Scan() {
                break
            }
            if len(buf.Text()) == 0 {
                continue
            }
            imageList = append(imageList, buf.Text())
        }
    }
    if len(op.Arch) == 0 {
        op.Arch = runtime.GOARCH
    }
    archs := strings.Split(op.Arch, ",")
    for _, image := range imageList {
        sourceImage := op.Source + "/" + image
        sourceImage = strings.ReplaceAll(sourceImage, "//", "/")
        targetImage := op.Target + "/" + image
        targetImage = strings.ReplaceAll(targetImage, "//", "/")
        err := syncImage(sourceImage, targetImage, archs)
        if err != nil {
            log.Error(err)
            return
        }
    }
    log.BKEFormat(log.NIL, "Image migration completed. ")
    cleanBuildImage()
}
```
#### 3.3 inspect 子命令详细设计
**功能**：检查镜像详细信息

**参数**：
- 位置参数：镜像地址
- `--dest-tls-verify`：验证 TLS 证书

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│          bke registry inspect image:tag                  │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   创建镜像源                      │
        │   - 解析镜像引用                  │
        │   - 创建 SystemContext           │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   获取 Manifest                  │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   判断 Manifest 类型              │
        │   - 多架构: 输出 Manifest List   │
        │   - 单架构: 输出 OCI Config      │
        └──────────────────────────────────┘
```
**关键代码实现**：
[manifests.go:104-133](file:///D:\code\github\bkeadm\pkg\registry\manifests.go#L104-L133) - Inspect 函数：
```go
func (op *Options) Inspect() {
    ctx := context.Background()

    imageSource, destinationCtx, err := op.createInspectImageSource(ctx)
    if err != nil {
        log.Error(err)
        return
    }
    defer func() { _ = imageSource.Close() }()

    rawManifest, _, err := imageSource.GetManifest(ctx, nil)
    if err != nil {
        log.Error(err)
        return
    }

    handled, err := op.handleMultiArchManifest(rawManifest)
    if err != nil {
        log.Error(err)
        return
    }
    if handled {
        return
    }

    op.printSingleArchConfig(ctx, destinationCtx, imageSource)
}
```
#### 3.4 manifests 子命令详细设计
**功能**：创建多架构镜像 manifest

**参数**：
- `--image`：目标镜像名称（必填）
- 位置参数：各架构镜像列表（至少 2 个）

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│          bke registry manifests --image=name             │
│                 img-amd64 img-arm64                      │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   解析各架构镜像信息               │
        │   - 提取架构后缀                  │
        │   - 构建 ImageArch 列表          │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   获取各镜像的 Digest 和 Size     │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   构建 DockerV2List Manifest     │
        │   - SchemaVersion: 2             │
        │   - MediaType: DockerV2List      │
        │   - Manifests: []Manifest        │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   推送 Manifest 到仓库            │
        └──────────────────────────────────┘
```
**关键代码实现**：
[manifests.go:261-288](file:///D:\code\github\bkeadm\pkg\registry\manifests.go#L261-L288) - CreateMultiArchImage 函数：
```go
func CreateMultiArchImage(img []ImageArch, target string) error {
    dvl := DockerV2List{
        SchemaVersion: 2,
        MediaType:     DockerV2ListMediaType,
        Manifests:     []Manifest{},
    }
    preferOCIIndex := false
    for _, im := range img {
        sha, size, childMediaType, err := getImageSha(im.Name)
        if err != nil {
            return err
        }
        if strings.HasPrefix(childMediaType, "application/vnd.oci.") {
            preferOCIIndex = true
        }
        if childMediaType == "" {
            childMediaType = DockerV2Schema2MediaType
        }
        dvl.Manifests = append(dvl.Manifests, Manifest{
            MediaType: childMediaType,
            Digest:    sha,
            Size:      size,
            Platform: struct {
                Architecture string `json:"architecture"`
                OS           string `json:"os"`
            }{
                Architecture: im.Architecture,
                OS:           im.OS,
            },
        })
    }
    if preferOCIIndex {
        dvl.MediaType = DockerV1ListMediaType
    }

    dvlByte, err := json.MarshalIndent(dvl, "", "    ")
    if err != nil {
        return err
    }
    err = putManifests(string(dvlByte), target)
    if err != nil {
        return err
    }
    return nil
}
```
#### 3.5 view 子命令详细设计
**功能**：查看仓库视图，列出镜像信息

**参数**：
- 位置参数：仓库地址
- `--prefix`：镜像路径前缀过滤
- `--tags`：显示的标签数量（默认 10）
- `--export`：导出镜像列表到文件

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│            bke registry view registry:port               │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   初始化 HTTP 客户端              │
        │   - 支持 HTTPS/HTTP              │
        │   - 跳过 TLS 验证                │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   获取仓库列表                    │
        │   GET /v2/_catalog               │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   遍历每个镜像                    │
        │   - 获取标签列表                  │
        │   - 获取 Manifest 信息            │
        │   - 提取架构信息                  │
        │   - 计算镜像大小                  │
        │   - 获取创建时间                  │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   输出结果                        │
        │   - 表格显示                      │
        │   - 或导出到文件                  │
        └──────────────────────────────────┘
```
**关键代码实现**：
[view.go:22-43](file:///D:\code\github\bkeadm\pkg\registry\view.go#L22-L43) - View 函数：
```go
func (op *Options) View() {
    httpClient, baseURL := initHTTPClient(op.Args[0])
    repos := fetchRepositories(httpClient, fmt.Sprintf("%s/v2/_catalog?n=10000", baseURL))
    if repos == nil {
        return
    } else if repos.Repositories == nil || len(repos.Repositories) == 0 {
        fmt.Println("no repositories found")
        return
    }

    headers := []string{"IMAGE", "TAGS", "ARCHITECTURE", "CREATE TIME", "SIZE"}
    rows, exportList := processRepositories(httpClient, baseURL, repos.Repositories, op.Prefix, op.Tags)
    outputResults(op.Export, exportList, headers, rows)
}
```
#### 3.6 delete 子命令详细设计
**功能**：删除指定镜像

**参数**：
- 位置参数：镜像地址
- `--dest-tls-verify`：验证 TLS 证书

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│          bke registry delete image:tag                   │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   解析镜像引用                    │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   创建 SystemContext             │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   删除镜像                        │
        │   ref.DeleteImage()              │
        └──────────────────────────────────┘
```
**关键代码实现**：
[delete.go:21-64](file:///D:\code\github\bkeadm\pkg\registry\delete.go#L21-L64) - Delete 函数：
```go
func (op *Options) Delete() {
    if len(op.Args) != 1 {
        log.Error("only one mirror can be deleted at a time")
        return
    }
    imageName := op.Args[0]
    if !strings.HasPrefix(imageName, "docker://") {
        imageName = fmt.Sprintf("docker://%s", imageName)
    }
    ref, err := alltransports.ParseImageName(imageName)
    if err != nil {
        log.Error(fmt.Sprintf("invalid image name %s: %v", imageName, err))
        return
    }

    destinationCtx, err := newSystemContext()
    if err != nil {
        log.Error(err)
        return
    }
    if op.DestTLSVerify {
        destinationCtx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(true)
        destinationCtx.DockerDaemonInsecureSkipTLSVerify = true
    }

    ctx := context.Background()
    err = retry.IfNecessary(ctx, func() error {
        return ref.DeleteImage(ctx, destinationCtx)
    }, &retry.Options{
        MaxRetry: 0,
        Delay:    0,
    })
    if err != nil {
        log.Error(err.Error())
        return
    }
    return
}
```
#### 3.7 download 子命令详细设计
**功能**：下载镜像内的文件（无需拉取整个镜像）

**参数**：
- `--image`：镜像地址（必填）
- `--downloadInImageFile, -f`：要下载的文件（必填）
- `--downloadToDir, -d`：下载目录（默认当前目录）
- `--username, -u`：仓库用户名
- `--password, -p`：仓库密码
- `--certDir`：证书目录
- `--src-tls-verify`：验证 TLS 证书

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│        bke registry download --image name                │
│              -f file1,file2 -d /path                     │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   创建镜像源                      │
        │   - 解析镜像引用                  │
        │   - 创建 SystemContext           │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   获取镜像层信息                  │
        │   - LayerInfos                   │
        │   - 反向遍历（从最新层开始）       │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   遍历每一层                      │
        │   - 获取层 Blob                  │
        │   - 解压缩        │
        │   - 解析 TAR 文件                │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   查找目标文件                    │
        │   - 匹配文件名                    │
        │   - 跳过 whiteout 文件            │
        │   - 跳过目录                      │
        └──────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   提取文件到目标目录               │
        └──────────────────────────────────┘
```
**关键代码实现**：
[download.go:54-98](file:///D:\code\github\bkeadm\pkg\registry\download.go#L54-L98) - Download 函数：
```go
func (od *OptionsDownload) Download() error {
    if err := od.ensureDownloadDir(); err != nil {
        return err
    }
    downloadFileMap := od.buildDownloadFileMap()
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    imageSource, registrySrcCtx, err := od.setupImageSource(ctx)
    if err != nil {
        return err
    }
    defer closeImage(imageSource)

    if err := od.downloadFilesFromLayers(ctx, imageSource, registrySrcCtx, downloadFileMap); err != nil {
        return err
    }

    od.logDownloadResults(downloadFileMap)
    return nil
}
```
[download.go:123-150](file:///D:\code\github\bkeadm\pkg\registry\download.go#L123-L150) - downloadFilesFromLayers 函数：
```go
func (od *OptionsDownload) downloadFilesFromLayers(ctx context.Context, imageSource types.ImageSource,
    registrySrcCtx *types.SystemContext, downloadFileMap map[string]string) error {
    imgCloser, err := image.FromSource(ctx, registrySrcCtx, imageSource)
    if err != nil {
        klog.Errorf("Error retrieving image: %v", err)
        return errors.Wrap(err, "Error retrieving image")
    }
    defer imgCloser.Close()

    cache := blobinfocache.DefaultCache(registrySrcCtx)
    layers := imgCloser.LayerInfos()

    for _, layer := range reverseLayers(layers) {
        log.Debugf("Processing layer %+v", layer)
        config := &LayerProcessConfig{
            SystemContext:   registrySrcCtx,
            ImageSource:     imageSource,
            Layer:           layer,
            DownloadToDir:   od.DownloadToDir,
            DownloadFileMap: downloadFileMap,
            Cache:           cache,
        }
        err = processLayer(ctx, config)
        if found(downloadFileMap) {
            break
        }
        if err != nil {
            continue
        }
    }
    return nil
}
```
#### 3.8 patch 子命令详细设计
**功能**：增量包镜像同步

**参数**：
- `--source`：增量包目录
- `--target`：目标仓库地址

**执行流程**：
```
┌─────────────────────────────────────────────────────────┐
│      bke registry patch --source /path --target repo     │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   调用 build.SpecificSync        │
        │   处理增量包同步                  │
        └──────────────────────────────────┘
```
### 四、架构设计
#### 4.1 整体架构图
```
┌────────────────────────────────────────────────────────────────┐
│                         bkeadm CLI                              │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                    bke registry                           │  │
│  │  ┌──────┐ ┌─────────┐ ┌───────┐ ┌──────┐ ┌────────┐    │  │
│  │  │ sync │ │transfer │ │inspect│ │ view │ │download│    │  │
│  │  └──────┘ └─────────┘ └───────┘ └──────┘ └────────┘    │  │
│  │  ┌──────────┐ ┌─────────┐ ┌────────┐ ┌───────┐         │  │
│  │  │ manifests│ │list-tags│ │ delete │ │ patch │         │  │
│  │  └──────────┘ └─────────┘ └────────┘ └───────┘         │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
                              │
                              │ containers/image API
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                    containers/image 库                          │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  - Image Reference 解析                                   │  │
│  │  - Manifest 操作                                          │  │
│  │  - Blob 传输                                              │  │
│  │  - 签名验证                                                │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
                              │
                              │ Registry API V2
                              ▼
┌────────────────────────────────────────────────────────────────┐
│                    Container Registry                           │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │  Docker Registry / Harbor / 其他兼容仓库                  │  │
│  └──────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────┘
```
#### 4.2 镜像同步流程对比
**sync (数据块同步)**：
```
源仓库 ──(Blob Transfer)──> 目标仓库
         无需本地存储
         直接传输数据块
```
**transfer (Docker 迁移)**：
```
源仓库 ──(Pull)──> 本地存储 ──(Push)──> 目标仓库
                   需要磁盘空间
                   支持 Docker manifest
```
#### 4.3 多架构镜像处理流程
```
┌─────────────────────────────────────────────────────────┐
│              多架构镜像同步流程                           │
└─────────────────────────────────────────────────────────┘
                           │
                           ▼
        ┌──────────────────────────────────┐
        │   尝试直接同步多架构镜像           │
        │   CopyRegistry(multi-arch=true)  │
        └──────────────────────────────────┘
                           │
                   ┌───────┴───────┐
                   │               │
                成功             失败
                   │               │
                   ▼               ▼
                完成      ┌─────────────────┐
                          │ 分别同步各架构    │
                          │ busybox:amd64   │
                          │ busybox:arm64   │
                          └─────────────────┘
                                   │
                                   ▼
                          ┌─────────────────┐
                          │ 创建 Manifest   │
                          │ busybox:latest  │
                          │ - amd64 digest  │
                          │ - arm64 digest  │
                          └─────────────────┘
                                   │
                                   ▼
                          ┌─────────────────┐
                          │ 推送 Manifest   │
                          └─────────────────┘
```
### 五、核心特性
#### 5.1 数据块同步
**优势**：
- 无需本地存储空间
- 直接传输镜像层数据
- 支持多架构镜像
- 支持重试机制

**实现**：
- 使用 `containers/image` 库的 `copy.Image` API
- 支持 `CopySystemImage` 和 `CopyAllImages` 模式
- 内置重试机制（最多 9 次）
#### 5.2 多架构镜像支持
**支持的架构**：
- amd64
- arm64

**Manifest 类型**：
- Docker V2 Schema 2 List
- OCI Image Index

**自动识别**：
- 自动检测子镜像的 MediaType
- 优先使用 OCI Index 格式
#### 5.3 压缩格式支持
**支持的格式**：
- gzip
- zstd
- xz
- tar

**自动检测**：
- 通过魔数识别压缩格式
- 自动解压缩处理
#### 5.4 安全特性
**TLS 验证**：
- 支持跳过 TLS 验证
- 支持自定义证书目录

**认证支持**：
- 用户名/密码认证
- Docker 配置文件认证
### 六、使用示例
#### 6.1 镜像同步
```bash
# 同步单个镜像
bke registry sync --source docker.io/library/busybox:1.35 \
                  --target 127.0.0.1:40443/library/busybox:1.35

# 同步多架构镜像
bke registry sync --source docker.io/library/busybox:1.35 \
                  --target 127.0.0.1:40443/library/busybox:1.35 \
                  --multi-arch

# 批量同步镜像
bke registry sync --source docker.io/library \
                  --target 127.0.0.1:40443/library \
                  -f image-list.txt

# 同步整个仓库
bke registry sync --source docker.io/library \
                  --target 127.0.0.1:40443/library \
                  --sync-repo
```
#### 6.2 镜像迁移
```bash
# 迁移单个镜像
bke registry transfer --source docker.io/library \
                      --image busybox:1.28 \
                      --target registry.cloud.com/k8s

# 迁移多架构镜像
bke registry transfer --source docker.io/library \
                      --image busybox:1.28 \
                      --target registry.cloud.com/k8s \
                      --arch amd64,arm64

# 批量迁移镜像
bke registry transfer --source docker.io/library \
                      --target registry.cloud.com/k8s \
                      -f image-list.txt
```
#### 6.3 镜像检查
```bash
# 检查镜像信息
bke registry inspect registry.bocloud.com/kubernetes/pause:3.8

# 跳过 TLS 验证
bke registry inspect --dest-tls-verify registry.bocloud.com/kubernetes/pause:3.8
```

#### 6.4 创建多架构镜像

```bash
# 创建多架构 manifest
bke registry manifests --image=127.0.0.1:40443/library/busybox:1.35 \
                       127.0.0.1:40443/library/busybox:1.35-amd64 \
                       127.0.0.1:40443/library/busybox:1.35-arm64
```
#### 6.5 查看仓库
```bash
# 查看仓库镜像
bke registry view 192.168.2.111:40443

# 按前缀过滤
bke registry view 192.168.2.111:40443 --prefix kubernetes/kube

# 导出镜像列表
bke registry view 192.168.2.111:40443 --export
```
#### 6.6 下载镜像内文件
```bash
# 下载单个文件
bke registry download --image repository/kubectl:v1.23.17 \
                      -f /opt/bocloud/kubectl \
                      -d /opt

# 下载多个文件
bke registry download --image repository/kubectl:v1.23.17 \
                      -f kubectl,kubectl-ns \
                      -d /opt
```
#### 6.7 删除镜像
```bash
# 删除镜像
bke registry delete 192.168.2.111:40443/library/busybox:1.35
```
### 七、设计优势
#### 7.1 高效同步
- 数据块直接传输，无需本地存储
- 支持批量同步和整库同步
- 内置重试机制保证可靠性
#### 7.2 多架构支持
- 自动识别和处理多架构镜像
- 支持创建多架构 manifest
- 兼容 Docker 和 OCI 格式
#### 7.3 灵活性
- 支持多种传输方式
- 支持多种压缩格式
- 支持自定义认证和证书
#### 7.4 易用性
- 统一的命令行接口
- 详细的日志输出
- 支持导出和导入
### 八、总结
`bke registry` 是一个功能完善的容器镜像仓库管理工具，提供了镜像同步、迁移、检查、删除等完整功能。其核心设计思想是通过 `containers/image` 库实现高效的镜像操作，支持多架构镜像管理，并提供多种传输方式以适应不同场景。该工具是 BKE 集群镜像管理的重要组成部分，为离线环境和在线环境的镜像同步提供了可靠的解决方案。

