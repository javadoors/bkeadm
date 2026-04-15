# bkeadm 版本配置管理分析
## bkeadm 版本配置管理分析
### 一、四个配置文件的定位与关系
bkeadm 的 `assets/` 目录下有 4 个 YAML 配置文件，它们都是 **`BuildConfig` 结构**的实例，用于 `bke build` 系列命令的构建输入：

| 配置文件 | 用途 | 目标场景 | 镜像源 |
|---------|------|---------|--------|
| `offline-artifacts.yaml` | **完整离线安装包**构建配置 | `bke build -f offline-artifacts.yaml` | hub.oepkgs.net + cr.openfuyao.cn |
| `k8s-offline-artifacts.yaml` | **K8s 专用离线包**构建配置 | `bke build -f k8s-offline-artifacts.yaml` | cr.openfuyao.cn |
| `online-artifacts.yaml` | **在线安装镜像**构建配置 | `bke build online-image -f online-artifacts.yaml` | cr.openfuyao.cn |
| `offline-component.yaml` | **扩展组件离线包**构建配置 | `bke build patch -f offline-component.yaml` | cr.openfuyao.cn + docker.io |
### 二、配置文件的数据结构（BuildConfig）
所有配置文件共享同一个 [BuildConfig](file:///d:/code/github/bkeadm/pkg/build/config.go#L27-L40) 结构：
```go
type BuildConfig struct {
    Registry          registry `yaml:"registry"`          // registry 基础镜像地址和架构
    OpenFuyaoVersion  string   `yaml:"openFuyaoVersion"`  // openFuyao 版本号
    KubernetesVersion string   `yaml:"kubernetesVersion"` // K8s 版本号
    EtcdVersion       string   `yaml:"etcdVersion"`       // etcd 版本号
    ContainerdVersion string   `yaml:"containerdVersion"` // containerd 版本号
    Repos             []Repo   `yaml:"repos"`             // 镜像仓库列表
    Rpms              []Rpm    `yaml:"rpms"`              // RPM 包列表
    Debs              []Deb    `yaml:"debs"`              // DEB 包列表
    Files             []File   `yaml:"files"`             // 二进制文件下载列表
    Patches           []File   `yaml:"patches"`           // 版本补丁配置文件列表
    Charts            []File   `yaml:"charts"`            // Helm Chart 列表
}
```
### 三、各配置文件在构建流程中的使用方式
#### 1. `offline-artifacts.yaml` → `bke build`（完整离线包）
**使用命令**：`bke build -f offline-artifacts.yaml -t bke.tar.gz`

**流程**（[build.go](file:///d:/code/github/bkeadm/pkg/build/build.go#L37-L56)）：
1. **加载配置** → `loadAndVerifyBuildConfig()` 解析 YAML 为 `BuildConfig`
2. **并行收集**：
   - **RPM + 二进制文件**：`buildRpms()` → 下载 `files` 中定义的 kubelet/kubectl/containerd/cni 等 + `rpms` 中的系统包
   - **镜像同步**：`buildRegistry()` + `syncRepo()` → 按 `repos` 定义从 sourceRepo 同步到本地 registry
3. **打包**：将镜像数据 + 源数据压缩为 `bke.tar.gz`

**关键特点**：
- `repos` 包含 **全部组件镜像**（K3s 基础服务 + K8s 组件 + openFuyao 组件 + 监控组件）
- `files` 包含 **全部二进制文件**（kubelet、kubectl、containerd、cni、helm、cfssl 等）
- `patches` 指向版本配置下载地址（`VersionConfig-latest.yaml`）
- `charts` 包含所有 Helm Chart
#### 2. `k8s-offline-artifacts.yaml` → `bke build`（K8s 专用离线包）
**使用命令**：`bke build -f k8s-offline-artifacts.yaml -t bke-k8s.tar.gz`

与 `offline-artifacts.yaml` 流程相同，但：
- `repos` 只包含 **K3s 基础服务 + K8s 组件 + openFuyao 核心组件**（无监控、Harbor 等）
- `files` 中 kubelet/kubectl 版本不同（v1.28.8/v1.29.1 vs v1.34.3-of.1）
- `charts` 只包含核心 Chart（oauth、console、installer 等）
#### 3. `online-artifacts.yaml` → `bke build online-image`（在线安装镜像）
**使用命令**：`bke build online-image -f online-artifacts.yaml -t cr.openfuyao.cn/openfuyao/bke-online-installed:latest`

**流程**（[onlineimage.go](file:///d:/code/github/bkeadm/pkg/build/onlineimage.go#L35-L68)）：
1. **加载配置** → 解析 YAML
2. **收集源文件** → `buildRpms()` 下载 `files` 中定义的二进制文件，打包为 `source.tar.gz`
3. **构建 Docker 镜像** → 将 `source.tar.gz` 打包进 Docker 镜像（`FROM scratch` + `COPY source.tar.gz`）
4. **推送镜像** → `docker push` 或 `docker buildx build --push`

**关键特点**：
- 只包含 **registry 镜像 + 基础二进制文件**（无 openFuyao 组件镜像）
- `repos` 只有一个 registry 镜像条目
- `files` 包含 kubelet、kubectl、containerd、cni 等基础文件
- **不含 charts 和 patches**
#### 4. `offline-component.yaml` → `bke build patch`（扩展组件补丁包）
**使用命令**：`bke build patch -f offline-component.yaml -t bke-patch-logging.tar.gz`

**流程**（[patch.go](file:///d:/code/github/bkeadm/pkg/build/patch.go#L33-L52)）：
1. **加载配置** → 解析 YAML
2. **并行收集**：
   - `collectHostFiles()` → 下载 `files` 中的文件
   - `collectPatchFiles()` → 下载 `patches` 中的版本配置
   - `collectChartFiles()` → 下载 `charts` 中的 Helm Chart
   - `collectImages()` → 按 `repos` 同步镜像
3. **打包** → 压缩为补丁包

**关键特点**：
- 只包含 **特定扩展组件**（如 logging 组件）
- 支持 `oci` 和 `registry` 两种镜像同步策略
- 补丁包结构包含 `volumes/patches/` 目录存放版本配置
### 四、版本配置（VersionConfig / Patches）的使用链路
这是最关键的部分——`Core-VersionConfig` 文件在 bkeadm 中的完整使用链路：
```
┌─────────────────────────────────────────────────────────────────────┐
│  构建阶段 (bke build)                                                │
│                                                                     │
│  offline-artifacts.yaml                                             │
│    patches:                                                         │
│      - address: https://openfuyao.obs.cn-north-4.myhuaweicloud.com │
│            /openFuyao/version-config/                               │
│        files:                                                       │
│          - VersionConfig-latest.yaml                                │
│                          │                                          │
│                          ▼                                          │
│  下载 VersionConfig-latest.yaml → 打包到                             │
│    /bke/volumes/patches/ 目录                                       │
│                          │                                          │
│  最终打包为 bke.tar.gz                                               │
└─────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  初始化阶段 (bke init)                                               │
│                                                                     │
│  解压 bke.tar.gz → patches 文件位于                                  │
│    /bke/mount/source_registry/files/patches/                        │
│                          │                                          │
│                          ▼                                          │
│  offlineGenerateDeployCM()  [离线模式]                               │
│    1. 遍历 patchesDir 下的文件                                       │
│    2. 从文件名提取版本号 (extractVersionFromFilename)                  │
│    3. 匹配 --oFVersion 参数指定的版本                                 │
│    4. 读取 YAML 文件内容                                             │
│    5. 调用 SetPatchConfig() → 写入 K8s ConfigMap                    │
│       命名空间: openfuyao-patch                                      │
│       ConfigMap 名: cm.<version>                                    │
│       Data Key: <version> → VersionConfig YAML 内容                  │
│                                                                     │
│  onlineGenerateDeployCM()  [在线模式]                                │
│    1. 从 --versionUrl 下载 index.yaml                                │
│    2. 解析 index.yaml 获取版本列表                                    │
│    3. 匹配 --oFVersion 指定的版本                                     │
│    4. 下载对应的 VersionConfig YAML 文件                              │
│    5. 同样调用 SetPatchConfig() → 写入 ConfigMap                     │
└─────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  集群部署阶段 (ensureClusterAPI)                                     │
│                                                                     │
│  getClusterAPIVersion()                                             │
│    1. 从 ConfigMap "cm.<oFVersion>" 读取数据                         │
│    2. 反序列化为 BuildConfig 结构                                     │
│    3. 从 BuildConfig.Repos 中查找:                                   │
│       - "bke-manifests" 镜像的 tag → manifestsVersion               │
│       - "cluster-api-provider-bke" 镜像的 tag → providerVersion     │
│    4. 使用这两个版本部署 Cluster API 组件                              │
│                                                                     │
│  ProcessPatchFiles()                                                │
│    1. 遍历 patchesDir 下所有版本配置文件                               │
│    2. 每个版本文件写入一个 ConfigMap                                   │
│    3. 生成 patch.<version> → cm.<version> 的映射                     │
│    4. 映射关系写入 BKECluster 配置的 data 中                          │
└─────────────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────────┐
│  集群创建阶段 (bke cluster create)                                   │
│                                                                     │
│  BKECluster 配置中包含:                                              │
│    patch.<version>: cm.<version>                                    │
│                                                                     │
│  cluster-api-provider-bke 控制器:                                    │
│    读取 ConfigMap 获取版本配置                                        │
│    → 确定各组件的镜像版本                                             │
│    → 执行集群安装部署                                                 │
└─────────────────────────────────────────────────────────────────────┘
```
### 五、VersionConfig 文件的核心作用
`VersionConfig-<version>.yaml`（即 `Core-VersionConfig-v26.03.yaml` 之类）本质上是一个 **`BuildConfig` 结构的 YAML 文件**，其核心作用是：
1. **版本锚定**：定义该版本下所有组件的精确镜像版本（tag），包括：
   - `bke-manifests` 的版本 → Cluster API manifests 部署版本
   - `cluster-api-provider-bke` 的版本 → Provider 部署版本
   - 其他所有组件镜像的版本
2. **ConfigMap 传递**：版本配置通过 ConfigMap 在 bkeadm 和 cluster-api-provider-bke 之间传递：
   - bkeadm `init` 阶段写入 ConfigMap
   - cluster-api-provider-bke 控制器读取 ConfigMap
3. **多版本支持**：patches 目录可包含多个版本的配置文件，支持同一管理集群管理不同版本的工作集群
### 六、四个配置文件的差异对比
| 维度 | offline-artifacts.yaml | k8s-offline-artifacts.yaml | online-artifacts.yaml | offline-component.yaml |
|------|----------------------|--------------------------|---------------------|----------------------|
| **构建命令** | `bke build` | `bke build` | `bke build online-image` | `bke build patch` |
| **镜像源** | hub.oepkgs.net + cr.openfuyao.cn | cr.openfuyao.cn | cr.openfuyao.cn | cr.openfuyao.cn + docker.io |
| **K8s 版本** | v1.34.3-of.1 | v1.28.8 / v1.29.1 | v1.33.1 | - |
| **containerd** | v2.1.1 | v1.7.14 | v2.1.1 | - |
| **组件范围** | 全部（K3s+K8s+openFuyao+监控） | K3s+K8s+openFuyao核心 | 仅registry+基础二进制 | 仅扩展组件（如logging） |
| **含 patches** | ✅ | ❌ | ❌ | ❌ |
| **含 charts** | ✅ 全部 | ✅ 核心 | ❌ | ✅ 扩展组件 |
| **产物类型** | tar.gz 离线包 | tar.gz 离线包 | Docker 镜像 | tar.gz 补丁包 |
| **安装模式** | 离线 | 离线 | 在线 | 补丁增量 |

