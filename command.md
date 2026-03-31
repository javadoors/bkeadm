      
       
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

这种设计确保了初始化过程的**可靠性**、**灵活性**和**可维护性**，同时支持多种部署场景。
        
