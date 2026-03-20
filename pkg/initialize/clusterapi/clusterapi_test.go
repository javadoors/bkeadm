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

package clusterapi

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/k8s"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
)

const (
	testZeroValue = 0
	testOneValue  = 1
	testTwoValue  = 2
	testThreeVal  = 3
)

func TestEnsureK8sClientWithExistingClient(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	err := ensureK8sClient()
	assert.NoError(t, err)
}

func TestEnsureK8sClientNewClientSuccess(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	global.K8s = nil

	patches := gomonkey.ApplyFunc(k8s.NewKubernetesClient, func(string) (k8s.KubernetesClient, error) {
		return &k8s.MockK8sClient{}, nil
	})
	defer patches.Reset()

	err := ensureK8sClient()
	assert.NoError(t, err)
}

func TestEnsureK8sClientNewClientError(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	global.K8s = nil

	patches := gomonkey.ApplyFunc(k8s.NewKubernetesClient, func(string) (k8s.KubernetesClient, error) {
		return nil, assert.AnError
	})
	defer patches.Reset()

	err := ensureK8sClient()
	assert.Error(t, err)
}

func TestWriteClusterAPITemplatesSuccess(t *testing.T) {
	tempDir := t.TempDir()
	tmplDir := filepath.Join(tempDir, "tmpl")

	err := writeClusterAPITemplates(tmplDir)
	assert.NoError(t, err)
	assert.DirExists(t, tmplDir)
}

func TestWriteClusterAPITemplatesMkdirError(t *testing.T) {
	patches := gomonkey.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error {
		return assert.AnError
	})
	defer patches.Reset()

	err := writeClusterAPITemplates("/invalid/path/tmpl")
	assert.Error(t, err)
}

func TestWriteClusterAPITemplatesWriteFileError(t *testing.T) {
	tempDir := t.TempDir()

	patches := gomonkey.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		return assert.AnError
	})
	defer patches.Reset()

	err := writeClusterAPITemplates(tempDir)
	assert.Error(t, err)
}

func TestWriteClusterAPITemplatesMultipleFilesError(t *testing.T) {
	tempDir := t.TempDir()
	writeCount := 0

	patches := gomonkey.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		writeCount++
		if writeCount == testTwoValue {
			return assert.AnError
		}
		return nil
	})
	defer patches.Reset()

	err := writeClusterAPITemplates(tempDir)
	assert.Error(t, err)
}

func TestAreAllPodsRunningAllRunning(t *testing.T) {
	pods := []corev1.Pod{
		{Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		{Status: corev1.PodStatus{Phase: corev1.PodRunning}},
	}

	result := areAllPodsRunning(pods)
	assert.True(t, result)
}

func TestAreAllPodsRunningNotAllRunning(t *testing.T) {
	pods := []corev1.Pod{
		{Status: corev1.PodStatus{Phase: corev1.PodRunning}},
		{Status: corev1.PodStatus{Phase: corev1.PodPending}},
	}

	result := areAllPodsRunning(pods)
	assert.False(t, result)
}

func TestAreAllPodsRunningWithContainerWaiting(t *testing.T) {
	pods := []corev1.Pod{
		{
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "ImagePullBackOff",
							},
						},
					},
				},
			},
		},
	}

	result := areAllPodsRunning(pods)
	assert.False(t, result)
}

func TestAreAllPodsRunningEmpty(t *testing.T) {
	pods := []corev1.Pod{}
	result := areAllPodsRunning(pods)
	assert.True(t, result)
}

func TestAreAllPodsRunningWithSucceededPhase(t *testing.T) {
	pods := []corev1.Pod{
		{Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
	}

	result := areAllPodsRunning(pods)
	assert.False(t, result)
}

func TestAreAllPodsRunningWithFailedPhase(t *testing.T) {
	pods := []corev1.Pod{
		{Status: corev1.PodStatus{Phase: corev1.PodFailed}},
	}

	result := areAllPodsRunning(pods)
	assert.False(t, result)
}

func TestAreAllPodsRunningWithUnknownPhase(t *testing.T) {
	pods := []corev1.Pod{
		{Status: corev1.PodStatus{Phase: corev1.PodUnknown}},
	}

	result := areAllPodsRunning(pods)
	assert.False(t, result)
}

func TestAreAllPodsRunningWithMultipleContainers(t *testing.T) {
	pods := []corev1.Pod{
		{
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{
					{
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
					{
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{
								Reason: "CrashLoopBackOff",
							},
						},
					},
				},
			},
		},
		{
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
			},
		},
	}

	result := areAllPodsRunning(pods)
	assert.False(t, result)
}

func TestDeployClusterAPISuccess(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	patches := gomonkey.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		return nil
	})

	patches.ApplyMethod(mockK8sClient, "InstallYaml", func(*k8s.MockK8sClient, string, map[string]string, string) error {
		return nil
	})

	patches.ApplyFunc(waitForClusterAPIPodsRunning, func() error {
		return nil
	})

	err := DeployClusterAPI("http://repo.example.com", "v1.0.0", "v1.0.0")
	assert.NoError(t, err)
}

func TestDeployClusterAPIEnsureClientError(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	global.K8s = nil

	patches := gomonkey.ApplyFunc(k8s.NewKubernetesClient, func(string) (k8s.KubernetesClient, error) {
		return nil, assert.AnError
	})
	defer patches.Reset()

	err := DeployClusterAPI("http://repo.example.com", "v1.0.0", "v1.0.0")
	assert.Error(t, err)
}

func TestDeployClusterAPIWriteTemplatesError(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	patches := gomonkey.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error {
		return assert.AnError
	})
	defer patches.Reset()

	err := DeployClusterAPI("http://repo.example.com", "v1.0.0", "v1.0.0")
	assert.Error(t, err)
}

func TestDeployClusterAPIInstallCertManagerError(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	patches := gomonkey.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		return nil
	})

	installCount := 0
	patches.ApplyMethod(mockK8sClient, "InstallYaml", func(*k8s.MockK8sClient, string, map[string]string, string) error {
		installCount++
		if installCount == testOneValue {
			return assert.AnError
		}
		return nil
	})

	patches.ApplyFunc(waitForClusterAPIPodsRunning, func() error {
		return nil
	})

	err := DeployClusterAPI("http://repo.example.com", "v1.0.0", "v1.0.0")
	assert.Error(t, err)
}

func TestDeployClusterAPIInstallClusterAPIError(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	patches := gomonkey.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		return nil
	})

	patches.ApplyMethod(mockK8sClient, "InstallYaml", func(*k8s.MockK8sClient, string, map[string]string, string) error {
		return assert.AnError
	})

	patches.ApplyFunc(waitForClusterAPIPodsRunning, func() error {
		return nil
	})

	patches.ApplyFunc(time.Sleep, func(time.Duration) {})

	patches.ApplyFunc(installClusterAPIWithRetry, func(string, string) error {
		return assert.AnError
	})

	err := DeployClusterAPI("http://repo.example.com", "v1.0.0", "v1.0.0")
	assert.Error(t, err)
}

func TestDeployClusterAPIInstallClusterAPIBKEError(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	patches := gomonkey.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		return nil
	})

	installCount := 0
	patches.ApplyMethod(mockK8sClient, "InstallYaml", func(*k8s.MockK8sClient, string, map[string]string, string) error {
		installCount++
		if installCount == testThreeVal {
			return assert.AnError
		}
		return nil
	})

	patches.ApplyFunc(waitForClusterAPIPodsRunning, func() error {
		return nil
	})

	patches.ApplyFunc(time.Sleep, func(time.Duration) {})

	err := DeployClusterAPI("http://repo.example.com", "v1.0.0", "v1.0.0")
	assert.Error(t, err)
}

func TestDeployClusterAPIWaitPodsError(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	patches := gomonkey.ApplyFunc(os.MkdirAll, func(string, os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		return nil
	})

	patches.ApplyMethod(mockK8sClient, "InstallYaml", func(*k8s.MockK8sClient, string, map[string]string, string) error {
		return nil
	})

	patches.ApplyFunc(waitForClusterAPIPodsRunning, func() error {
		return assert.AnError
	})

	err := DeployClusterAPI("http://repo.example.com", "v1.0.0", "v1.0.0")
	assert.Error(t, err)
}

func TestInstallClusterAPIWithRetrySuccess(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	patches := gomonkey.ApplyMethodFunc(mockK8sClient, "InstallYaml", func(string, map[string]string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(time.Sleep, func(time.Duration) {})

	err := installClusterAPIWithRetry("/test/yaml/file.yaml", "http://repo.example.com")
	assert.NoError(t, err)
}

func TestInstallClusterAPIWithRetryMultipleRetriesSuccess(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	retryCount := 0
	patches := gomonkey.ApplyMethodFunc(mockK8sClient, "InstallYaml", func(string, map[string]string, string) error {
		retryCount++
		if retryCount < testThreeVal {
			return assert.AnError
		}
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(time.Sleep, func(time.Duration) {})

	err := installClusterAPIWithRetry("/test/yaml/file.yaml", "http://repo.example.com")
	assert.NoError(t, err)
	assert.Equal(t, testThreeVal, retryCount)
}

func TestInstallClusterAPIWithRetryInstallError(t *testing.T) {
	originalK8s := global.K8s
	defer func() { global.K8s = originalK8s }()

	mockK8sClient := &k8s.MockK8sClient{}
	global.K8s = mockK8sClient

	callCount := 0
	maxCalls := 2

	patches := gomonkey.ApplyMethodFunc(mockK8sClient, "InstallYaml", func(string, map[string]string, string) error {
		callCount++
		if callCount < maxCalls {
			return assert.AnError
		}
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(time.Sleep, func(time.Duration) {})

	err := installClusterAPIWithRetry("/test/yaml/file.yaml", "http://repo.example.com")
	assert.NoError(t, err)
	assert.Equal(t, maxCalls, callCount)
}

func TestWaitForClusterAPIPodsRunningSuccess(t *testing.T) {
	patches := gomonkey.ApplyFunc(waitForClusterAPIPodsRunning, func() error {
		return nil
	})
	defer patches.Reset()

	err := waitForClusterAPIPodsRunning()
	assert.NoError(t, err)
}

func TestWaitForClusterAPIPodsRunningError(t *testing.T) {
	patches := gomonkey.ApplyFunc(waitForClusterAPIPodsRunning, func() error {
		return assert.AnError
	})
	defer patches.Reset()

	err := waitForClusterAPIPodsRunning()
	assert.Error(t, err)
}
