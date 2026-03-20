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

package initialize

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/spf13/afero"
	"gopkg.in/yaml.v3"
	"gopkg.openfuyao.cn/cluster-api-provider-bke/api/bkecommon/v1beta1"
	bkecommon "gopkg.openfuyao.cn/cluster-api-provider-bke/common"
	configinit "gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/initialize"
	configsource "gopkg.openfuyao.cn/cluster-api-provider-bke/common/source"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"gopkg.openfuyao.cn/bkeadm/pkg/build"
	"gopkg.openfuyao.cn/bkeadm/pkg/cluster"
	"gopkg.openfuyao.cn/bkeadm/pkg/common/types"
	"gopkg.openfuyao.cn/bkeadm/pkg/config"
	econd "gopkg.openfuyao.cn/bkeadm/pkg/executor/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/k8s"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure/k3s"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure/kubelet"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize/bkeagent"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize/bkeconfig"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize/bkeconsole"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize/clusterapi"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize/repository"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize/syscompat"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize/timezone"
	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type Options struct {
	root.Options
	File           string   `json:"file"`
	Args           []string `json:"args"`
	HostIP         string   `json:"hostIP"`
	Domain         string   `json:"domain"`
	KubernetesPort string   `json:"kubernetesPort"`
	ImageRepoPort  string   `json:"imageRepoPort"`
	YumRepoPort    string   `json:"yumRepoPort"`
	ChartRepoPort  string   `json:"chartRepoPort"`
	ClusterAPI     string   `json:"clusterAPI"`
	OFVersion      string   `json:"OFVersion"`
	VersionUrl     string   `json:"versionUrl"`
	NtpServer      string   `json:"ntpServer"`
	Runtime        string   `json:"runtime"`
	RuntimeStorage string   `json:"runtimeStorage"`
	OnlineImage    string   `json:"onlineImage"`
	OtherRepo      string   `json:"otherRepo"`
	OtherSource    string   `json:"otherSource"`
	OtherChart     string   `json:"otherChart"` // TODO: helm chart 私有源地址
	InstallConsole bool     `json:"installConsole"`
	EnableNTP      bool     `json:"enableNTP"` // 是否启用NTP服务，默认为true

	ImageRepoCAFile    string `json:"imageRepoCAFile"`    // 镜像仓库CA证书文件
	ImageRepoUsername  string `json:"imageRepoUsername"`  // 镜像仓库用户名
	ImageRepoPassword  string `json:"imageRepoPassword"`  // 镜像仓库密码
	ImageRepoTLSVerify bool   `json:"imageRepoTLSVerify"` // 是否验证TLS证书

	ImageFilePath   string `json:"imageFilePath"`   // 引导节点初始化本地镜像路径
	AgentHealthPort string `json:"agentHealthPort"` // 集群节点代理的健康监听端口

	// Dependency Injection
	FS               afero.Fs
	DownloadFunc     func(url, dest string) error
	SetPatchConfigFn func(version, path, key string) error
	K8sClient        k8s.KubernetesClient
}

var oc repository.OtherRepo

// 主初始化
func (op *Options) Initialize() {
	log.BKEFormat(log.INFO, fmt.Sprint("BKE initialize ..."))
	op.nodeInfo()
	err := op.Validate()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Validation failure, %s", err.Error()))
		return
	}
	err = op.setTimezone()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Timezone failure, %s", err.Error()))
		return
	}
	err = op.prepareEnvironment()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to prepare environment, %s", err.Error()))
		return
	}
	err = op.ensureContainerServer()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to start the container service, %s", err.Error()))
		return
	}
	err = op.ensureRepository()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to start warehouse, %s", err.Error()))
		return
	}
	err = op.ensureClusterAPI()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to start cluster API, %s", err.Error()))
		return
	}
	// 条件性安装 bkeconsole
	if op.InstallConsole {
		err = op.ensureConsoleAll()
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to start Console, %s", err.Error()))
			return
		}
	} else {
		log.BKEFormat(log.INFO, "Skipping bkeconsole installation as requested")
	}

	// Generating a Configuration File
	op.generateClusterConfig()
	// chmod /bke dir and file permission (reason: openEuler 20.03 umask 0077 need modify permission)
	op.modifyPermission()
	log.BKEFormat(log.INFO, "BKE initialization is complete")
	op.deployCluster()
}

// 节点信息收集打印
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

	if op.InstallConsole {
		log.BKEFormat(log.INFO, "BKE Console: ENABLED")
	} else {
		log.BKEFormat(log.INFO, "BKE Console: DISABLED")
	}

}

// 验证初始化参数，环境验证，验证运行环境是否满足安装要求
func (op *Options) Validate() error {
	log.BKEFormat(log.INFO, fmt.Sprint("BKE initialize environment check..."))
	var err error

	oc, err = repository.ParseOnlineConfig(op.Domain, op.OnlineImage, op.OtherRepo, op.OtherSource, op.OtherChart)

	if err != nil {
		return errors.New(fmt.Sprintf("Configuration parsing failure %s", err.Error()))
	}

	op.logAuthMode()

	if err = op.validateDiskSpace(); err != nil {
		return err
	}

	if err = op.validatePorts(); err != nil {
		return err
	}

	op.setGlobalCustomExtra()
	return nil
}

// logAuthMode 记录认证模式
func (op *Options) logAuthMode() {
	if op.ImageRepoTLSVerify {
		if op.ImageRepoUsername != "" && op.ImageRepoPassword != "" {
			log.BKEFormat(log.INFO, "Password authentication will be used")
		} else if op.ImageRepoCAFile != "" {
			log.BKEFormat(log.INFO, "CA certificate authentication will be used")
		} else {
			log.BKEFormat(log.WARN, "Client authentication enabled but no credentials provided")
		}
	}
}

// validateDiskSpace 检查磁盘空间
func (op *Options) validateDiskSpace() error {
	_, free := utils.DiskUsage(global.Workspace)
	if utils.Exists(path.Join(global.Workspace, utils.ImageDataDirectory)) {
		if free/1024/1024/1024 < utils.MinDiskSpaceExisting {
			return errors.New(fmt.Sprintf("The available space of the working directory %s is less than %d GB",
				global.Workspace, utils.MinDiskSpaceExisting))
		}
	} else {
		if free/1024/1024/1024 < utils.MinDiskSpace {
			return errors.New(fmt.Sprintf("The available space of the working directory %s is less than %d GB",
				global.Workspace, utils.MinDiskSpace))
		}
	}
	return nil
}

// validatePorts 检查端口占用
func (op *Options) validatePorts() error {
	ports := []string{op.KubernetesPort, op.ImageRepoPort, op.ChartRepoPort, op.YumRepoPort, "2049"}
	ports = op.filterExistingContainerPorts(ports)

	err := utils.CheckPorts(ports)
	if err != nil {
		return errors.New(fmt.Sprintf("The port is already in use %s", err.Error()))
	}
	return nil
}

// filterExistingContainerPorts 过滤已存在容器的端口
func (op *Options) filterExistingContainerPorts(ports []string) []string {
	if infrastructure.IsDocker() {
		ports = op.filterDockerContainerPorts(ports)
	}
	if infrastructure.IsContainerd() {
		ports = op.filterContainerdPorts(ports)
	}
	return ports
}

// filterDockerContainerPorts 过滤 Docker 容器端口
func (op *Options) filterDockerContainerPorts(ports []string) []string {
	if _, ok := global.Docker.ContainerExists(utils.LocalKubernetesName); ok {
		ports = utils.RemoveStringObject(ports, op.KubernetesPort)
	}
	if _, ok := global.Docker.ContainerExists(utils.LocalImageRegistryName); ok {
		ports = utils.RemoveStringObject(ports, op.ImageRepoPort)
	}
	if _, ok := global.Docker.ContainerExists(utils.LocalChartRegistryName); ok {
		ports = utils.RemoveStringObject(ports, op.ChartRepoPort)
	}
	if _, ok := global.Docker.ContainerExists(utils.LocalNFSRegistryName); ok {
		ports = utils.RemoveStringObject(ports, "2049")
	}
	if _, ok := global.Docker.ContainerExists(utils.LocalYumRegistryName); ok {
		ports = utils.RemoveStringObject(ports, op.YumRepoPort)
	}
	return ports
}

// filterContainerdPorts 过滤 Containerd 容器端口
func (op *Options) filterContainerdPorts(ports []string) []string {
	if _, ok := econd.ContainerExists(utils.LocalKubernetesName); ok {
		ports = utils.RemoveStringObject(ports, op.KubernetesPort)
	}
	if _, ok := econd.ContainerExists(utils.LocalImageRegistryName); ok {
		ports = utils.RemoveStringObject(ports, op.ImageRepoPort)
	}
	if _, ok := econd.ContainerExists(utils.LocalChartRegistryName); ok {
		ports = utils.RemoveStringObject(ports, op.ChartRepoPort)
	}
	if _, ok := econd.ContainerExists(utils.LocalNFSRegistryName); ok {
		ports = utils.RemoveStringObject(ports, "2049")
	}
	if _, ok := econd.ContainerExists(utils.LocalYumRegistryName); ok {
		ports = utils.RemoveStringObject(ports, op.YumRepoPort)
	}
	return ports
}

// setGlobalCustomExtra 设置全局自定义扩展配置
func (op *Options) setGlobalCustomExtra() {
	global.CustomExtra["domain"] = op.Domain
	global.CustomExtra["host"] = op.HostIP
	global.CustomExtra["imageRepoPort"] = op.ImageRepoPort
	global.CustomExtra["yumRepoPort"] = op.YumRepoPort
	global.CustomExtra["chartRepoPort"] = op.ChartRepoPort
	global.CustomExtra["clusterapi"] = op.ClusterAPI
	global.CustomExtra["nfsserverpath"] = "/"
	global.CustomExtra["onlineImage"] = op.OnlineImage
	global.CustomExtra["otherRepo"] = oc.Repo
	global.CustomExtra["otherRepoIp"] = oc.RepoIP
	global.CustomExtra["otherSource"] = oc.Source
	global.CustomExtra["otherChart"] = oc.ChartRepo
	global.CustomExtra["otherChartIp"] = oc.ChartRepoIP
}

// 设置时区和NTP服务器
func (op *Options) setTimezone() error {
	log.BKEFormat(log.INFO, "set up the host machine zone")
	err := timezone.SetTimeZone()
	if err != nil {
		return err
	}
	// 如果禁用了NTP服务，则跳过NTP服务器设置
	if !op.EnableNTP {
		log.BKEFormat(log.INFO, "NTP service is disabled, skipping NTP server setup")
		op.NtpServer = "" // 设置为空字符串，后续配置会跳过NTP设置
		return nil
	}
	log.BKEFormat(log.INFO, "set ntp server")
	newNTPServer, err := timezone.NTPServer(op.NtpServer, op.HostIP, len(oc.Repo) > 0)
	if err != nil {
		return err
	}
	op.NtpServer = newNTPServer
	return nil
}

// 准备初始化环境，配置下载源，设置hosts，配置私有仓库CA证书等
func (op *Options) prepareEnvironment() error {
	log.BKEFormat(log.INFO, "config local source")

	op.configLocalSource()

	hostIP, domain := op.resolveHostIPAndDomain()
	if err := syscompat.SetHosts(hostIP, domain); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to set hosts %s", err.Error()))
	}

	clientAuthConfig := op.buildClientAuthConfig()
	if err := op.configurePrivateRegistry(clientAuthConfig); err != nil {
		return err
	}

	return op.initRepositories(clientAuthConfig)
}

// configLocalSource 配置本地源
func (op *Options) configLocalSource() {
	// OtherRepo设置私有镜像源，OnlineImage指定二进制来源，两者有一个就认为是在线安装
	if op.OtherRepo == "" && op.OnlineImage == "" {
		baseurl := "file://" + path.Join(global.Workspace, utils.SourceDataDirectory)
		if len(oc.Source) > 0 {
			baseurl = oc.Source
		}
		err := configsource.SetSource(baseurl)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to set download source %s", err.Error()))
		}
	}
}

// resolveHostIPAndDomain 解析主机IP和域名
func (op *Options) resolveHostIPAndDomain() (string, string) {
	hostIP, domain := op.HostIP, op.Domain
	// 如果otherRepo不为空，判定为在线模式
	if op.OtherRepo != "" || op.OnlineImage != "" {
		registryHost, _ := repository.ParseRegistryHostPort(op.OtherRepo)
		if net.ParseIP(registryHost) != nil {
			hostIP = registryHost
			log.BKEFormat(log.INFO, fmt.Sprintf("在线模式：domain：%s 绑定到otherRepo的IP：%s", domain, hostIP))
		} else {
			log.BKEFormat(log.INFO, fmt.Sprintf("在线模式：domain：%s 绑定到默认IP：%s", domain, hostIP))
		}
		// 如果otherRepo为空，判定为离线模式
	} else {
		log.BKEFormat(log.INFO, fmt.Sprintf("离线模式：domain：%s 绑定到引导节点IP：%s", domain, hostIP))
	}
	// 处理默认仓库地址的情况
	if strings.Contains(oc.Repo, configinit.DefaultImageRepo) {
		hostIP = oc.RepoIP
		domain = strings.Split(strings.Split(oc.Repo, "/")[0], ":")[0]
	}
	return hostIP, domain
}

// buildClientAuthConfig 构建客户端认证配置
func (op *Options) buildClientAuthConfig() *repository.CertificateConfig {
	return &repository.CertificateConfig{
		TLSVerify: op.ImageRepoTLSVerify,
		Username:  op.ImageRepoUsername,
		Password:  op.ImageRepoPassword,
		CAFile:    op.ImageRepoCAFile,
	}
}

// configurePrivateRegistry 配置私有仓库
func (op *Options) configurePrivateRegistry(cfg *repository.CertificateConfig) error {
	registryHost, registryPort := repository.ParseRegistryHostPort(oc.Repo)
	if registryHost != "" && registryPort != "" {
		cfg.RegistryHost = registryHost
		cfg.RegistryPort = registryPort
		if cfg.TLSVerify && cfg.CAFile != "" {
			if err := repository.SetupCACertificate(cfg); err != nil {
				return fmt.Errorf("配置私有仓库CA证书失败：%v", err)
			}
		}
	} else {
		log.BKEFormat(log.WARN, "无法解析私有仓库地址，跳过CA证书配置")
	}
	return nil
}

// initRepositories 初始化仓库
func (op *Options) initRepositories(clientAuthConfig *repository.CertificateConfig) error {
	if err := repository.RepoInit(oc, clientAuthConfig); err != nil {
		return err
	}
	if err := repository.DecompressionSystemSourceFile(); err != nil {
		return err
	}
	if op.ImageFilePath == "" {
		if err := repository.SourceInit(oc); err != nil {
			return err
		}
	}
	if err := syscompat.RepoUpdate(); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to update repo %s", err.Error()))
	}
	if err := syscompat.Compat(); err != nil {
		return errors.New(fmt.Sprintf("The system is not compatible %s", err.Error()))
	}
	syscompat.SetSysctl()
	return nil
}

// 容器运行时安装
func (op *Options) ensureContainerServer() error {
	err := repository.PrepareRepositoryDependOn(op.ImageFilePath)
	if err != nil {
		return err
	}
	op.modifyPermission()
	result, err := repository.VerifyContainerdFile(op.ImageFilePath)
	if err != nil {
		return err
	}
	containerdFile := result.ContainerdList[0]
	cniPluginFile := result.CniPluginList[0]
	for _, cond := range result.ContainerdList {
		if strings.Contains(cond, runtime.GOARCH) {
			containerdFile = cond
			continue
		}
	}
	for _, cni := range result.CniPluginList {
		if strings.Contains(cni, runtime.GOARCH) {
			cniPluginFile = cni
			continue
		}
	}
	err = infrastructure.RuntimeInstall(infrastructure.RuntimeConfig{
		Runtime:        op.Runtime,
		RuntimeStorage: op.RuntimeStorage,
		Domain:         op.Domain,
		ImageRepoPort:  op.ImageRepoPort,
		ContainerdFile: result.FilePath + "/" + containerdFile,
		CniPluginFile:  result.FilePath + "/" + cniPluginFile,
		DockerdFile:    result.FilePath + "/" + strings.Replace(utils.KylinDocker, "{.arch}", runtime.GOARCH, -1),
		HostIP:         op.HostIP,
		CAFile:         op.ImageRepoCAFile,
	})
	if err != nil {
		return err
	}
	return nil
}

// 仓库服务启动
func (op *Options) ensureRepository() error {
	log.BKEFormat(log.INFO, "Start the base dependency service")
	var err error
	// 新安装方式，registry镜像从本地获取
	if op.ImageFilePath != "" {
		err = repository.LoadLocalImage()
		if err != nil {
			return err
		}
	}
	// 加载本地仓库镜像
	err = repository.LoadLocalRepository()
	if err != nil {
		return err
	}
	// 启动镜像仓库服务
	err = repository.ContainerServer(op.ImageFilePath, op.ImageRepoPort, oc.Repo, oc.Image)
	if err != nil {
		return err
	}
	// sync local image to registry
	if op.ImageFilePath != "" {
		err = repository.SyncLocalImage(op.ImageRepoPort)
		if err != nil {
			return err
		}
	}
	// 启动YUM仓库服务
	err = repository.YumServer(op.ImageFilePath, op.ImageRepoPort, op.YumRepoPort, oc.Repo, oc.Image)
	if err != nil {
		return err
	}
	// 启动Chart仓库服务
	err = repository.ChartServer(op.ImageFilePath, op.ImageRepoPort, op.ChartRepoPort, oc.Repo, oc.Image)
	if err != nil {
		return err
	}
	if op.ImageFilePath == "" {
		// 启动NFS服务
		err = repository.NFSServer(op.ImageRepoPort, oc.Repo, oc.Image)
		if err != nil {
			return err
		}
	}
	return nil
}

// 从文件名中提取版本号
func extractVersionFromFilename(filename string) string {
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	versionRegex := regexp.MustCompile(`^(?:.*-)?(latest|v\d+\.\d+(?:[-.]\w+)*)$`)
	matches := versionRegex.FindStringSubmatch(base)
	if len(matches) >= utils.MatchFields {
		return matches[1]
	}
	return ""
}

// 离线模式生成ConfigMap
func (op *Options) offlineGenerateDeployCM(patchesDir string) error {
	fs := op.FS
	if fs == nil {
		fs = afero.NewOsFs()
	}
	setPatchFn := op.SetPatchConfigFn
	if setPatchFn == nil {
		setPatchFn = bkeconfig.SetPatchConfig
	}

	if _, err := fs.Stat(patchesDir); os.IsNotExist(err) {
		log.BKEFormat(log.WARN, fmt.Sprintf("patchesDir %s not exist, use default", patchesDir))
		return err
	}

	entries, err := afero.ReadDir(fs, patchesDir)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("read %s fail %s, use default", patchesDir, err))
		return err
	}

	for _, entry := range entries {
		version := extractVersionFromFilename(entry.Name())
		if op.OFVersion == version {
			log.BKEFormat(log.INFO, fmt.Sprintf("version %s file, generate cm", op.OFVersion))
			fullPath := filepath.Join(patchesDir, entry.Name())
			cmKey := fmt.Sprintf("%s%s", utils.PatchValuePrefix, version)
			if err = setPatchFn(version, fullPath, cmKey); err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("generate cm fail %s, use default", err))
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("offline patch %s not exist, use default", op.OFVersion)
}

// 解析YAML字节数据到切片映射
func parseYAMLBytesToSliceMap(data []byte) ([]map[string]string, error) {
	var rawList []map[string]string
	if err := yaml.Unmarshal(data, &rawList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML: %w", err)
	}

	result := make([]map[string]string, 0, len(rawList))
	for i, item := range rawList {
		if _, exists := item["openFuyaoVersion"]; !exists {
			return nil, fmt.Errorf("item at index %d is missing required field 'openFuyaoVersion'", i)
		}
		if _, exists := item["filePath"]; !exists {
			return nil, fmt.Errorf("item at index %d is missing required field 'filePath'", i)
		}
		item["filePath"], _ = strings.CutPrefix(item["filePath"], "./")
		result = append(result, item)
	}
	return result, nil
}

// 在线模式生成ConfigMap
func (op *Options) onlineGenerateDeployCM() error {
	fs := op.FS
	if fs == nil {
		fs = afero.NewOsFs()
	}
	downloadFn := op.DownloadFunc
	if downloadFn == nil {
		downloadFn = utils.DownloadFile
	}
	setPatchFn := op.SetPatchConfigFn
	if setPatchFn == nil {
		setPatchFn = bkeconfig.SetPatchConfig
	}

	patchesDir := filepath.Join(global.Workspace, utils.PatchDataDirectory)
	if err := fs.MkdirAll(patchesDir, utils.DefaultDirPermission); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("mkdir dir %s err %v, use default", patchesDir, err))
		return err
	}

	url := op.VersionUrl
	if strings.HasSuffix(url, "/") {
		url = strings.TrimSuffix(url, "/")
	}
	indexURL := fmt.Sprintf("%s/index.yaml", url)
	indexFile := filepath.Join(patchesDir, "index.yaml")

	if err := downloadFn(indexURL, indexFile); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("download file %s err %v, use default", indexURL, err))
		return err
	}
	defer func() {
		_ = fs.Remove(indexFile)
	}()

	data, err := afero.ReadFile(fs, indexFile)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("read index.yaml failed: %v", err))
		return err
	}

	return op.processIndexYAML(downloadFn, setPatchFn, data, url, patchesDir)
}

// 处理index.yaml数据，索引处理
func (op *Options) processIndexYAML(
	downloadFn func(string, string) error,
	setPatchFn func(string, string, string) error,
	data []byte,
	baseURL, patchesDir string,
) error {
	patchRes, err := parseYAMLBytesToSliceMap(data)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("parseYAMLFileToSliceMap err %v, use default", err))
		return err
	}

	for _, value := range patchRes {
		if value["openFuyaoVersion"] == op.OFVersion {
			filePath := value["filePath"]
			downloadURL := fmt.Sprintf("%s/%s", baseURL, filePath)
			downloadFile := filepath.Join(patchesDir, filePath)

			if err = downloadFn(downloadURL, downloadFile); err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("download file %s err %v, use default", downloadURL, err))
				return err
			}

			cmKey := fmt.Sprintf("%s%s", utils.PatchValuePrefix, op.OFVersion)
			if err = setPatchFn(op.OFVersion, downloadFile, cmKey); err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("generate cm fail %s, use default", err))
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("online patch %s not exist, use default", op.OFVersion)
}

// 生成部署所需的ConfigMap
func (op *Options) generateDeployCM() error {
	// generate patch config map from local image
	if op.ImageFilePath != "" {
		patchesDir := filepath.Join(global.Workspace, utils.LocalPatchDirectory)
		return op.offlineGenerateDeployCM(patchesDir)
	}
	if oc.Repo == "" && oc.Image == "" {
		patchesDir := filepath.Join(global.Workspace, utils.PatchDataDirectory)
		return op.offlineGenerateDeployCM(patchesDir)
	} else {
		return op.onlineGenerateDeployCM()
	}
}

// 获取Cluster API版本
func (op *Options) getClusterAPIVersion(openFuyaoVersion, defaultVersion string) (string, string) {
	var client k8s.KubernetesClient
	if op.K8sClient != nil {
		client = op.K8sClient
	} else if global.K8s != nil {
		client = global.K8s
	} else {
		var err error
		client, err = k8s.NewKubernetesClient("")
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("failed to init k8s client: %v", err))
			return defaultVersion, defaultVersion
		}
		global.K8s = client
	}

	patchCmKey := fmt.Sprintf("cm.%s", openFuyaoVersion)
	k8sClient := client.GetClient()
	patchConfigMap, err := k8sClient.CoreV1().ConfigMaps("openfuyao-patch").Get(context.TODO(), patchCmKey, metav1.GetOptions{})
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("failed to get patch cm, err: %v", err))
		return defaultVersion, defaultVersion
	}

	data, ok := patchConfigMap.Data[openFuyaoVersion]
	if !ok {
		log.BKEFormat(log.WARN, fmt.Sprintf("cm data not contain %s key", openFuyaoVersion))
		return defaultVersion, defaultVersion
	}

	cfg := &build.BuildConfig{}
	if err = yaml.Unmarshal([]byte(data), cfg); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Unable to serialize err %s", err))
		return defaultVersion, defaultVersion
	}

	return extractVersionsFromConfig(cfg, defaultVersion)
}

// 扁平化镜像列表
func flattenImages(cfg *build.BuildConfig) []build.Image {
	if cfg == nil {
		return nil
	}
	var images []build.Image
	for _, repo := range cfg.Repos {
		for _, sub := range repo.SubImages {
			images = append(images, sub.Images...)
		}
	}
	return images
}

// 查找指定镜像的标签
func findImageTag(cfg *build.BuildConfig, imageName, defaultVersion string) string {
	for _, img := range flattenImages(cfg) {
		if img.Name == imageName && len(img.Tag) > 0 {
			return img.Tag[0]
		}
	}
	return defaultVersion
}

// 从配置中提取版本信息
func extractVersionsFromConfig(cfg *build.BuildConfig, defaultVersion string) (string, string) {
	manifestsVersion := findImageTag(cfg, "bke-manifests", defaultVersion)
	providerVersion := findImageTag(cfg, "cluster-api-provider-bke", defaultVersion)

	return manifestsVersion, providerVersion
}

// Cluster API安装
func (op *Options) ensureClusterAPI() error {
	err := infrastructure.StartLocalKubernetes(k3s.Config{
		OnlineImage:    oc.Image,
		OtherRepo:      oc.Repo,
		OtherRepoIP:    oc.RepoIP,
		HostIP:         op.HostIP,
		ImageRepo:      op.Domain,
		ImageRepoPort:  op.ImageRepoPort,
		KubernetesPort: op.KubernetesPort,
	}, op.ImageFilePath)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to start kubernetes %s", err.Error()))
		return err
	}

	// 将需要的openfuyao版本信息写入到configmap供后续安装使用
	if op.OFVersion != "" {
		if err = op.generateDeployCM(); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Deploy version %s not in released version list", op.OFVersion))
			return fmt.Errorf("version %s not in released version list", op.OFVersion)
		}
	}

	err = containerd.ApplyContainerdCfg(fmt.Sprintf("%s:%s", op.Domain, op.ImageRepoPort))
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install containerd config %s", err.Error()))
		return err
	}

	err = kubelet.ApplyKubeletCfg()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install kubelet config %s", err.Error()))
		return err
	}

	err = bkeagent.InstallBKEAgentCRD()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install bkeagent %s", err.Error()))
		return err
	}

	var repo string
	localRepoPath := fmt.Sprintf("%s:%s/%s/", op.Domain, "443", bkecommon.ImageRegistryKubernetes)

	// 优先级：ImageFilePath > oc.Repo > (oc.Image为空时使用本地) > 默认值
	if op.ImageFilePath != "" {
		// ImageFilePath 不为空，使用本地仓库路径
		repo = localRepoPath
	} else if oc.Repo != "" {
		// oc.Repo 不为空，使用 oc.Repo
		repo = oc.Repo
	} else if oc.Image == "" {
		// oc.Image 为空（离线场景），使用本地仓库路径
		repo = localRepoPath
	}
	// 如果都不满足，repo 保持为空字符串（使用默认值）
	manifestsVersion, providerVersion := op.getClusterAPIVersion(op.OFVersion, op.ClusterAPI)
	err = clusterapi.DeployClusterAPI(repo, manifestsVersion, providerVersion)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to deploy cluster-api %s", err.Error()))
		return err
	}
	log.BKEFormat(log.INFO, "The cluster-api deployment is complete")
	return nil
}

// BKE Console安装
func (op *Options) ensureConsoleAll() error {
	// 检查是否启用 console 安装
	if !op.InstallConsole {
		log.BKEFormat(log.INFO, "BKE Console installation is disabled")
		return nil
	}

	log.BKEFormat(log.INFO, "Starting BKE Console installation...")

	var repo string
	localRepoPath := fmt.Sprintf("%s:%s/%s/", op.Domain, "443", bkecommon.ImageRegistryKubernetes)

	// 优先级：oc.Repo > (oc.Image为空时使用本地) > 默认值
	if oc.Repo != "" {
		// oc.Repo 不为空，使用 oc.Repo
		repo = oc.Repo
	} else if oc.Image == "" {
		// oc.Image 为空（离线场景），使用本地仓库路径
		repo = localRepoPath
	}
	var sRestartConfig types.K3sRestartConfig
	sRestartConfig = types.K3sRestartConfig{
		OnlineImage:    oc.Image,
		OtherRepo:      oc.Repo,
		OtherRepoIp:    oc.RepoIP,
		HostIP:         op.HostIP,
		ImageRepo:      op.Domain,
		ImageRepoPort:  op.ImageRepoPort,
		KubernetesPort: op.KubernetesPort,
	}

	err := bkeconsole.DeployConsoleAll(sRestartConfig, repo, op.OFVersion)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to deloy console %s", err.Error()))
		return err
	}
	log.BKEFormat(log.INFO, "The bke console deployment is complete")
	return nil
}

// 生成集群配置文件
func (op *Options) generateClusterConfig() {
	log.BKEFormat(log.INFO, "Generate the cluster configuration file")

	data, repo, err := op.prepareClusterConfigData()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("generateClusterConfig func is error %s", err))
		return
	}

	op.createClusterConfigFile(data, repo[0], repo[1], repo[2])
}

func (op *Options) prepareClusterConfigData() (map[string]string, []v1beta1.Repo, error) {
	k8sVersion, err := op.getStartKubernetesVersion()
	if err != nil {
		return nil, []v1beta1.Repo{}, err
	}

	data := map[string]string{
		"chartRepoPort":     fmt.Sprintf("%s", op.ChartRepoPort),
		"clusterapi":        op.ClusterAPI,
		"domain":            op.Domain,
		"host":              op.HostIP,
		"httpDomain":        configinit.DefaultYumRepo,
		"httpIp":            "",
		"httpRepo":          oc.Source,
		"imageRepoPort":     fmt.Sprintf("%s", op.ImageRepoPort),
		"ntpServer":         op.NtpServer,
		"otherRepo":         oc.Repo,
		"otherRepoIp":       oc.RepoIP,
		"runtime":           op.Runtime,
		"yumRepoPort":       fmt.Sprintf("%s", op.YumRepoPort),
		"kubernetesVersion": k8sVersion,
		"agentHealthPort":   op.AgentHealthPort,
	}

	patchesDir := filepath.Join(global.Workspace, utils.PatchDataDirectory)
	if op.ImageFilePath != "" {
		patchesDir = filepath.Join(global.Workspace, utils.LocalPatchDirectory)
	}

	if patchMap := op.ProcessPatchFiles(patchesDir); patchMap != nil {
		for k, v := range patchMap {
			data[k] = v
		}
	}

	imageRepo := op.prepareImageRepoConfig()

	yumRepo := op.prepareHTTPRepoConfig()

	chartRepo := op.prepareChartRepoConfig()

	for k, v := range global.CustomExtra {
		data[k] = v
	}

	return data, []v1beta1.Repo{imageRepo, yumRepo, chartRepo}, nil
}

func (op *Options) prepareImageRepoConfig() v1beta1.Repo {
	imageRepo := v1beta1.Repo{
		Domain: op.Domain,
		Ip:     op.HostIP,
		Port:   op.ImageRepoPort,
		Prefix: bkecommon.ImageRegistryKubernetes,
	}

	if oc.Repo != "" {
		img := strings.Split(oc.Repo, "/")
		img1 := strings.Split(img[0], ":")
		port := "443"
		if len(img1) == utils.HttpUrlFields {
			port = img1[1]
		}
		imageRepo = v1beta1.Repo{
			Domain: img1[0],
			Ip:     oc.RepoIP,
			Port:   port,
			Prefix: strings.TrimRight(strings.Join(img[1:], "/"), "/"),
		}
	} else if oc.Image != "" { // 在线安装，未指定Repo时K8s资源yaml文件就使用默认的镜像，使用prefix做标识
		imageRepo.Prefix = ""
		imageRepo.Domain = "default"
	}

	return imageRepo
}

func (op *Options) prepareChartRepoConfig() v1beta1.Repo {
	ChartRepoIP, err := utils.LoopIP(configinit.DefaultChartRepo)
	if err == nil {
		chartRepo := v1beta1.Repo{
			Domain: configinit.DefaultChartRepo,
			Ip:     ChartRepoIP[0],
			Port:   "443",
			Prefix: "charts",
		}
		return chartRepo
	} else {
		chartRepo := v1beta1.Repo{
			Domain: "",
			Ip:     op.HostIP,
			Port:   op.ChartRepoPort,
			Prefix: "",
		}
		return chartRepo
	}
}

func (op *Options) prepareHTTPRepoConfig() v1beta1.Repo {
	yumRepo := v1beta1.Repo{
		Domain: configinit.DefaultYumRepo,
		Ip:     op.HostIP,
		Port:   op.YumRepoPort,
	}

	if oc.Source != "" {
		httpRepo := strings.TrimLeft(oc.Source, "http://")
		httpRepoArray := strings.Split(httpRepo, ":")
		port := "80"
		if len(httpRepoArray) == utils.HttpUrlFields {
			port = httpRepoArray[1]
		}

		yumRepo = v1beta1.Repo{
			Domain: configinit.DefaultYumRepo,
			Port:   port,
		}

		if net.ParseIP(httpRepoArray[0]) == nil {
			global.CustomExtra["httpIp"] = httpRepoArray[0]
			yumRepo.Ip = httpRepoArray[0]
		} else {
			yumRepo.Domain = httpRepoArray[0]
		}
	}

	return yumRepo
}

func (op *Options) createClusterConfigFile(data map[string]string, imageRepo, yumRepo, chartRepo v1beta1.Repo) {
	err := bkeconfig.SetKubernetesConfig(data, bkecommon.BKEClusterConfigFileName, "cluster-system")
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to generate the cluster configuration file %s", err.Error()))
		return
	}

	if op.File != "" {
		return
	}

	// 创建集群配置
	c := config.Options{
		Directory:       fmt.Sprintf("%s/cluster", global.Workspace),
		Product:         "fuyao-allinone",
		Domain:          op.Domain,
		ImageRepoPort:   op.ImageRepoPort,
		AgentHealthPort: op.AgentHealthPort,
	}
	c.Config(global.CustomExtra, imageRepo, yumRepo, chartRepo, op.NtpServer)

	log.BKEFormat(log.HINT, fmt.Sprintf("Run `bke cluster create -f %s/cluster/1master-cluster.yaml -n %s/cluster/1master-node.yaml`"+
		"command to deploy the cluster", global.Workspace, global.Workspace))
}

func versionLess(v1, v2 string) bool {
	num1 := strings.TrimPrefix(v1, "v")
	num2 := strings.TrimPrefix(v2, "v")

	parts1 := strings.Split(num1, ".")
	parts2 := strings.Split(num2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		if parts1[i] != parts2[i] {
			return parts1[i] < parts2[i]
		}
	}

	return len(parts1) < len(parts2)
}

func (op *Options) getStartKubernetesVersion() (string, error) {
	sourceRegistry := fmt.Sprintf("%s/mount/source_registry/files", global.Workspace)
	if !utils.Exists(sourceRegistry) {
		return configinit.DefaultKubernetesVersion, nil
	}
	entries, err := os.ReadDir(sourceRegistry)
	if err != nil {
		return "", err
	}
	arches := []string{"-arm64", "-amd64", "-x86_64", "-ppc64le", "-s390x"}
	re := regexp.MustCompile(`^kubectl-(v\d+\.\d+\.\d+(?:[-.][a-zA-Z0-9]+)*)$`)

	var versions []string

	for _, entry := range entries {
		base := entry.Name()
		for _, arch := range arches {
			if strings.HasSuffix(base, arch) {
				base = strings.TrimSuffix(base, arch)
				break
			}
		}
		matches := re.FindStringSubmatch(base)
		if len(matches) > 1 {
			versions = append(versions, matches[1])
		}
	}

	if len(versions) == 0 {
		return "", errors.New("no kubernetes version found")
	}

	sort.Slice(versions, func(i, j int) bool {
		return versionLess(versions[i], versions[j])
	})

	return versions[0], nil
}

func (op *Options) ProcessPatchFiles(patchesDir string) map[string]string {
	if _, err := os.Stat(patchesDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(patchesDir)
	if err != nil {
		return nil
	}

	patchFiles := make(map[string]string)

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			fmt.Printf("Warning: failed to get file info for %s: %v\n", entry.Name(), err)
			continue
		}

		if info.Size() == 0 {
			fmt.Printf("Warning: skipping empty file: %s\n", entry.Name())
			continue
		}
		version := extractVersionFromFilename(entry.Name())
		if version == "" {
			continue
		}
		fullPath := filepath.Join(patchesDir, entry.Name())
		// 根据文件内容构造数据，将patch内容写入到configmap，为了便于处理，统一使用 openfuyao-patch 命名空间
		bkeConfigMapKey := fmt.Sprintf("%s%s", utils.PatchKeyPrefix, version)
		patchConfigMapName := fmt.Sprintf("%s%s", utils.PatchValuePrefix, version)
		if err = bkeconfig.SetPatchConfig(version, fullPath, patchConfigMapName); err != nil {
			continue
		}
		patchFiles[bkeConfigMapKey] = patchConfigMapName
	}

	return patchFiles
}

func (op *Options) modifyPermission() {
	var workDir string
	if utils.IsFile("/opt/BKE_WORKSPACE") {
		f, err := os.ReadFile("/opt/BKE_WORKSPACE")
		if err == nil {
			workDir = string(f)
			workDir = strings.TrimSpace(workDir)
			workDir = strings.TrimRight(workDir, "\n")
			workDir = strings.TrimRight(workDir, "\r")
			workDir = strings.TrimRight(workDir, "\t")
		}
	}

	if os.Getenv("BKE_WORKSPACE") != "" {
		workDir = os.Getenv("BKE_WORKSPACE")
	}
	if workDir == "" {
		workDir = "/bke"
	}

	// modify permission: file 644 dir 755
	err := filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			log.BKEFormat(log.INFO, fmt.Sprintf("Walk %s err %v", workDir, err))
			return err
		}

		if info.IsDir() {
			err = os.Chmod(path, utils.DefaultDirPermission)
			if err != nil {
				log.BKEFormat(log.INFO, fmt.Sprintf("path %s mod dir permission err %v", path, err))
			}
		} else {
			err = os.Chmod(path, utils.DefaultFilePermission)
			if err != nil {
				log.BKEFormat(log.INFO, fmt.Sprintf("path %s mod file permission err %v", path, err))
			}
		}
		return nil
	})

	if err != nil {
		log.BKEFormat(log.INFO, fmt.Sprintf("workDir %s mod permission err %v", workDir, err))
	} else {
		log.BKEFormat(log.INFO, fmt.Sprintf("workDir %s mod permission success", workDir))
	}
}

func (op *Options) deployCluster() {
	if op.File == "" {
		return
	}
	log.BKEFormat(log.INFO, "Starting to deploy the cluster...")
	c := cluster.Options{
		File:      op.File,
		NtpServer: op.NtpServer,
	}
	c.Cluster()
}
