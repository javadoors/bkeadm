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

package server

import (
	"fmt"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/go-connections/nat"

	econd "gopkg.openfuyao.cn/bkeadm/pkg/executor/containerd"
	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure/k3s"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func StartImageRegistry(name, image, imageRegistryPort, imageDataDirectory string) error {
	if infrastructure.IsContainerd() && !infrastructure.IsDocker() {
		return startImageRegistryWithContainerd(name, image, imageRegistryPort, imageDataDirectory)
	}

	if !infrastructure.IsDocker() {
		log.BKEFormat(log.ERROR, "docker or containerd runtime not found.")
		return nil
	}
	return startImageRegistryWithDocker(name, image, imageRegistryPort, imageDataDirectory)
}

// startImageRegistryWithDocker 使用 Docker 启动镜像仓库服务
func startImageRegistryWithDocker(name, image, imageRegistryPort, imageDataDirectory string) error {
	certPath := fmt.Sprintf("/etc/docker/%s", name)
	if err := generateConfig(certPath, imageRegistryPort); err != nil {
		log.BKEFormat(log.ERROR, "Failed to generate config.")
		return err
	}
	if err := global.Docker.EnsureImageExists(docker.ImageRef{Image: image},
		utils.RetryOptions{MaxRetry: 3, Delay: 1}); err != nil {
		return err
	}
	serverRunFlag, err := global.Docker.EnsureContainerRun(name)
	if err != nil {
		return err
	}
	if serverRunFlag {
		log.BKEFormat(log.INFO, "The mirror warehouse service is already running. ")
		return nil
	}

	if err := runDockerImageRegistry(name, image, imageRegistryPort, imageDataDirectory, certPath); err != nil {
		return err
	}

	waitForDockerContainerRunning(name, "container")
	return nil
}

// runDockerImageRegistry 运行 Docker 镜像仓库容器
func runDockerImageRegistry(name, image, imageRegistryPort, imageDataDirectory, certPath string) error {
	err := global.Docker.Run(
		&container.Config{
			Image:        image,
			ExposedPorts: map[nat.Port]struct{}{"443/tcp": {}},
			Env:          []string{"GODEBUG=x509ignoreCN=0"},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{Type: mount.TypeBind, Source: imageDataDirectory, Target: "/var/lib/registry"},
				{Type: mount.TypeBind, Source: certPath, Target: "/etc/docker/registry"},
			},
			PortBindings: map[nat.Port][]nat.PortBinding{
				nat.Port("443/tcp"): {{HostIP: "0.0.0.0", HostPort: imageRegistryPort}},
			},
			RestartPolicy: container.RestartPolicy{Name: "always", MaximumRetryCount: 0},
		}, nil, nil, name)
	if err != nil {
		log.BKEFormat(log.WARN, "The image repository service fails to be deployed")
		return err
	}
	return nil
}

func startImageRegistryWithContainerd(name, image, imageRegistryPort, imageDataDirectory string) error {
	certPath := fmt.Sprintf("%s/%s", k3s.DefaultK3sDataDir, name)
	err := generateConfig(certPath, imageRegistryPort)
	if err != nil {
		log.BKEFormat(log.ERROR, "Failed to generate config.")
		return err
	}
	err = econd.EnsureImageExists(image)
	if err != nil {
		return err
	}
	serverRunFlag, err := econd.EnsureContainerRun(name)
	if err != nil {
		return err
	}
	// 服务已经运行
	if serverRunFlag {
		log.BKEFormat(log.INFO, "The mirror warehouse service is already running. ")
		return nil
	}
	script := []string{
		"run", "-d", fmt.Sprintf("--name=%s", name),
		"-p", fmt.Sprintf("%s:443", imageRegistryPort),
		"--restart=always", fmt.Sprintf("--ip=10.4.0.%d", utils.RandInt(utils.MinRegistryIp, utils.MaxRegistryIp)),
		"-v", fmt.Sprintf("%s:/var/lib/registry", imageDataDirectory),
		"-v", fmt.Sprintf("%s:/etc/docker/registry", certPath),
		image,
	}
	err = econd.Run(script)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("The image repository service fails to be deployed by containerd: %s", err))
		return err
	}

	for {
		log.BKEFormat(log.INFO, "Wait for the container mirroring service to start...")
		time.Sleep(utils.ContainerStartWaitSeconds * time.Second)
		info, err := econd.ContainerInspect(name)
		log.Debugf(info.Name, info.State)
		if err != nil {
			continue
		}
		if info.State.Running {
			break
		}
	}
	log.BKEFormat(log.INFO, "The container mirroring service is started by containerd. ")
	return nil
}

func RemoveImageRegistry(name string) error {
	log.BKEFormat(log.INFO, "Remove the image repository")
	if err := os.RemoveAll("/etc/docker/" + name); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove /etc/docker/%s: %v", name, err))
	}
	return removeContainerWithRetry(name, nil)
}

func generateConfig(certPath, port string) error {
	err := SetRegistryConfig(certPath)
	if err != nil {
		return err
	}
	err = SetServerCertificate(certPath)
	if err != nil {
		return err
	}
	err = SetClientCertificate(certPath, port)
	if err != nil {
		return err
	}
	err = SetClientLocalCertificate(certPath, port)
	if err != nil {
		return err
	}
	return nil
}
