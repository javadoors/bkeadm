/******************************************************************
 * Copyright (c) 2024 Bocloud Technologies Co., Ltd.
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
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	configinit "gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/initialize"

	"gopkg.openfuyao.cn/bkeadm/pkg/cluster"
	"gopkg.openfuyao.cn/bkeadm/pkg/initialize"
	"gopkg.openfuyao.cn/bkeadm/utils"
)

var initOption initialize.Options
var confirm bool

func isValidImageName(name string) bool {
	if name == "" {
		return false
	}
	invalidPrefixes := []string{".", "-", "_"}
	invalidSuffixes := []string{".", "-", "_"}
	for _, prefix := range invalidPrefixes {
		if strings.HasPrefix(name, prefix) {
			return false
		}
	}
	for _, suffix := range invalidSuffixes {
		if strings.HasSuffix(name, suffix) {
			return false
		}
	}
	return true
}

func isValidImageFormat(image string) bool {
	if image == "" {
		return false
	}

	imagePattern := `^([a-zA-Z0-9][a-zA-Z0-9-_.]*/)?([a-zA-Z0-9][a-zA-Z0-9-_.]*/)*[a-zA-Z0-9][a-zA-Z0-9-_.]*(:[a-zA-Z0-9][a-zA-Z0-9-_.-]*)?$`
	matched, err := regexp.MatchString(imagePattern, image)
	if err != nil {
		return false
	}

	// Additional validation: check for invalid characters and structure
	// Image name should not contain consecutive dots or slashes
	if strings.Contains(image, "..") || strings.Contains(image, "//") {
		return false
	}

	// Image name should not start or end with dot, dash, or underscore
	parts := strings.Split(image, "/")
	if len(parts) > 0 {
		lastPart := parts[len(parts)-1]
		// Remove tag if present
		if strings.Contains(lastPart, ":") {
			lastPart = strings.Split(lastPart, ":")[0]
		}
		if !isValidImageName(lastPart) {
			return false
		}
	}

	return matched
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize the boot node",
	Long:  `Initialize the boot node, including node check, warehouse start, cluster installation, and so on`,
	Example: `
# Initialize the boot node
bke init 
# Initialize with console installation enabled
bke init --installConsole=true
# Initialize with console installation disabled
bke init --installConsole=false
# Initialize the boot node and deploy the management cluster
bke init --file bkecluster.yaml
# Download files from the specified image and install the cluster
bke init --otherRepo cr.openfuyao.cn/openfuyao/bke-online-installed:latest
# Use external mirror repositories and source repositories
bke init --otherRepo cr.openfuyao.cn/openfuyao --otherSource http://192.168.1.120:40080

bke init --otherRepo cr.openfuyao.cn/openfuyao --otherSource http://192.168.100.48:8080（端口号） 

`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if initOption.HostIP == "" {
			ip, _ := utils.GetOutBoundIP()
			initOption.HostIP = ip
		}
		if initOption.HostIP == "" {
			ip, _ := utils.GetIntranetIp()
			initOption.HostIP = ip
		}

		// 仅当用户未传入 --domain 时，才使用默认值
		if initOption.Domain == "" {
			initOption.Domain = configinit.DefaultImageRepo
		}
		// 同理，ImageRepoPort 也需要这样处理（如果需要）
		if initOption.ImageRepoPort == "" {
			initOption.ImageRepoPort = configinit.DefaultImageRepoPort
		}

		if initOption.File != "" {
			bkeCluster, err := cluster.NewBKEClusterFromFile(initOption.File)
			if err != nil {
				return err
			}
			imageRepo := bkeCluster.Spec.ClusterConfig.Cluster.ImageRepo
			imgPort := ":443"
			// 如果用户传入了端口号，则使用用户传入的端口号
			if len(imageRepo.Port) > 0 {
				imgPort = ":" + imageRepo.Port
			}
			// 优先使用 IP 地址判断
			if len(imageRepo.Ip) > 0 {
				// 如果传入的 IP 和本机 IP 不同，说明用户想要使用自定义的镜像仓库地址
				if imageRepo.Ip != initOption.HostIP {
					// 如果用户没有传入域名，且域名就是默认值，则使用 IP 地址拼接仓库前缀
					if imageRepo.Domain == "" || imageRepo.Domain == configinit.DefaultImageRepo {
						initOption.OtherRepo = imageRepo.Ip + imgPort + "/" + imageRepo.Prefix
						// 否则使用用户传入的域名拼接仓库前缀
					} else {
						initOption.OtherRepo = imageRepo.Domain + imgPort + "/" + imageRepo.Prefix
					}
				}
			} else {
				if len(imageRepo.Domain) > 0 {
					_, err := utils.LoopIP(imageRepo.Domain)
					if err != nil {
						return err
					}
					initOption.OtherRepo = imageRepo.Domain + imgPort + "/" + imageRepo.Prefix
				}
			}

			httpRepo := bkeCluster.Spec.ClusterConfig.Cluster.HTTPRepo
			httpPort := ":80"
			if len(httpRepo.Port) > 0 {
				httpPort = ":" + httpRepo.Port
			}
			if len(httpRepo.Ip) > 0 {
				if httpRepo.Ip != initOption.HostIP {
					initOption.OtherSource = "http://" + httpRepo.Ip + httpPort
				}
			} else {
				if len(httpRepo.Domain) > 0 {
					if net.ParseIP(httpRepo.Domain) == nil {
						initOption.OtherSource = "http://" + httpRepo.Domain + httpPort
					} else {
						_, err := utils.LoopIP(httpRepo.Domain)
						if err != nil {
							return err
						}
						initOption.OtherSource = "http://" + httpRepo.Domain + httpPort
					}
				}
			}
			if len(httpRepo.Port) > 0 {
				initOption.YumRepoPort = httpRepo.Port
			}

			chartRepo := bkeCluster.Spec.ClusterConfig.Cluster.ChartRepo
			chartPort := ":443"
			if len(chartRepo.Port) > 0 {
				chartPort = ":" + chartRepo.Port
			}
			if len(chartRepo.Ip) > 0 {
				if chartRepo.Domain != "" || chartRepo.Domain != configinit.DefaultChartRepo {
					initOption.OtherChart = "oci://" + configinit.DefaultChartRepo + "/" + chartRepo.Prefix
				} else {
					initOption.OtherChart = chartRepo.Ip + chartPort + "/" + chartRepo.Prefix
				}
			} else {
				if len(chartRepo.Domain) > 0 {
					_, err := utils.LoopIP(chartRepo.Domain)
					if err != nil {
						return err
					}
					initOption.OtherChart = chartRepo.Domain + chartPort + "/" + chartRepo.Prefix
				}
			}

			if len(bkeCluster.Spec.ClusterConfig.Cluster.NTPServer) > 0 {
				initOption.NtpServer = bkeCluster.Spec.ClusterConfig.Cluster.NTPServer
			}
		}
		if len(initOption.ClusterAPI) == 0 {
			initOption.ClusterAPI = "latest"
		}
		if len(initOption.OFVersion) == 0 {
			initOption.OFVersion = "latest"
		}
		if len(initOption.VersionUrl) == 0 {
			initOption.VersionUrl = utils.DefaultPatchDownURL
		}
		if len(initOption.NtpServer) == 0 {
			initOption.NtpServer = configinit.DefaultNTPServer
		}
		if len(initOption.Runtime) == 0 {
			initOption.Runtime = "docker"
		}

		if len(initOption.RuntimeStorage) == 0 {
			if initOption.Runtime == "docker" {
				initOption.RuntimeStorage = configinit.DefaultCRIDockerDataRootDir
			} else {
				initOption.RuntimeStorage = configinit.DefaultCRIContainerdDataRootDir
			}
		}

		fmt.Println(fmt.Sprintf("--hostIP:            %s", initOption.HostIP))
		fmt.Println(fmt.Sprintf("--domain:            %s", initOption.Domain))
		fmt.Println(fmt.Sprintf("--kubernetesPort:    %s", initOption.KubernetesPort))
		fmt.Println(fmt.Sprintf("--imageRepoPort:     %s", initOption.ImageRepoPort))
		fmt.Println(fmt.Sprintf("--yumRepoPort:       %s", initOption.YumRepoPort))
		fmt.Println(fmt.Sprintf("--chartRepoPort:     %s", initOption.ChartRepoPort))
		fmt.Println(fmt.Sprintf("--ntpServer:         %s", initOption.NtpServer))
		fmt.Println(fmt.Sprintf("--runtime:           %s", initOption.Runtime))
		fmt.Println(fmt.Sprintf("--runtimeStorage:    %s", initOption.RuntimeStorage))
		fmt.Println(fmt.Sprintf("--clusterAPI:        %s", initOption.ClusterAPI))
		fmt.Println(fmt.Sprintf("--oFVersion:         %s", initOption.OFVersion))
		fmt.Println(fmt.Sprintf("--versionUrl:        %s", initOption.VersionUrl))
		fmt.Println(fmt.Sprintf("--enableNTP:         %v", initOption.EnableNTP))
		fmt.Println(fmt.Sprintf("--agentHealthPort:   %v", initOption.AgentHealthPort))
		port, err := strconv.Atoi(initOption.AgentHealthPort)
		if err != nil || port < utils.MinPort || port > utils.MaxPort {
			return errors.New("The agent health port must be legal. ")
		}
		if initOption.ImageFilePath != "" {
			fmt.Println(fmt.Sprintf("--imageFilePath:     %v", initOption.ImageFilePath))
		}
		if initOption.File != "" {
			fmt.Println(fmt.Sprintf("--file:              %s", initOption.File))
		}
		if initOption.OnlineImage != "" {
			fmt.Println(fmt.Sprintf("--onlineImage:       %s", initOption.OnlineImage))
		}
		if initOption.OtherRepo != "" {
			fmt.Println(fmt.Sprintf("--otherRepo:         %s", initOption.OtherRepo))
		}
		if initOption.OtherSource != "" {
			fmt.Println(fmt.Sprintf("--otherSource:       %s", initOption.OtherSource))
		}

		if initOption.HostIP == "" {
			return errors.New("The host IP address must be set. ")
		}
		if initOption.KubernetesPort == "" {
			return errors.New("The kubernetes port must be set. ")
		}
		if initOption.Domain == "" {
			return errors.New("The host domain name is a must. ")
		}
		if initOption.ImageRepoPort == "" {
			return errors.New("The image server port must be set. ")
		}
		if initOption.ChartRepoPort == "" {
			return errors.New("The yum chart port must be set. ")
		}
		if initOption.YumRepoPort == "" {
			return errors.New("The yum server port must be set. ")
		}
		if !utils.IsNum(initOption.ImageRepoPort) {
			return errors.New("The image server port is not a number. ")
		}
		if !utils.IsNum(initOption.YumRepoPort) {
			return errors.New("The yum server port is not a number. ")
		}
		if !utils.IsNum(initOption.ChartRepoPort) {
			return errors.New("The chart server port is not a number. ")
		}
		if len(initOption.OnlineImage) > 0 {
			if !isValidImageFormat(initOption.OnlineImage) {
				return errors.New("the parameter `onlineImage` must be a valid Docker image reference format (e.g., registry/repository/image:tag)")
			}
		}
		if len(initOption.OtherSource) > 0 {
			if !strings.HasPrefix(initOption.OtherSource, "http://") {
				return errors.New("The parameter `otherSource` must start with http:// ")
			}
		}
		if initOption.ImageFilePath != "" {
			if initOption.InstallConsole {
				return errors.New("local image file and install console is mutually exclusive. ")
			}
			// offline deploy contains all local image, so two is exclusive.
			if initOption.OtherRepo == "" {
				return errors.New("local image file and offline deploy is mutually exclusive. ")
			}
			filename := strings.ToLower(initOption.ImageFilePath)
			if !strings.HasSuffix(filename, ".tar.gz") && !strings.HasSuffix(filename, ".tgz") {
				return errors.New("local image file is not tar.gz or tgz format")
			}
			_, err := os.Stat(initOption.ImageFilePath)
			if err != nil {
				return errors.New("local image file not exist")
			}
		}
		if !utils.PromptForConfirmation(confirm) {
			return fmt.Errorf("operation cancelled by user")
		}
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		initOption.Args = args
		initOption.Options = options
		initOption.Initialize()
	},
}

func registerInitCommand() {
	rootCmd.AddCommand(initCmd)
	// Here you will define your flags and configuration settings.
	initCmd.Flags().StringVarP(&initOption.File, "file", "f", "", "bkecluster.yaml")
	initCmd.Flags().StringVar(&initOption.Domain,
		"domain", configinit.DefaultImageRepo, "The domain name of the host is customized")
	initCmd.Flags().StringVar(&initOption.ImageRepoPort,
		"imageRepoPort", configinit.DefaultImageRepoPort, "image repository port")
	initCmd.Flags().StringVar(&initOption.HostIP, "hostIP", "", "local kubernetes api server")
	initCmd.Flags().StringVar(&initOption.KubernetesPort,
		"kubernetesPort", utils.DefaultKubernetesPort, "local kubernetes port")
	initCmd.Flags().StringVar(&initOption.YumRepoPort,
		"yumRepoPort", configinit.DefaultYumRepoPort, "yum repository port")
	initCmd.Flags().StringVar(&initOption.ChartRepoPort,
		"chartRepoPort", utils.DefaultChartRegistryPort, "chart repository port")
	initCmd.Flags().StringVar(&initOption.NtpServer,
		"ntpServer", configinit.DefaultNTPServer, "value is `local`, an ntp service is started on the host")
	initCmd.Flags().StringVar(&initOption.Runtime, "runtime", "containerd", "docker/containerd")
	initCmd.Flags().StringVarP(&initOption.RuntimeStorage,
		"runtimeStorage", "s", "", "default `/var/lib/docker or /var/lib/containerd`")

	// online installation mode
	initCmd.Flags().StringVar(&initOption.OnlineImage, "onlineImage", "", "Online image to provide binary files and os rpm")
	initCmd.Flags().StringVar(&initOption.OtherRepo, "otherRepo", "", "Provides the private source address for image mirror installation")
	initCmd.Flags().StringVar(&initOption.OtherSource, "otherSource", "", "Provides the private source address for system package installation")
	initCmd.Flags().StringVar(&initOption.OtherChart, "otherChart", "", "Provides the private chart address for helm chart installation")

	initCmd.Flags().StringVar(&initOption.ClusterAPI, "clusterAPI", "latest", "cluster-api-bke version")
	initCmd.Flags().BoolVar(&confirm, "confirm", false, "Skip the confirmation")
	initCmd.Flags().StringVarP(&initOption.OFVersion,
		"oFVersion", "v", "latest", "openfuyao version")
	initCmd.Flags().StringVar(&initOption.VersionUrl,
		"versionUrl", utils.DefaultPatchDownURL, "online openfuyao version config download url")

	initCmd.Flags().BoolVar(&initOption.InstallConsole,
		"installConsole", true, "Whether to install bkeconsole (default: true)")
	initCmd.Flags().BoolVar(&initOption.EnableNTP,
		"enableNTP", true, "enable or disable NTP service, default is true")

	initCmd.Flags().BoolVar(&initOption.ImageRepoTLSVerify,
		"imageRepoTLSVerify", true, "Enable TLS verification for image repository")
	initCmd.Flags().StringVar(&initOption.ImageRepoCAFile,
		"imageRepoCAFile", "", "CA certificate file for image repository")
	initCmd.Flags().StringVar(&initOption.ImageRepoUsername,
		"imageRepoUsername", "", "Username for image repository authentication")
	initCmd.Flags().StringVar(&initOption.ImageRepoPassword,
		"imageRepoPassword", "", "Password for image repository authentication")

	initCmd.Flags().StringVar(&initOption.ImageFilePath, "imageFilePath", "", "image file path")
	initCmd.Flags().StringVar(&initOption.AgentHealthPort, "agentHealthPort", utils.DefaultAgentHealthPort, "agent listen health port")
}
