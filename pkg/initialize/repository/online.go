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

package repository

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	bkeinit "gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/initialize"
	"gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/validation"
	"gopkg.openfuyao.cn/cluster-api-provider-bke/common/warehouse"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

var (
	sourceRegistry = fmt.Sprintf("%s/mount/source_registry/files", global.Workspace)
)

type OtherRepo struct {
	Repo        string `json:"repo"`
	RepoIP      string `json:"repoIP"`
	Image       string `json:"image"`
	Source      string `json:"source"`
	ChartRepo   string `json:"chartRepo"`   // 新增
	ChartRepoIP string `json:"chartRepoIP"` // 新增
}

func ParseOnlineConfig(domain, image, repo, source, chartRepo string) (OtherRepo, error) {
	o := OtherRepo{}
	if len(image) > 0 {
		o.Image = image
	}
	if len(source) > 0 {
		o.Source = source
	}
	if len(repo) > 0 {
		o.Repo = repo
		repos := strings.Split(repo, "/")
		repos2 := strings.Split(repos[0], ":")
		if net.ParseIP(repos2[0]) != nil {
			o.RepoIP = repos2[0]
			o.Repo = strings.Replace(o.Repo, o.RepoIP, domain, -1)
			o.Image = strings.Replace(o.Image, o.RepoIP, domain, -1)
		} else {
			ip4, err := utils.LoopIP(repos2[0])
			if err != nil {
				return o, err
			}
			if len(ip4) == 0 {
				return o, errors.New(fmt.Sprintf("Domain name resolution failure, %s", repos2[0]))
			}
			o.RepoIP = ip4[0]
		}
		if !strings.HasSuffix(o.Repo, "/") {
			o.Repo = o.Repo + "/"
		}
	}
	// 新增chartRepo解析
	if len(chartRepo) > 0 {
		o.ChartRepo = chartRepo
		repos := strings.Split(chartRepo, "/")
		repos2 := strings.Split(repos[0], ":")
		if net.ParseIP(repos2[0]) != nil {
			o.ChartRepoIP = repos2[0]
			o.ChartRepo = strings.Replace(o.ChartRepo, o.ChartRepoIP, domain, -1)
		} else {
			ip4, err := utils.LoopIP(repos2[0])
			if err != nil {
				return o, err
			}
			if len(ip4) == 0 {
				return o, errors.New(fmt.Sprintf("Domain name resolution failure, %s", repos2[0]))
			}
			o.ChartRepoIP = ip4[0]
		}
		if !strings.HasSuffix(o.ChartRepo, "/") {
			o.ChartRepo = o.ChartRepo + "/"
		}
	}
	return o, nil
}

// 从镜像仓库下载初始化源文件
func SourceInit(oc OtherRepo) error {
	if len(oc.Source) == 0 {
		return nil
	}
	if !utils.Exists(sourceRegistry) {
		err := os.MkdirAll(sourceRegistry, utils.DefaultDirPermission)
		if err != nil {
			return err
		}
	}
	err := sourceBaseFile(oc.Source)
	if err != nil {
		return err
	}
	err = sourceRuntime(oc.Source)
	if err != nil {
		return err
	}
	return nil
}

type CertificateConfig struct {
	TLSVerify    bool
	CAFile       string
	Username     string
	Password     string
	RegistryHost string // 新增：镜像仓库主机名
	RegistryPort string // 新增：镜像仓库端口
}

// 修改：根据实际域名设置CA证书（单向认证）
func SetupCACertificate(config *CertificateConfig) error {
	if config.CAFile == "" {
		return nil
	}

	// 定义需要处理的证书目录列表
	certDirs := []string{
		fmt.Sprintf("/etc/containerd/certs.d/%s:%s", config.RegistryHost, config.RegistryPort),
		fmt.Sprintf("/etc/containerd/certs.d/%s", config.RegistryHost),
	}

	// 循环处理所有目录：创建目录并复制CA证书
	for _, dir := range certDirs {
		// 创建目录（含父目录）
		if err := os.MkdirAll(dir, utils.DefaultDirPermission); err != nil {
			return err
		}
		// 复制CA证书到目标目录
		destPath := filepath.Join(dir, "ca.crt")
		if err := copyFile(config.CAFile, destPath); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	if err := dstFile.Chmod(utils.DefaultFilePermission); err != nil {
		log.Warnf("failed to set file permission for %s: %s", dst, err.Error())
	}

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// 新增：从镜像仓库URL解析主机名和端口
func ParseRegistryHostPort(imageRepo string) (host, port string) {
	if imageRepo == "" {
		return "", ""
	}

	// 移除协议前缀
	repo := strings.TrimPrefix(imageRepo, "http://")
	repo = strings.TrimPrefix(repo, "https://")

	// 分割主机和路径
	parts := strings.Split(repo, "/")
	if len(parts) == 0 {
		return "", ""
	}

	// 分割主机和端口
	hostPort := strings.Split(parts[0], ":")
	if len(hostPort) == 1 {
		// 没有端口，使用默认端口
		return hostPort[0], "443"
	} else if len(hostPort) == utils.HttpUrlFields {
		return hostPort[0], hostPort[1]
	}

	return "", ""
}

// 从镜像仓库下载YUM源数据
func RepoInit(oc OtherRepo, certConfig *CertificateConfig) error {
	if len(oc.Image) == 0 {
		return nil
	}
	if utils.Exists(yumDataFile) {
		return nil
	}
	if err := cleanTempYumDataFile(); err != nil {
		return err
	}

	registryHost, registryPort := ParseRegistryHostPort(oc.Image)
	certConfig.RegistryHost = registryHost
	certConfig.RegistryPort = registryPort

	od := buildDownloadOptions(oc, certConfig)
	if err := setupTLSCertificate(oc, certConfig, &od); err != nil {
		return err
	}

	log.BKEFormat(log.INFO, "Download source file...")
	if err := od.Download(); err != nil {
		return err
	}
	return finalizeYumDataFile()
}

// cleanTempYumDataFile 清理临时 yum 数据文件
func cleanTempYumDataFile() error {
	if utils.Exists(yumDataFile + ".temp") {
		return os.RemoveAll(yumDataFile + ".temp")
	}
	return nil
}

// buildDownloadOptions 构建下载选项
func buildDownloadOptions(oc OtherRepo, certConfig *CertificateConfig) registry.OptionsDownload {
	return registry.OptionsDownload{
		Options:             root.Options{},
		SrcTLSVerify:        certConfig.TLSVerify,
		Image:               oc.Image,
		Username:            certConfig.Username,
		Password:            certConfig.Password,
		CertDir:             certConfig.CAFile,
		DownloadToDir:       yumDataFile + ".temp",
		DownloadInImageFile: "source.tar.gz",
	}
}

// setupTLSCertificate 设置 TLS 证书
func setupTLSCertificate(oc OtherRepo, certConfig *CertificateConfig, od *registry.OptionsDownload) error {
	if certConfig.TLSVerify {
		if err := SetupCACertificate(certConfig); err != nil {
			return err
		}
		log.BKEFormat(log.INFO, "Using client certificate authentication(CA only)")
	}

	if strings.Contains(oc.Image, bkeinit.DefaultImageRepo) {
		od.SrcTLSVerify = true
		img1 := strings.Split(oc.Image, "/")
		img2 := strings.Split(img1[0], ":")
		if len(img2) != utils.HttpUrlFields {
			return errors.New(fmt.Sprintf("The domain name and port must be included, %s", oc.Image))
		}
		if err := warehouse.SetClientCertificate(img2[1]); err != nil {
			return err
		}
		od.CertDir = fmt.Sprintf("/etc/docker/certs.d/deploy.bocloud.k8s:%s", img2[1])
	}
	return nil
}

// finalizeYumDataFile 完成 yum 数据文件处理
func finalizeYumDataFile() error {
	if err := os.Rename(yumDataFile+".temp/source.tar.gz", yumDataFile); err != nil {
		return err
	}
	if err := os.RemoveAll(yumDataFile + ".temp"); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove temp directory: %v", err))
	}
	return nil
}

// 下载基础文件下载chart和nfs数据包
func sourceBaseFile(httpRepo string) error {
	if !utils.Exists(chartDataFile) {
		log.BKEFormat(log.INFO, fmt.Sprintf("download %s/files/chart.tar.gz", httpRepo))
		err := utils.DownloadFile(httpRepo+"/files/charts.tar.gz", chartDataFile)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to download chart.tar.gz package: %s", err))
		}
	}
	if !utils.Exists(nfsDataFile) {
		log.BKEFormat(log.INFO, fmt.Sprintf("download %s/files/nfsshare.tar.gz", httpRepo))
		if utils.Exists(nfsDataFile + ".temp") {
			err := os.RemoveAll(nfsDataFile + ".temp")
			if err != nil {
				return err
			}
		}
		err := utils.DownloadFile(httpRepo+"/files/nfsshare.tar.gz", nfsDataFile+".temp")
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to download nfsshare.tar.gz package: %s", err))
		}
		err = os.Rename(nfsDataFile+".temp", nfsDataFile)
		if err != nil {
			return err
		}
	}
	return nil
}

// downloadRuntime download containerd cni
func sourceRuntime(httpRepo string) error {
	exists, err := checkLocalRuntimeFilesExist()
	if err != nil {
		return err
	}
	if exists {
		return nil
	}

	filesURL := httpRepo + "/files/"
	files, err := fetchRemoteFileList(filesURL)
	if err != nil {
		return err
	}

	return downloadRuntimeFiles(filesURL, files.containerd, files.cni, files.kubectl)
}

// checkLocalRuntimeFilesExist 检查本地运行时文件是否已存在
func checkLocalRuntimeFilesExist() (bool, error) {
	containerdList := make([]string, 0)
	cniPluginList := make([]string, 0)
	kubectlList := make([]string, 0)

	entries, err := os.ReadDir(sourceRegistry)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "kubectl") {
			kubectlList = append(kubectlList, entry.Name())
			continue
		}
		if strings.HasPrefix(entry.Name(), utils.CniPluginPrefix) {
			cniPluginList = append(cniPluginList, entry.Name())
			continue
		}
		if err = validation.ValidateCustomExtra(map[string]string{"containerd": entry.Name()}); err != nil {
			continue
		}
		containerdList = append(containerdList, entry.Name())
	}

	return len(containerdList) > 0 && len(cniPluginList) > 0 && len(kubectlList) > 0, nil
}

// runtimeFiles 运行时文件集合
type runtimeFiles struct {
	containerd []string
	cni        []string
	kubectl    []string
}

// fetchRemoteFileList 从远程获取文件列表
func fetchRemoteFileList(filesURL string) (*runtimeFiles, error) {
	resp, err := http.Get(filesURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != utils.HTTPStatusOK {
		return nil, errors.New(fmt.Sprintf(" get url %s, status code %d", filesURL, resp.StatusCode))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlData := string(body)
	if len(htmlData) == 0 {
		return nil, errors.New(fmt.Sprintf("url: %s, Failed to get download list", filesURL))
	}

	return parseFileListFromHTML(htmlData)
}

// parseFileListFromHTML 从 HTML 内容解析文件列表
func parseFileListFromHTML(htmlData string) (*runtimeFiles, error) {
	re := regexp.MustCompile(`<a href="(.*?)">(.*?)</a>`)
	result := re.FindAllStringSubmatch(htmlData, -1)

	files := &runtimeFiles{
		containerd: make([]string, 0),
		cni:        make([]string, 0),
		kubectl:    make([]string, 0),
	}

	for _, res := range result {
		if len(res) < utils.MatchFields {
			continue
		}
		if strings.HasPrefix(res[1], "containerd") {
			files.containerd = append(files.containerd, res[1])
		}
		if strings.HasPrefix(res[1], utils.CniPluginPrefix) {
			files.cni = append(files.cni, res[1])
		}
		if strings.HasPrefix(res[1], "kubectl-") {
			files.kubectl = append(files.kubectl, res[1])
		}
	}
	return files, nil
}

// downloadRuntimeFiles 下载运行时文件
func downloadRuntimeFiles(filesURL string, containerd, cni, kubectl []string) error {
	for _, con := range containerd {
		log.BKEFormat(log.INFO, fmt.Sprintf("download %s", filesURL+con))
		if err := utils.DownloadFile(filesURL+con, sourceRegistry+"/"+con); err != nil {
			return err
		}
	}
	for _, cn := range cni {
		log.BKEFormat(log.INFO, fmt.Sprintf("download %s", filesURL+cn))
		if err := utils.DownloadFile(filesURL+cn, sourceRegistry+"/"+cn); err != nil {
			return err
		}
	}
	if len(kubectl) == 0 {
		return errors.New("no kubectl files found in http repo: " + filesURL)
	}
	for _, kubectlFile := range kubectl {
		log.BKEFormat(log.INFO, fmt.Sprintf("download %s", filesURL+kubectlFile))
		if err := utils.DownloadFile(filesURL+kubectlFile, sourceRegistry+"/"+kubectlFile); err != nil {
			return fmt.Errorf("failed to download kubectl file %s: %w", kubectlFile, err)
		}
	}
	return nil
}

func sourceRuntimeKylin(httpRepo string) error {
	httpRepo = httpRepo + "/files/"
	var result []string
	result = append(result, strings.Replace(utils.KylinDocker, "{.arch}", "arm64", -1))
	result = append(result, strings.Replace(utils.KylinDocker, "{.arch}", "amd64", -1))
	for _, res := range result {
		if !utils.Exists(res) {
			log.BKEFormat(log.INFO, fmt.Sprintf("download %s", httpRepo+res))
			err := utils.DownloadFile(httpRepo+res, sourceRegistry+"/"+res)
			if err != nil {
				log.Errorf("Failed to download %s: %s", res, err)
				return err
			}
		}
	}
	return nil
}
