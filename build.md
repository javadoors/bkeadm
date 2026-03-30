
# bkeadm Docker 镜像构建设计
## 一、概述
bkeadm 提供两种类型的 Docker 镜像构建：

| 镜像类型 | Dockerfile | 用途 | 大小 |
|----------|------------|------|------|
| **bkeadm 二进制镜像** | `Dockerfile` | 包含 bkeadm CLI 工具 | ~30MB |
| **在线安装镜像** | `Dockerfile.online` | 包含完整离线资源 | ~2GB+ |
## 二、镜像架构
```
┌─────────────────────────────────────────────────────────────────────────────────────┐
│                              bkeadm 镜像构建架构                                     │
├─────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                        镜像类型一: bkeadm 二进制镜像                          │   │
│  │                                                                             │   │
│  │   ┌───────────────┐      ┌───────────────┐      ┌───────────────┐          │   │
│  │   │ Builder Stage │      │ Build Stage   │      │ Release Stage │          │   │
│  │   │ (golang:1.24) │ ───▶ │ 编译 Go 代码   │ ───▶ │ (scratch)     │          │   │
│  │   │               │      │               │      │ 仅含 bkeadm   │          │   │
│  │   └───────────────┘      └───────────────┘      └───────────────┘          │   │
│  │                                                                             │   │
│  │   输出: cr.openfuyao.cn/openfuyao/bkeadm:v1.0.0 (~30MB)                     │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                     │
│  ┌─────────────────────────────────────────────────────────────────────────────┐   │
│  │                        镜像类型二: 在线安装镜像                               │   │
│  │                                                                             │   │
│  │   ┌───────────────┐      ┌───────────────┐      ┌───────────────┐          │   │
│  │   │ Builder Stage │      │ Download Stage│      │ Release Stage │          │   │
│  │   │ (alpine:3.18) │ ───▶ │ 下载所有资源   │ ───▶ │ (scratch)     │          │   │
│  │   │               │      │ 打包 source    │      │ 含 source.tar │          │   │
│  │   └───────────────┘      └───────────────┘      └───────────────┘          │   │
│  │                                                                             │   │
│  │   输出: cr.openfuyao.cn/openfuyao/bke-online:v1.0.0 (~2GB+)                 │   │
│  └─────────────────────────────────────────────────────────────────────────────┘   │
│                                                                                     │
└─────────────────────────────────────────────────────────────────────────────────────┘
```
## 三、bkeadm 二进制镜像设计
### 3.1 Dockerfile 分析
```dockerfile
# syntax=docker/dockerfile:latest

# 构建参数定义
ARG BUILDER=golang
ARG BUILDER_VERSION=1.24.5
ARG BUILDER_IMAGE=cr.openfuyao.cn/openfuyao/builder/$BUILDER:$BUILDER_VERSION

# 版本信息参数
ARG COMMIT
ARG VERSION
ARG TARGETPLATFORM
ARG SOURCE_DATE_EPOCH

# Go 编译参数
ARG GOFLAGS='-tags=osusergo,netgo,remote,exclude_graphdriver_btrfs,btrfs_noversion,exclude_graphdriver_devicemapper,containers_image_openpgp'
ARG GOLDFLAGS="-X gopkg.openfuyao.cn/bkeadm/utils/version.GitCommitID=$COMMIT \
               -X gopkg.openfuyao.cn/bkeadm/utils/version.Version=$VERSION \
               -X gopkg.openfuyao.cn/bkeadm/utils/version.Architecture=$TARGETPLATFORM \
               -X gopkg.openfuyao.cn/bkeadm/utils/version.Timestamp=$SOURCE_DATE_EPOCH"

# 构建阶段
FROM $BUILDER_IMAGE AS build

# 发布阶段 - 使用 scratch 最小化镜像
FROM scratch AS release
COPY --link --from=build --chmod=555 /go/bin/app ./bkeadm
ENTRYPOINT ["/bkeadm"]
```
### 3.2 构建参数说明
| 参数 | 说明 | 示例 |
|------|------|------|
| `BUILDER` | 基础构建镜像 | `golang` |
| `BUILDER_VERSION` | Go 版本 | `1.24.5` |
| `COMMIT` | Git Commit ID | `abc123` |
| `VERSION` | 版本号 | `v1.0.0` |
| `TARGETPLATFORM` | 目标平台 | `linux/amd64` |
| `SOURCE_DATE_EPOCH` | 构建时间戳 | `1704067200` |
### 3.3 Go 编译标签说明
```bash
GOFLAGS='-tags=osusergo,netgo,remote,exclude_graphdriver_btrfs,btrfs_noversion,exclude_graphdriver_devicemapper,containers_image_openpgp'
```
| 标签 | 作用 |
|------|------|
| `osusergo` | 使用纯 Go 实现用户/组查找 |
| `netgo` | 使用纯 Go 网络库 |
| `remote` | 启用远程镜像支持 |
| `exclude_graphdriver_btrfs` | 排除 btrfs 存储驱动 |
| `exclude_graphdriver_devicemapper` | 排除 devicemapper 存储驱动 |
| `containers_image_openpgp` | 使用 OpenPGP 签名验证 |
### 3.4 版本信息注入
```go
// 通过 ldflags 注入版本信息
var (
    GitCommitID  string
    Version      string
    Architecture string
    Timestamp    string
)
```
## 四、在线安装镜像设计
### 4.1 Dockerfile.online 分析
```dockerfile
# 第一阶段：下载资源
FROM alpine:3.18 AS builder

WORKDIR /build

# 安装必要工具
RUN apk add curl tar gzip

# 下载 RPM 包
RUN curl -L -o rpm.tar.gz https://openfuyao.obs.cn-north-4.myhuaweicloud.com/rpm/releases/download/v0.0.1/rpm.tar.gz && \
    tar -xzf rpm.tar.gz -C tmp --no-same-owner && \
    rm rpm.tar.gz

# 下载 Kubernetes 组件
RUN curl -L -o kubectl-v1.34.3-of.1-arm64 https://.../kubectl && \
    curl -L -o kubelet-v1.34.3-of.1-arm64 https://.../kubelet && \
    curl -L -o kubelet-v1.34.3-of.1-amd64 https://.../kubelet

# 下载 Containerd
RUN curl -L -o containerd-v2.1.1-linux-amd64.tar.gz https://.../containerd-*.tar.gz

# 下载 CNI 插件
RUN curl -L -o cni-plugins-linux-amd64-v1.4.1.tgz https://.../cni-plugins-*.tgz

# 下载 Helm
RUN curl -L -o helm-v3.14.2-linux-amd64.tar.gz https://.../helm-*.tar.gz

# 下载工具
RUN curl -L -o yq_linux_amd64 https://.../yq_linux_amd64 && \
    curl -L -o jq-linux-amd64 https://.../jq-linux-amd64

# 下载证书工具
RUN curl -L -o cfssl_1.6.4_linux_amd64 https://.../cfssl && \
    curl -L -o cfssljson_1.6.4_linux_amd64 https://.../cfssljson

# 下载 etcdctl
RUN curl -L -o etcdctl-v3.5.6-linux-amd64 https://.../etcdctl

# 下载 Charts
RUN curl -L -o charts.tar.gz https://.../charts.tar.gz

# 组织目录结构
RUN mkdir -p source/files && \
    mv tmp/* source/ && \
    mv kubectl-* source/files/ && \
    mv kubelet-* source/files/ && \
    ...

# 打包
RUN tar -czf source.tar.gz -C source .

# 第二阶段：最终镜像
FROM scratch
COPY --from=builder /build/source.tar.gz /bkesource/source.tar.gz
```
### 4.2 资源清单
```
source.tar.gz
├── files/
│   ├── kubectl-v1.34.3-of.1-amd64      # Kubectl (amd64)
│   ├── kubectl-v1.34.3-of.1-arm64      # Kubectl (arm64)
│   ├── kubelet-v1.34.3-of.1-amd64      # Kubelet (amd64)
│   ├── kubelet-v1.34.3-of.1-arm64      # Kubelet (arm64)
│   ├── containerd-v2.1.1-linux-amd64.tar.gz
│   ├── containerd-v2.1.1-linux-arm64.tar.gz
│   ├── cni-plugins-linux-amd64-v1.4.1.tgz
│   ├── cni-plugins-linux-arm64-v1.4.1.tgz
│   ├── helm-v3.14.2-linux-amd64.tar.gz
│   ├── helm-v3.14.2-linux-arm64.tar.gz
│   ├── yq_linux_amd64                   # YAML 处理工具
│   ├── yq_linux_arm64
│   ├── jq-linux-amd64                   # JSON 处理工具
│   ├── jq-linux-arm64
│   ├── cfssl_1.6.4_linux_amd64          # 证书工具
│   ├── cfssl_1.6.4_linux_arm64
│   ├── cfssljson_1.6.4_linux_amd64
│   ├── cfssljson_1.6.4_linux_arm64
│   ├── cfssl-certinfo_1.6.4_linux_amd64
│   ├── cfssl-certinfo_1.6.4_linux_arm64
│   ├── runc-amd64                       # 容器运行时
│   ├── runc-arm64
│   ├── etcdctl-v3.5.6-linux-amd64       # Etcd 客户端
│   ├── etcdctl-v3.5.6-linux-arm64
│   ├── charts.tar.gz                    # Helm Charts
│   └── nfsshare.tar.gz                  # NFS 共享
└── rpm/
    ├── centos/
    │   ├── 7/amd64/
    │   └── 8/amd64/
    └── ubuntu/
        └── 20.04/amd64/
```
## 五、构建流程
### 5.1 构建流程图
```mermaid
flowchart TB
    subgraph Binary["bkeadm 二进制镜像构建"]
        B1[准备构建参数] --> B2[拉取 Builder 镜像]
        B2 --> B3[编译 Go 代码]
        B3 --> B4[生成 bkeadm 二进制]
        B4 --> B5[复制到 scratch 镜像]
        B5 --> B6[推送到镜像仓库]
    end

    subgraph Online["在线安装镜像构建"]
        O1[准备资源列表] --> O2[拉取 alpine 镜像]
        O2 --> O3[并行下载资源文件]
        O3 --> O4[组织目录结构]
        O4 --> O5[打包 source.tar.gz]
        O5 --> O6[复制到 scratch 镜像]
        O6 --> O7[推送到镜像仓库]
    end

    START([开始构建]) --> Binary
    START --> Online
    B6 --> END([构建完成])
    O7 --> END
```
### 5.2 构建命令
#### 构建 bkeadm 二进制镜像
```bash
# 单架构构建
docker buildx build . -f build/Dockerfile \
    -o type=image,name=cr.openfuyao.cn/openfuyao/bkeadm:v1.0.0,push=true \
    --platform=linux/amd64 \
    --build-arg=COMMIT=$(git rev-parse HEAD) \
    --build-arg=VERSION=v1.0.0 \
    --build-arg=SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct)

# 多架构构建
docker buildx build . -f build/Dockerfile \
    -o type=image,name=cr.openfuyao.cn/openfuyao/bkeadm:v1.0.0,push=true \
    --platform=linux/amd64,linux/arm64 \
    --provenance=false \
    --build-arg=COMMIT=$(git rev-parse HEAD) \
    --build-arg=VERSION=v1.0.0 \
    --build-arg=SOURCE_DATE_EPOCH=$(git log -1 --pretty=%ct)
```
#### 构建在线安装镜像
```bash
# 构建在线安装镜像
docker buildx build . -f build/Dockerfile.online \
    -o type=image,name=cr.openfuyao.cn/openfuyao/bke-online:v1.0.0,push=true \
    --platform=linux/amd64,linux/arm64 \
    --provenance=false
```
## 六、Makefile 构建
### 6.1 Makefile 结构
```makefile
# 版本信息
GOLANG=1.19.x
ARCH ?= linux/amd64,linux/arm64
version = "v1.0.0"
COMMIT_ID ?= $(shell git rev-parse HEAD)

# ldflags
LDFLAGS = -s -w \
    -X main.gitCommitId=$(COMMIT_ID) \
    -X main.architecture=$(shell go env GOHOSTOS)/$(shell go env GOHOSTARCH) \
    -X main.timestamp=$(timestamp) \
    -X main.ver=$(version)

# 目标
.PHONY: build
build:
    @go build -ldflags="$(LDFLAGS)" -o bin/bke .

.PHONY: docker-build
docker-build:
    CGO_ENABLED=0 GOARCH=$(ARCH) go build \
        -tags "remote exclude_graphdriver_btrfs btrfs_noversion exclude_graphdriver_devicemapper containers_image_openpgp" \
        -ldflags "$(LDFLAGS)" \
        -o bin/bke_$(ARCH) .

.PHONY: buildx
buildx:
    @docker run --privileged --rm tonistiigi/binfmt --install all
    @docker buildx create --use --name mybuilder
    @docker buildx inspect mybuilder --bootstrap

.PHONY: docker
docker:
    @docker build -t registry.cn-hangzhou.aliyuncs.com/bocloud/bkeadm:latest .
    @docker push registry.cn-hangzhou.aliyuncs.com/bocloud/bkeadm:latest
```
### 6.2 常用构建命令
```bash
# 本地编译
make build

# 多架构编译
make release

# Docker 构建
make docker

# 设置 buildx 多架构支持
make buildx
```
## 七、镜像使用场景
### 7.1 bkeadm 二进制镜像
```bash
# 提取 bkeadm 二进制
docker create --name bkeadm cr.openfuyao.cn/openfuyao/bkeadm:v1.0.0
docker cp bkeadm:/bkeadm ./bkeadm
docker rm bkeadm

# 直接运行
docker run --rm -v $(pwd):/workspace cr.openfuyao.cn/openfuyao/bkeadm:v1.0.0 version
```
### 7.2 在线安装镜像
```bash
# 提取安装资源
docker create --name bke-source cr.openfuyao.cn/openfuyao/bke-online:v1.0.0
docker cp bke-source:/bkesource/source.tar.gz ./source.tar.gz
docker rm bke-source

# 解压使用
tar -xzf source.tar.gz
```
## 八、多架构支持
### 8.1 支持的架构
| 架构 | 说明 |
|------|------|
| `linux/amd64` | x86_64 架构 (主流服务器) |
| `linux/arm64` | ARM64 架构 (国产化服务器) |
### 8.2 多架构构建配置
```bash
# 安装 QEMU 模拟器
docker run --privileged --rm tonistiigi/binfmt --install all

# 创建 buildx 构建器
docker buildx create --use --name mybuilder
docker buildx inspect mybuilder --bootstrap

# 验证构建器支持的平台
docker buildx inspect --bootstrap
```
## 九、镜像优化策略
### 9.1 当前优化措施
| 优化项 | 方法 | 效果 |
|--------|------|------|
| 最小化基础镜像 | 使用 `scratch` | 镜像大小 ~30MB |
| 静态编译 | `CGO_ENABLED=0` | 无动态依赖 |
| 去除调试信息 | `-ldflags="-s -w"` | 减小二进制大小 |
| 多阶段构建 | Builder + Release | 仅保留必要文件 |
### 9.2 构建缓存优化
```dockerfile
# 利用构建缓存 - 先复制 go.mod
COPY go.mod go.sum ./
RUN go mod download

# 再复制源代码
COPY . .
RUN go build -o /go/bin/app
```
## 十、CI/CD 集成
### 10.1 GitHub Actions 示例
```yaml
name: Build and Push Image

on:
  push:
    tags:
      - 'v*'

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up QEMU
        uses: docker/setup-qemu-action@v3

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Login to Registry
        uses: docker/login-action@v3
        with:
          registry: cr.openfuyao.cn
          username: ${{ secrets.REGISTRY_USER }}
          password: ${{ secrets.REGISTRY_PASSWORD }}

      - name: Build and Push
        uses: docker/build-push-action@v5
        with:
          context: .
          file: build/Dockerfile
          platforms: linux/amd64,linux/arm64
          push: true
          tags: |
            cr.openfuyao.cn/openfuyao/bkeadm:${{ github.ref_name }}
            cr.openfuyao.cn/openfuyao/bkeadm:latest
          build-args: |
            COMMIT=${{ github.sha }}
            VERSION=${{ github.ref_name }}
            SOURCE_DATE_EPOCH=${{ github.event.head_commit.timestamp }}
```
## 十一、总结
| 维度 | bkeadm 二进制镜像 | 在线安装镜像 |
|------|-------------------|--------------|
| **基础镜像** | scratch | scratch |
| **构建基础** | golang:1.24.5 | alpine:3.18 |
| **内容** | bkeadm 二进制 | 完整离线资源包 |
| **大小** | ~30MB | ~2GB+ |
| **用途** | CLI 工具分发 | 离线环境部署 |
| **多架构** | amd64, arm64 | amd64, arm64 |
        
