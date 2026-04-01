      
       
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


