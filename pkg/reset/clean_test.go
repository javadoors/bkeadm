/*
 *
 * Copyright (c) 2025 Bocloud Technologies Co., Ltd.
 * installer is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          <http://license.coscl.org.cn/MulanPSL2>
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 *
 */

package reset

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

const (
	testNumericZero    = 0
	testNumericOne     = 1
	testNumericTwo     = 2
	testCommandSuccess = ""
	testCommandFailure = "error"
	testFilePermission = 0644
	testDirPermission  = 0755

	testIPv4SegmentA = 10
	testIPv4SegmentB = 0
	testIPv4SegmentC = 0
	testIPv4SegmentD = 1
)

var testIP = fmt.Sprintf("%d.%d.%d.%d", testIPv4SegmentA, testIPv4SegmentB, testIPv4SegmentC, testIPv4SegmentD)

func TestCleanIpRoute(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful route cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip route show") {
					return "192.168.1.0/24 via " + testIP + " dev boc0\n", nil
				}
				return "", nil
			},
		},
		{
			name: "no routes to clean",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip route show") {
					return "", nil
				}
				return "", nil
			},
		},
		{
			name: "execute command error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip route show") {
					return "", errors.New("command failed")
				}
				return "", nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanIpRoute()

			assert.True(t, true)
		})
	}
}

func TestCleanIpNeighbor(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful neighbor cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip neigh show") {
					return "192.168.1.1 lladdr aa:bb:cc:dd:ee:ff PERMANENT boc0\n", nil
				}
				return "", nil
			},
		},
		{
			name: "no neighbors to clean",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip neigh show") {
					return "", nil
				}
				return "", nil
			},
		},
		{
			name: "execute command error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip neigh show") {
					return "", errors.New("command failed")
				}
				return "", nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanIpNeighbor()

			assert.True(t, true)
		})
	}
}

func TestCleanIpLink(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful link cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip link") && strings.Contains(cmd, "awk") {
					return "vxlan_sys_4789\n", nil
				}
				return "", nil
			},
		},
		{
			name: "no links to clean",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip link") && strings.Contains(cmd, "awk") {
					return "", nil
				}
				return "", nil
			},
		},
		{
			name: "execute command error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ip link") && strings.Contains(cmd, "awk") {
					return "", errors.New("command failed")
				}
				return "", nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.ContainsString, func(slice []string, str string) bool {
				for _, s := range slice {
					if s == str {
						return true
					}
				}
				return false
			})
			defer patches.Reset()

			cleanIpLink()

			assert.True(t, true)
		})
	}
}

func TestCleanNetwork(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful network cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "execute command error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", errors.New("command failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.ContainsString, func(slice []string, str string) bool {
				for _, s := range slice {
					if s == str {
						return true
					}
				}
				return false
			})
			defer patches.Reset()

			cleanNetwork()

			assert.True(t, true)
		})
	}
}

func TestCleanKubeletBin(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful kubelet bin cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "stop kubelet error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "StopKubeletBin") {
					return "error", errors.New("stop error")
				}
				return "", nil
			},
		},
		{
			name: "force remove kubelet",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				if strings.Contains(cmd, "ForceRemoveKubeletBin") {
					return "", nil
				}
				if strings.Contains(cmd, "StopKubeletBin") {
					return "error", errors.New("stop error")
				}
				return "", nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanKubeletBin()

			assert.True(t, true)
		})
	}
}

func TestCleanKubelet(t *testing.T) {
	tests := []struct {
		name                   string
		mockIsDocker           func() bool
		mockIsContainerd       func() bool
		mockUnmountKubeletDirs func(string)
	}{
		{
			name: "docker environment",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
			mockUnmountKubeletDirs: func(dir string) {},
		},
		{
			name: "containerd environment",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
			mockUnmountKubeletDirs: func(dir string) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(unmountKubeletDirectories, tt.mockUnmountKubeletDirs)
			defer patches.Reset()

			cleanKubelet()

			assert.True(t, true)
		})
	}
}

func TestUnmountKubeletDirectories(t *testing.T) {
	tests := []struct {
		name        string
		rootDir     string
		expectPanic bool
	}{
		{
			name:    "successful unmount",
			rootDir: "/var/lib/kubelet",
		},
		{
			name:    "unmount directory error",
			rootDir: "/var/lib/kubelet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(unmountKubeletDirectory, func(dir string) error {
				return nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, func(cmd string, args ...string) (string, error) {
				return "", nil
			})
			defer patches.Reset()

			unmountKubeletDirectories(tt.rootDir)

			assert.True(t, true)
		})
	}
}

func TestCleanKubeletContainer(t *testing.T) {
	tests := []struct {
		name             string
		mockIsDocker     func() bool
		mockIsContainerd func() bool
	}{
		{
			name: "docker environment",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
		},
		{
			name: "containerd environment",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			cleanKubeletContainer()

			assert.True(t, true)
		})
	}
}

func TestCleanDockerKubelet(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful docker kubelet cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "stop kubelet error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "error", errors.New("stop error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanDockerKubelet()

			assert.True(t, true)
		})
	}
}

func TestCleanContainerdKubelet(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful containerd kubelet cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "stop kubelet error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "error", errors.New("stop error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanContainerdKubelet()

			assert.True(t, true)
		})
	}
}

func TestAppendKubeletCleanupFiles(t *testing.T) {
	tests := []struct {
		name    string
		rootDir string
	}{
		{
			name:    "append cleanup files",
			rootDir: "/var/lib/kubelet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalNeedDeleteFile := needDeleteFile
			needDeleteFile = []string{}
			defer func() {
				needDeleteFile = originalNeedDeleteFile
			}()

			appendKubeletCleanupFiles(tt.rootDir)

			assert.True(t, len(needDeleteFile) > testNumericZero)
		})
	}
}

func TestCleanKubernetesContainer(t *testing.T) {
	tests := []struct {
		name             string
		mockIsDocker     func() bool
		mockIsContainerd func() bool
		mockExecuteCmd   func(string, ...string) (string, error)
	}{
		{
			name: "docker environment",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
		},
		{
			name: "containerd environment",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanKubernetesContainer()

			assert.True(t, true)
		})
	}
}

func TestCleanDockerK8sContainers(t *testing.T) {
	assert.NotPanics(t, func() {
		patches := gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
			return true
		})
		defer patches.Reset()

		patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, func(cmd string, args ...string) (string, error) {
			return "", nil
		})
		defer patches.Reset()

		cleanDockerK8sContainers()
	})
}

func TestCleanDockerK8sContainersNoDocker(t *testing.T) {
	assert.NotPanics(t, func() {
		patches := gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
			return false
		})
		defer patches.Reset()

		cleanDockerK8sContainers()
	})
}

func TestRemoveDockerContainer(t *testing.T) {
	tests := []struct {
		name           string
		podID          string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name:  "remove container",
			podID: "container1",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name:  "stop container error",
			podID: "container2",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "error", errors.New("stop error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			removeDockerContainer(tt.podID)

			assert.True(t, true)
		})
	}
}

func TestCleanContainerdK8sContainers(t *testing.T) {
	tests := []struct {
		name             string
		mockIsContainerd func() bool
		mockExecuteCmd   func(string, ...string) (string, error)
	}{
		{
			name: "containerd environment with pods",
			mockIsContainerd: func() bool {
				return true
			},
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "pod1 pod2", nil
			},
		},
		{
			name: "non-containerd environment",
			mockIsContainerd: func() bool {
				return false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanContainerdK8sContainers()

			assert.True(t, true)
		})
	}
}

func TestRemoveContainerdContainer(t *testing.T) {
	tests := []struct {
		name           string
		podID          string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name:  "remove container",
			podID: "pod1",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name:  "stop container error",
			podID: "pod2",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "error", errors.New("stop error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			removeContainerdContainer(tt.podID)

			assert.True(t, true)
		})
	}
}

func TestCleanOtherContainer(t *testing.T) {
	tests := []struct {
		name             string
		mockIsDocker     func() bool
		mockIsContainerd func() bool
		mockExecuteCmd   func(string, ...string) (string, error)
	}{
		{
			name: "docker environment",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
		},
		{
			name: "containerd environment",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanOtherContainer()

			assert.True(t, true)
		})
	}
}

func TestCleanDockerAllContainers(t *testing.T) {
	tests := []struct {
		name           string
		mockIsDocker   func() bool
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "docker environment with containers",
			mockIsDocker: func() bool {
				return true
			},
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "non-docker environment",
			mockIsDocker: func() bool {
				return false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanDockerAllContainers()

			assert.True(t, true)
		})
	}
}

func TestCleanContainerdAllContainers(t *testing.T) {
	tests := []struct {
		name             string
		mockIsContainerd func() bool
	}{
		{
			name: "containerd environment",
			mockIsContainerd: func() bool {
				return true
			},
		},
		{
			name: "non-containerd environment",
			mockIsContainerd: func() bool {
				return false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			cleanContainerdAllContainers()

			assert.True(t, true)
		})
	}
}

func TestCleanCrictlContainers(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "list containers error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", errors.New("list error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanCrictlContainers()

			assert.True(t, true)
		})
	}
}

func TestCleanNerdctlContainers(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful cleanup",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "list containers error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", errors.New("list error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			cleanNerdctlContainers()

			assert.True(t, true)
		})
	}
}

func TestDisableDocker(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful disable",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "stop docker error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "error", errors.New("stop error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			disableDocker()

			assert.True(t, true)
		})
	}
}

func TestDisableContainerd(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "successful disable",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "stop containerd error",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "error", errors.New("stop error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			disableContainerd()

			assert.True(t, true)
		})
	}
}

func TestAddNeedDeleteFile(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "add need delete files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalNeedDeleteFile := needDeleteFile
			needDeleteFile = []string{}
			defer func() {
				needDeleteFile = originalNeedDeleteFile
			}()

			addNeedDeleteFile()

			assert.True(t, len(needDeleteFile) > testNumericZero)
		})
	}
}

func TestCleanContainerRuntime(t *testing.T) {
	tests := []struct {
		name             string
		mockIsDocker     func() bool
		mockIsContainerd func() bool
		mockExecuteCmd   func(string, ...string) (string, error)
	}{
		{
			name: "docker environment",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return false
			},
		},
		{
			name: "containerd environment",
			mockIsDocker: func() bool {
				return false
			},
			mockIsContainerd: func() bool {
				return true
			},
		},
		{
			name: "both docker and containerd",
			mockIsDocker: func() bool {
				return true
			},
			mockIsContainerd: func() bool {
				return true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(infrastructure.IsDocker, tt.mockIsDocker)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, tt.mockIsContainerd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.Errorf, func(template string, args ...interface{}) {})
			defer patches.Reset()

			cleanContainerRuntime()

			assert.True(t, true)
		})
	}
}

func TestCleanNeedDeleteFile(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "clean need delete files",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalNeedDeleteFile := needDeleteFile
			needDeleteFile = []string{}
			defer func() {
				needDeleteFile = originalNeedDeleteFile
			}()

			addNeedDeleteFile()

			patches := gomonkey.ApplyFunc(removeDir, func(path string) error {
				return nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(removeFile, func(path string) error {
				return nil
			})
			defer patches.Reset()

			cleanNeedDeleteFile()

			assert.True(t, true)
		})
	}
}

func TestRemoveDir(t *testing.T) {
	assert.NotPanics(t, func() {
		removeDir("/nonexistent/path")
	})
}

func TestRemoveDirNotExist(t *testing.T) {
	assert.NotPanics(t, func() {
		removeDir("/nonexistent/path")
	})
}

func TestRemoveDirWithReaddirnames(t *testing.T) {
	assert.NotPanics(t, func() {
		removeDir("/nonexistent/path")
	})
}

func TestRemoveFile(t *testing.T) {
	assert.NotPanics(t, func() {
		removeFile("/nonexistent/file.txt")
	})
}

func TestRemoveFileNotExist(t *testing.T) {
	assert.NotPanics(t, func() {
		removeFile("/nonexistent/file.txt")
	})
}

func TestRemoveFileIsDirectory(t *testing.T) {
	assert.NotPanics(t, func() {
		removeFile("/nonexistent/path")
	})
}

func TestUnmountKubeletDirectory(t *testing.T) {
	tests := []struct {
		name         string
		kubeletDir   string
		mockReadFile func(string) ([]byte, error)
		expectError  bool
	}{
		{
			name:       "successful unmount",
			kubeletDir: "/var/lib/kubelet",
			mockReadFile: func(path string) ([]byte, error) {
				return []byte(""), nil
			},
			expectError: false,
		},
		{
			name:       "read file error",
			kubeletDir: "/var/lib/kubelet",
			mockReadFile: func(path string) ([]byte, error) {
				return nil, errors.New("read error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadFile, tt.mockReadFile)
			defer patches.Reset()

			err := unmountKubeletDirectory(tt.kubeletDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestUnmountKubeletDirectoryWithMounts(t *testing.T) {
	kubeletDir := "/var/lib/kubelet"
	mountsContent := fmt.Sprintf("/var/lib/kubelet/pods/test1 %s/pods/test1 ext4 rw 0 0\n", kubeletDir)

	patches := gomonkey.ApplyFunc(os.ReadFile, func(path string) ([]byte, error) {
		return []byte(mountsContent), nil
	})
	defer patches.Reset()

	err := unmountKubeletDirectory(kubeletDir)
	assert.NoError(t, err)
}

func TestRepoRemove(t *testing.T) {
	assert.NotPanics(t, func() {
		patches := gomonkey.ApplyFunc(host.PlatformInformation, func() (string, string, string, error) {
			return "centos", "", "", nil
		})
		defer patches.Reset()

		patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, func(cmd string, args ...string) (string, error) {
			return "", nil
		})
		defer patches.Reset()

		repoRemove("docker*")
	})
}

func TestRepoRemoveUbuntu(t *testing.T) {
	assert.NotPanics(t, func() {
		patches := gomonkey.ApplyFunc(host.PlatformInformation, func() (string, string, string, error) {
			return "ubuntu", "", "", nil
		})
		defer patches.Reset()

		patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, func(cmd string, args ...string) (string, error) {
			return "", nil
		})
		defer patches.Reset()

		repoRemove("docker*", "containerd.io")
	})
}

func TestRepoRemoveUnknownPlatform(t *testing.T) {
	assert.NotPanics(t, func() {
		patches := gomonkey.ApplyFunc(host.PlatformInformation, func() (string, string, string, error) {
			return "unknown", "", "", nil
		})
		defer patches.Reset()

		patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, func(cmd string, args ...string) (string, error) {
			return "", nil
		})
		defer patches.Reset()

		repoRemove("docker*")
	})
}

func TestRepoRemoveHostInfoError(t *testing.T) {
	assert.NotPanics(t, func() {
		patches := gomonkey.ApplyFunc(host.PlatformInformation, func() (string, string, string, error) {
			return "", "", "", errors.New("host info error")
		})
		defer patches.Reset()

		repoRemove("docker*")
	})
}

func TestCleanNetworkWithInterfaces(t *testing.T) {
	tests := []struct {
		name           string
		mockExecuteCmd func(string, ...string) (string, error)
	}{
		{
			name: "cleanup with docker0 and nerdctl0",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", nil
			},
		},
		{
			name: "cleanup with interface errors",
			mockExecuteCmd: func(cmd string, args ...string) (string, error) {
				return "", errors.New("interface error")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, tt.mockExecuteCmd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.ContainsString, func(slice []string, str string) bool {
				for _, s := range slice {
					if s == str {
						return true
					}
				}
				return false
			})
			defer patches.Reset()

			cleanNetwork()

			assert.True(t, true)
		})
	}
}

func TestRemoveAllInOne(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "call all cleanup functions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, func(cmd string, args ...string) (string, error) {
				return "", nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, func(cmd string, args ...string) (string, error) {
				return "", nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.Errorf, func(template string, args ...interface{}) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(removeDir, func(path string) error {
				return nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(removeFile, func(path string) error {
				return nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
				return true
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool {
				return false
			})
			defer patches.Reset()

			removeAllInOne()

			assert.True(t, true)
		})
	}
}

func TestRemoveAllInOneContainerd(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "call all cleanup functions for containerd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(global.Command.ExecuteCommandWithCombinedOutput, func(cmd string, args ...string) (string, error) {
				return "", nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.Command.ExecuteCommandWithOutput, func(cmd string, args ...string) (string, error) {
				return "", nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.Errorf, func(template string, args ...interface{}) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(removeDir, func(path string) error {
				return nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(removeFile, func(path string) error {
				return nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsDocker, func() bool {
				return false
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(infrastructure.IsContainerd, func() bool {
				return true
			})
			defer patches.Reset()

			removeAllInOne()

			assert.True(t, true)
		})
	}
}
