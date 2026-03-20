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

package reset

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/shirou/gopsutil/v3/host"
	bkeinit "gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/initialize"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

var needDeleteFile = []string{}

const (
	dockerCleanKubelet = "docker ps -a | grep kubelet | awk '{print $1}' | xargs docker rm -f --volumes"
	// todo docker container clean
	// rm all containers in k8s.io namespace
	nerdctlCleanContainer = "nerdctl -n k8s.io ps -a | grep -v CONTAINER | awk '{print $1}' | xargs nerdctl -n k8s.io rm -f -v"
	// umount all mount points in k8s.io namespace
	kubeletDirUnmont = "for m in $(sudo tac /proc/mounts | sudo awk '{print $2}'|sudo grep /var/lib/kubelet);do   sudo umount $m||true;   done"

	// nerdctl only for kubelet container
	nerdctlStopKubelet        = "nerdctl -n k8s.io ps -a | grep kubelet | awk '{print $1}' | xargs nerdctl -n k8s.io stop"
	nerdctlRemoveKubelet      = "nerdctl -n k8s.io ps -a | grep kubelet | awk '{print $1}' | xargs nerdctl -n k8s.io rm --volumes"
	nerdctlForceRemoveKubelet = "nerdctl -n k8s.io ps -a | grep kubelet | awk '{print $1}' | xargs nerdctl -n k8s.io rm -f --volumes"

	// docker only for kubelet container
	dockerStopKubelet        = "docker ps -a | grep kubelet | awk '{print $1}' | xargs docker stop"
	dockerRemoveKubelet      = "docker ps -a | grep kubelet | awk '{print $1}' | xargs docker rm --volumes"
	dockerForceRemoveKubelet = "docker ps -a | grep kubelet | awk '{print $1}' | xargs docker rm -f --volumes"

	// docker for all k8s containers
	dockerListContainers  = "docker ps -a --filter name=k8s_ -q | grep -v kubelet"
	dockerStopContainer   = "docker stop"
	dockerRemoveContainer = "docker rm --volumes"
	dockerForceRemovePod  = "docker rm -f --volumes"
	// docker 清理所有数据
	dockerCleanAll = "docker system prune -a -f --volumes"
	// docker 列出所有容器
	dockerListAllContainers = "docker ps -a -q"

	// crictl for all containers
	crictlListContainers  = "crictl pods -q"
	crictlStopContainer   = "crictl stopp"
	crictlRemoveContainer = "crictl rmp"
	crictlForceRemovePod  = "crictl rmp -f"
	// crictl 删除所有容器
	crictlCleanAllContainer = "crictl rmp -a -f"

	// nerdctl for all containers
	nerdctlListContainers       = "nerdctl ps -a -q"
	nerdctlForceRemoveContainer = "nerdctl rm -f --volumes"
	// nerdctl 清理所有数据
	nerdctlCleanAll = "nerdctl --namespace k8s.io system prune -a -f --volumes && nerdctl system image prune -a -f"

	// StopKubeletBin停止二进制安装的kubelet
	StopKubeletBin = "systemctl stop kubelet  && systemctl disable kubelet && rm -f /etc/systemd/system/kubelet.service"

	// RemoveKubeletBin卸载二进制安装的kubelet
	RemoveKubeletBin = "sudo rm -f $(which kubelet)"

	// ForceRemoveKubeletBin强制删除
	ForceRemoveKubeletBin = "sudo rm -rf /var/lib/kubelet " +
		"&& sudo rm -rf /usr/bin/kubelet " +
		"&& sudo rm -rf /etc/systemd/system/kubelet.service"
)

func cleanIpRoute() {
	// clean ip route about boc0 and cali
	cmd := `ip route show all | awk '($1 != "default" && $3 ~ /^(boc0|cali.*)$/) {printf "%s %s %s\n", $1, $2, $3}'`
	output, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", cmd)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("get route failed: %v", err))
		log.Debugf("get route output: %s", output)
		return
	}
	routes := strings.Split(output, "\n")
	for _, route := range routes {
		// delete route
		if route != "" {
			output, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c",
				"ip route del "+route)
			if err != nil {
				log.Debugf("delete route output: %s", output)
			}
		}
	}
}

func cleanIpNeighbor() {
	// clean ip neighbor about boc0 and cali
	cmd := `ip neigh show all | awk '($3~/^(boc0|cali.*)$/) {print}' | awk '$4 == "lladdr" || 
$4 == "FAILED" {printf "%s %s %s\n", $1, $2, $3}'`
	output, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", cmd)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("delete neighbor failed: %v", err))
		log.Debugf("delete neighbor output: %s", output)
		return
	}
	neighbors := strings.Split(output, "\n")
	for _, neighbor := range neighbors {
		// delete neighbor
		if neighbor != "" {
			output, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c",
				"ip neigh del "+neighbor)
			if err != nil {
				log.Debugf("delete neighbor output: %s", output)
			}
		}
	}
}

func cleanIpLink() {
	needRemoveInters := []string{"vxlan_sys_4789", "gre_sys", "genev_sys_6081", "erspan_sys", "vxlan.calico"}
	cmd := `ip link | awk '/state/ {gsub(/:/, ""); print $2}'`
	output, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", cmd)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("get interface failed: %v", err))
		log.Debugf("get interface output: %s", output)
		return
	}
	inters := strings.Split(output, "\n")
	for _, inter := range inters {
		if utils.ContainsString(needRemoveInters, inter) {
			log.BKEFormat(log.INFO, fmt.Sprintf("down interface: %s", inter))
			output, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c",
				"ip link set "+inter+" down")
			if err != nil {
				log.Debugf("down interface output: %s", output)
			}
			log.BKEFormat(log.INFO, fmt.Sprintf("delete interface: %s", inter))
			output, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c",
				"ip link del "+inter)
			if err != nil {
				log.Debugf("delete interface output: %s", output)
			}
		}
	}
}

func cleanNetwork() {
	log.BKEFormat(log.INFO, "clean network...")
	// clean iptables rule
	cmd := "iptables -F -t raw && iptables -F -t filter && iptables -t nat -F && iptables -t mangle -F && iptables -X -t nat && iptables -X -t raw && iptables -X -t mangle && iptables -X -t filter"
	output, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", cmd)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("clean iptables rule failed: %v", err))
		log.Debugf("clean iptables rule output: %s", output)
	}

	// clean ip route
	cleanIpRoute()

	// clean ip neighbor about boc0 and cali
	cleanIpNeighbor()

	// clean virtual interface about calico and fabric
	// fabric interface name vxlan_sys_4789 gre_sys genev_sys_6081 erspan_sys
	// calico interface name
	cleanIpLink()

	output, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", "ip link del nerdctl0")
	if err != nil {
		log.Debugf("delete interface output: %s", output)
	}

	output, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", "ip link del docker0")
	if err != nil {
		log.Debugf("delete interface output: %s", output)
	}

	needDeleteFile = append(needDeleteFile, []string{
		"/etc/kubernetes",
	}...)
}

func cleanKubeletBin() {
	log.BKEFormat(log.INFO, "clean kubelet...")
	var (
		err error
		out string
	)

	out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", StopKubeletBin)
	if err != nil {
		log.Debugf("stop kubelet container output: %s", out)
	} else {
		out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", RemoveKubeletBin)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("remove kubelet container failed: %v", err))
			log.Debugf("remove kubelet container output: %s", out)
		}
	}
	if err != nil {
		out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", ForceRemoveKubeletBin)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("force remove kubelet container failed: %v", err))
			log.Debugf("force remove kubelet container output: %s", out)
		}
	}

	needDeleteFile = append(needDeleteFile, []string{
		"/etc/cni/net.d/10-calico.conflist",
		"/etc/kubernetes",
	}...)
}

func cleanKubelet() {
	log.BKEFormat(log.INFO, "clean kubelet...")
	rootDir := bkeinit.DefaultKubeletRootDir

	unmountKubeletDirectories(rootDir)
	cleanKubeletContainer()
	appendKubeletCleanupFiles(rootDir)
}

// unmountKubeletDirectories 卸载 kubelet 相关目录
func unmountKubeletDirectories(rootDir string) {
	if err := unmountKubeletDirectory(rootDir); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("umount kubelet directory failed: %v", err))
	}

	cmd := fmt.Sprintf("for m in $(sudo tac /proc/mounts | "+
		"sudo awk '{print $2}'|sudo grep %s | grep -v %s);do   sudo umount $m||true;   done", rootDir, rootDir)
	out, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", cmd)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("umount kubelet directory failed: %v", err))
		log.Debugf("umount kubelet directory output: %s", out)
	}
}

// cleanKubeletContainer 清理 kubelet 容器
func cleanKubeletContainer() {
	if infrastructure.IsDocker() {
		cleanDockerKubelet()
	}
	if infrastructure.IsContainerd() {
		cleanContainerdKubelet()
	}
}

// cleanDockerKubelet 使用 Docker 清理 kubelet 容器
func cleanDockerKubelet() {
	out, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", dockerStopKubelet)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("stop kubelet container failed: %v", err))
		log.Debugf("stop kubelet container output: %s", out)
	} else {
		out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", dockerRemoveKubelet)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("remove kubelet container failed: %v", err))
			log.Debugf("remove kubelet container output: %s", out)
		}
	}
	if err != nil {
		out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", dockerForceRemoveKubelet)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("force remove kubelet container failed: %v", err))
			log.Debugf("force remove kubelet container output: %s", out)
		}
	}
}

// cleanContainerdKubelet 使用 Containerd 清理 kubelet 容器
func cleanContainerdKubelet() {
	out, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", nerdctlStopKubelet)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("stop kubelet container failed: %v", err))
		log.Debugf("stop kubelet container output: %s", out)
	} else {
		out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", nerdctlRemoveKubelet)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("remove kubelet container failed: %v", err))
			log.Debugf("remove kubelet container output: %s", out)
		}
	}
	if err != nil {
		out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", nerdctlForceRemoveKubelet)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("force remove kubelet container failed: %v", err))
			log.Debugf("force remove kubelet container output: %s", out)
		}
	}
}

// appendKubeletCleanupFiles 添加 kubelet 需要删除的文件
func appendKubeletCleanupFiles(rootDir string) {
	needDeleteFile = append(needDeleteFile, rootDir+"/pki")
	needDeleteFile = append(needDeleteFile, rootDir)
	needDeleteFile = append(needDeleteFile, []string{
		rootDir + "/pki",
		rootDir,
		"/etc/cni/net.d/10-calico.conflist",
		"/etc/cni/net.d/boc.conflist",
		"/etc/kubernetes",
	}...)
}

func cleanKubernetesContainer() {
	log.BKEFormat(log.INFO, "clean kubernetes container...")

	cleanDockerK8sContainers()
	cleanContainerdK8sContainers()

	rootDir := bkeinit.DefaultKubeletRootDir
	unmountKubeletDirectories(rootDir)
	needDeleteFile = append(needDeleteFile, rootDir)
}

// cleanDockerK8sContainers 清理 Docker 中的 K8s 容器
func cleanDockerK8sContainers() bool {
	if !infrastructure.IsDocker() {
		return false
	}
	out, err := global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", dockerListContainers)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("list containers failed: %s", err))
		log.Debugf("list containers output: %s", out)
		return false
	}
	pods := strings.Fields(out)
	for _, pod := range pods {
		removeDockerContainer(pod)
	}
	return true
}

// removeDockerContainer 移除单个 Docker 容器
func removeDockerContainer(pod string) {
	out, err := global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", dockerStopContainer+" "+pod)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("stop container failed: %s", err))
		log.Debugf("stop container output: %s", out)
	} else {
		out, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", dockerRemoveContainer+" "+pod)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("remove container failed: %s", err))
			log.Debugf("remove container output: %s", out)
		}
	}
	if err != nil {
		out, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", dockerForceRemovePod+" "+pod)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("force remove container failed: %s", err))
			log.Debugf("force remove container output: %s", out)
		}
	}
}

// cleanContainerdK8sContainers 清理 Containerd 中的 K8s 容器
func cleanContainerdK8sContainers() {
	if !infrastructure.IsContainerd() {
		return
	}
	out, err := global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", crictlListContainers)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("list containers failed: %s", err))
		log.Debugf("list containers output: %s", out)
		return
	}
	pods := strings.Fields(out)
	for _, pod := range pods {
		removeContainerdContainer(pod)
	}
}

// removeContainerdContainer 移除单个 Containerd 容器
func removeContainerdContainer(pod string) {
	out, err := global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", crictlStopContainer+" "+pod)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("stop container %s failed: %s", pod, err))
		log.Debugf("stop container %s output: %s", pod, out)
	} else {
		out, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", crictlRemoveContainer+" "+pod)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("remove container %s failed: %s", pod, err))
			log.Debugf("remove container %s output: %s", pod, out)
		}
	}
	if err != nil {
		out, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", crictlForceRemovePod+" "+pod)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("force remove container %s failed: %s", pod, err))
			log.Debugf("force remove container %s output: %s", pod, out)
		}
	}
}

func cleanOtherContainer() {
	log.BKEFormat(log.INFO, "clean other container...")
	cleanDockerAllContainers()
	cleanContainerdAllContainers()
}

// cleanDockerAllContainers 清理所有 Docker 容器
func cleanDockerAllContainers() {
	if !infrastructure.IsDocker() {
		return
	}
	out, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", dockerListAllContainers)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("list all containers failed: %v", err))
		log.Debugf("list all containers output: %s", out)
		return
	}
	pods := strings.Fields(out)
	for _, pod := range pods {
		out, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", dockerForceRemovePod+" "+pod)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("force remove container failed: %s", err))
			log.Debugf("force remove container %s output: %s", pod, out)
		}
	}
	out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", dockerCleanAll)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("clean docker failed: %v", err))
		log.Debugf("clean docker output: %s", out)
	}
}

// cleanContainerdAllContainers 清理所有 Containerd 容器
func cleanContainerdAllContainers() {
	if !infrastructure.IsContainerd() {
		return
	}
	cleanCrictlContainers()
	cleanNerdctlContainers()
}

// cleanCrictlContainers 使用 crictl 清理容器
func cleanCrictlContainers() {
	out, err := global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", crictlListContainers)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("list containers failed: %s", err))
		log.Debugf("crictl list containers output: %s", out)
		return
	}
	pods := strings.Fields(out)
	for _, pod := range pods {
		out, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", crictlForceRemovePod+" "+pod)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("force remove container %s failed: %s", pod, err))
			log.Debugf("force remove container %s output: %s", pod, out)
		}
	}
}

// cleanNerdctlContainers 使用 nerdctl 清理容器
func cleanNerdctlContainers() {
	out, err := global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", nerdctlListContainers)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("list containers failed: %s", err))
		log.Debugf("nerdctl list containers output: %s", out)
		return
	}
	pods := strings.Fields(out)
	for _, pod := range pods {
		out, err = global.Command.ExecuteCommandWithOutput("/bin/sh", "-c", nerdctlForceRemoveContainer+" "+pod)
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("force remove container %s failed: %s", pod, err))
			log.Debugf("force remove container %s output: %s", pod, out)
		}
	}
}

func disableDocker() {
	output, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", "systemctl stop docker")
	if err != nil {
		log.Debugf("stop docker output: %s, %s", output, err.Error())
	}
	output, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", "systemctl disable docker")
	if err != nil {
		log.Debugf("disable docker output: %s, %s", output, err.Error())
	}
}

func disableContainerd() {
	out, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", "systemctl stop containerd")
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("stop containerd failed: %v", err))
		log.Debugf("stop containerd output: %s, %s", out, err.Error())
	}
	out, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", "systemctl disable containerd")
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("disable containerd failed: %v", err))
		log.Debugf("disable containerd output: %s, %s", out, err.Error())
	}
}

func addNeedDeleteFile() {
	needDeleteFile = append(needDeleteFile, []string{
		bkeinit.DefaultCRIDockerDataRootDir,
		"/etc/docker",
		"/var/lib/cni",
		"/etc/cni",
		"/opt/cni",
	}...)

	needDeleteFile = append(needDeleteFile, []string{
		"/usr/bin/containerd",
		"/usr/bin/containerd-shim",
		"/usr/bin/containerd-shim-runc-v1",
		"/usr/bin/containerd-shim-runc-v2",
		"/usr/bin/crictl",
		"/etc/crictl.yaml",
		"/usr/bin/ctr",
		"/usr/bin/nerdctl",
		"/usr/bin/containerd-stress",
		"/usr/lib/systemd/system/containerd.service",
		"/usr/local/sbin/runc",
		"/etc/containerd",
		"/usr/local/beyondvm",
		bkeinit.DefaultCRIContainerdDataRootDir,
		"/var/lib/cni",
		"/etc/cni",
		"/opt/cni",
		"/var/lib/nerdctl",
		"/etc/docker/certs.d",
		"/usr/bin/docker",
		"/usr/bin/dockerd",
		"/usr/bin/docker-init",
		"/usr/bin/docker-proxy",
		"/usr/bin/runc",
		"/usr/lib/systemd/system/docker.service",
		"/etc/systemd/system/containerd.service.d",
	}...)
}

func cleanContainerRuntime() {
	log.BKEFormat(log.INFO, "clean container runtime...")

	needRemoveDocker := false
	if infrastructure.IsDocker() || infrastructure.IsContainerd() {
		if infrastructure.IsDocker() {
			needRemoveDocker = true
		}

		if infrastructure.IsDocker() {
			disableDocker()
		}

		if infrastructure.IsContainerd() {
			disableContainerd()
		}
	}

	if needRemoveDocker {
		//  remove docker-ce docker-ce-cli containerd.io
		if err := repoRemove("docker*", "containerd.io"); err != nil {
			log.Errorf("remove docker failed: %v", err)
		}
	}

	addNeedDeleteFile()
}

func cleanNeedDeleteFile() {
	log.BKEFormat(log.INFO, "clean file...")
	needDeleteFile = append(needDeleteFile, []string{
		"/etc/openFuyao/haproxy",
		"/etc/openFuyao/keepalived",
		"/usr/bin/calicoctl",
		"/etc/calico",
		"/usr/bin/kubectl",
		"/etc/openFuyao/addons",
		"/etc/openFuyao/bkeagent",
		"/usr/local/bin/bkeagent",
		fmt.Sprintf("%s/.kube/config", os.Getenv("HOME")),
	}...)

	for _, f := range needDeleteFile {
		err := removeDir(f)
		if err != nil {
			log.Debugf("remove dir %s failed: %v", f, err)
			err = removeFile(f)
			if err != nil {
				log.Debugf("remove file %s failed: %v", f, err)
			}
		}
	}
}

// removeDir removes everything in a directory, and contains the directory itself
func removeDir(filePath string) error {
	// If the directory doesn't even exist there's nothing to do, and we do not consider this an error
	if s, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	} else if !s.IsDir() {
		return errors.New(fmt.Sprintf("path %s is not a directory", filePath))
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	// Read the names of all files and directories in the directory
	dirnames, err := f.Readdirnames(-1)
	if err != nil {
		return err
	}
	// Remove all files and directories in the directory
	for _, name := range dirnames {
		if err = os.RemoveAll(filepath.Join(filePath, name)); err != nil {
			return err
		}
	}
	return os.Remove(filePath) // delete directory itself
}

func removeFile(filePath string) error {
	if s, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	} else if s.IsDir() {
		return errors.New(fmt.Sprintf("path %s is a directory", filePath))
	}
	return os.Remove(filePath)
}

// unmountKubeletDirectory unmounts all directories that are mounted under the kubelet run directory.
func unmountKubeletDirectory(absoluteKubeletRunDirectory string) error {
	raw, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return err
	}

	if !strings.HasSuffix(absoluteKubeletRunDirectory, "/") {
		// "/" 用于确保路径以斜杠结尾，防止匹配例如 "/var/lib/kubelet"
		absoluteKubeletRunDirectory += "/"
	}

	mounts := strings.Split(string(raw), "\n")
	for _, mount := range mounts {
		//转义空格
		mount = strings.ReplaceAll(mount, `\040`, " ")
		m := strings.Split(mount, " ")

		if len(m) < utils.MatchFields || !strings.HasPrefix(m[1], absoluteKubeletRunDirectory) {
			continue
		}

		// 排除absoluteKubeletRunDirectory本身
		if m[1] == absoluteKubeletRunDirectory || m[1] == absoluteKubeletRunDirectory[:len(absoluteKubeletRunDirectory)-1] {
			continue
		}

		if err = syscall.Unmount(m[1], 0); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to unmount mounted directory in %s: %s, err: %v", absoluteKubeletRunDirectory, m[1], err))
		}
	}
	return nil
}

func repoRemove(packages ...string) error {
	packageManager := "unknown"
	h, _, _, err := host.PlatformInformation()
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("get host platform information failed, err: %v", err))
	}
	switch h {
	case "ubuntu", "debian":
		packageManager = "apt"
	case "centos", "kylin", "redhat", "fedora", "openeuler":
		packageManager = "yum"
	default:
		packageManager = "unknown"
	}
	output, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", fmt.Sprintf("%s remove -y %s", packageManager, strings.Join(packages, " ")))
	if err != nil {
		return errors.New(fmt.Sprintf("remove packages %q failed, err: %v, out: %s", strings.Join(packages, " "), err, output))
	}
	return nil
}
