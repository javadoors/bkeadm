/*
 * Copyright (c) 2025 Bocloud Technologies Co., Ltd.
 * installer is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          <http://license.coscl.org.cn/MulanPSL2>
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */

package timezone

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/server"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/cluster-api-provider-bke/common/ntp"
)

const (
	testIPv4SegmentA = 127
	testIPv4SegmentB = 0
	testIPv4SegmentC = 0
	testIPv4SegmentD = 1
)

const (
	testFileModeReadOnly = 0644
	testFileModeExec     = 0755
)

var testLoopbackIP = net.IPv4(
	testIPv4SegmentA,
	testIPv4SegmentB,
	testIPv4SegmentC,
	testIPv4SegmentD,
)

func TestFindTimezoneFileNotFound(t *testing.T) {
	result, err := findTimezoneFile("Invalid/Timezone")
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestCopyFileSuccess(t *testing.T) {
	src := t.TempDir() + "/src"
	dst := t.TempDir() + "/dst"
	content := []byte("test content")

	err := os.WriteFile(src, content, testFileModeReadOnly)
	assert.NoError(t, err)

	err = copyFile(src, dst)
	assert.NoError(t, err)

	result, err := os.ReadFile(dst)
	assert.NoError(t, err)
	assert.Equal(t, content, result)
}

func TestCopyFileReadError(t *testing.T) {
	dst := t.TempDir() + "/dst"
	err := copyFile("/nonexistent/file", dst)
	assert.Error(t, err)
}

func TestCopyFileWriteError(t *testing.T) {
	src := t.TempDir() + "/src"
	dst := "/invalid/dst"
	content := []byte("test content")

	err := os.WriteFile(src, content, testFileModeReadOnly)
	assert.NoError(t, err)

	err = copyFile(src, dst)
	assert.Error(t, err)
}

func TestNTPServerLocalName(t *testing.T) {
	patches := gomonkey.ApplyFunc(startLocalNTPServer, func(string) (string, error) {
		return testLoopbackIP.String() + ":123", nil
	})
	defer patches.Reset()

	result, err := NTPServer("local", testLoopbackIP.String(), false)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestNTPServerExternalSuccess(t *testing.T) {
	patches := gomonkey.ApplyFunc(ntp.Date, func(string) error {
		return nil
	})
	defer patches.Reset()

	result, err := NTPServer("pool.ntp.org", testLoopbackIP.String(), false)
	assert.NoError(t, err)
	assert.Equal(t, "pool.ntp.org", result)
}

func TestNTPServerExternalFailOffline(t *testing.T) {
	patches := gomonkey.ApplyFunc(ntp.Date, func(string) error {
		return assert.AnError
	})
	defer patches.Reset()

	result, err := NTPServer("pool.ntp.org", testLoopbackIP.String(), false)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestNTPServerDefaultServer(t *testing.T) {
	patches := gomonkey.ApplyFunc(ntp.Date, func(string) error {
		return assert.AnError
	})
	defer patches.Reset()

	patches.ApplyFunc(startLocalNTPServer, func(string) (string, error) {
		return testLoopbackIP.String() + ":123", nil
	})
	defer patches.Reset()

	result, err := NTPServer("cn.pool.ntp.org:123", testLoopbackIP.String(), false)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestNTPServerUnreachableWithOnline(t *testing.T) {
	patches := gomonkey.ApplyFunc(ntp.Date, func(string) error {
		return assert.AnError
	})
	defer patches.Reset()

	result, err := NTPServer("unreachable.server", testLoopbackIP.String(), true)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestTryConnectExternalNTPSuccess(t *testing.T) {
	patches := gomonkey.ApplyFunc(ntp.Date, func(string) error {
		return nil
	})
	defer patches.Reset()

	result, err := tryConnectExternalNTP("pool.ntp.org", false)
	assert.NoError(t, err)
	assert.Equal(t, "pool.ntp.org", result)
}

func TestTryConnectExternalNTPFailOffline(t *testing.T) {
	patches := gomonkey.ApplyFunc(ntp.Date, func(string) error {
		return assert.AnError
	})
	defer patches.Reset()

	result, err := tryConnectExternalNTP("pool.ntp.org", false)
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestTryConnectExternalNTPRetrySuccess(t *testing.T) {
	callCount := 0
	patches := gomonkey.ApplyFunc(ntp.Date, func(string) error {
		callCount++
		if callCount < 2 {
			return assert.AnError
		}
		return nil
	})
	defer patches.Reset()

	result, err := tryConnectExternalNTP("pool.ntp.org", true)
	assert.NoError(t, err)
	assert.Equal(t, "pool.ntp.org", result)
}

func TestRetryNTPConnectionAllFail(t *testing.T) {
	patches := gomonkey.ApplyFunc(ntp.Date, func(string) error {
		return assert.AnError
	})
	defer patches.Reset()

	result, err := retryNTPConnection("pool.ntp.org")
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestStartLocalNTPServerSuccess(t *testing.T) {
	originalK8s := global.K8s
	global.K8s = nil
	defer func() { global.K8s = originalK8s }()

	patches := gomonkey.ApplyFunc(utils.ExecPath, func() (string, error) {
		return "/fake/path", nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Getenv, func(string) string {
		return ""
	})

	patches.ApplyFunc(os.Getpid, func() int {
		return 12345
	})

	patches.ApplyFunc((*exec.Cmd).Start, func(*exec.Cmd) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(server.TryConnectNTPServer, func(string) bool {
		return true
	})
	defer patches.Reset()

	result, err := startLocalNTPServer(testLoopbackIP.String())
	assert.NoError(t, err)
	assert.Contains(t, result, testLoopbackIP.String())
}

func TestStartLocalNTPServerExecPathError(t *testing.T) {
	patches := gomonkey.ApplyFunc(utils.ExecPath, func() (string, error) {
		return "", assert.AnError
	})
	defer patches.Reset()

	result, err := startLocalNTPServer(testLoopbackIP.String())
	assert.Error(t, err)
	assert.Empty(t, result)
}

func TestSetTimeZoneAlreadySet(t *testing.T) {
	patches := gomonkey.ApplyFunc(utils.Exists, func(string) bool {
		return true
	})
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte("Asia/Shanghai"), nil
	})
	defer patches.Reset()

	err := SetTimeZone()
	assert.NoError(t, err)
}

func TestSetTimeZoneReadTimezoneError(t *testing.T) {
	patches := gomonkey.ApplyFunc(utils.Exists, func(string) bool {
		return true
	})
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return nil, assert.AnError
	})
	defer patches.Reset()

	err := SetTimeZone()
	assert.Error(t, err)
}

func TestSetTimeZoneFindTimezoneFileError(t *testing.T) {
	patches := gomonkey.ApplyFunc(utils.Exists, func(string) bool {
		return false
	})
	defer patches.Reset()

	patches.ApplyFunc(findTimezoneFile, func(string) (string, error) {
		return "", assert.AnError
	})
	defer patches.Reset()

	err := SetTimeZone()
	assert.Error(t, err)
}

func TestSetTimeZoneSymlinkSuccess(t *testing.T) {
	tempDir := t.TempDir()
	zoneInfoPath := filepath.Join(tempDir, "Asia/Shanghai")

	err := os.MkdirAll(filepath.Dir(zoneInfoPath), testFileModeExec)
	assert.NoError(t, err)

	err = os.WriteFile(zoneInfoPath, []byte("tzdata"), testFileModeReadOnly)
	assert.NoError(t, err)

	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return path == timezonePath
	})
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte("America/New_York"), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(findTimezoneFile, func(zone string) (string, error) {
		return zoneInfoPath, nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Stat, func(string) (os.FileInfo, error) {
		return nil, assert.AnError
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Rename, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Symlink, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	err = SetTimeZone()
	assert.NoError(t, err)
}

func TestSetTimeZoneSymlinkErrorCopySuccess(t *testing.T) {
	copyFileCalled := false

	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return path == timezonePath
	})
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte("America/New_York"), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(findTimezoneFile, func(zone string) (string, error) {
		return "/usr/share/zoneinfo/Asia/Shanghai", nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Stat, func(string) (os.FileInfo, error) {
		return nil, nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Rename, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Symlink, func(target, link string) error {
		return assert.AnError
	})
	defer patches.Reset()

	patches.ApplyFunc(copyFile, func(src, dst string) error {
		copyFileCalled = true
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.WriteFile, func(path string, data []byte, perm os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	err := SetTimeZone()
	assert.NoError(t, err)
	assert.True(t, copyFileCalled, "copyFile should have been called")
}

func TestSetTimeZoneCopyFileError(t *testing.T) {
	tempDir := t.TempDir()
	zoneInfoPath := filepath.Join(tempDir, "Asia/Shanghai")

	err := os.MkdirAll(filepath.Dir(zoneInfoPath), testFileModeExec)
	assert.NoError(t, err)

	err = os.WriteFile(zoneInfoPath, []byte("tzdata"), testFileModeReadOnly)
	assert.NoError(t, err)

	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return path == timezonePath || path == localtimePath
	})
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte("America/New_York"), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(findTimezoneFile, func(zone string) (string, error) {
		return zoneInfoPath, nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Stat, func(string) (os.FileInfo, error) {
		return nil, nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Symlink, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Rename, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(copyFile, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	err = SetTimeZone()
	assert.Error(t, err)
}

func TestSetTimeZoneWriteFileError(t *testing.T) {
	tempDir := t.TempDir()
	zoneInfoPath := filepath.Join(tempDir, "Asia/Shanghai")

	err := os.MkdirAll(filepath.Dir(zoneInfoPath), testFileModeExec)
	assert.NoError(t, err)

	err = os.WriteFile(zoneInfoPath, []byte("tzdata"), testFileModeReadOnly)
	assert.NoError(t, err)

	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return false
	})
	defer patches.Reset()

	patches.ApplyFunc(findTimezoneFile, func(zone string) (string, error) {
		return zoneInfoPath, nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Stat, func(string) (os.FileInfo, error) {
		return nil, assert.AnError
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Rename, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Symlink, func(string, string) error {
		return assert.AnError
	})
	defer patches.Reset()

	patches.ApplyFunc(copyFile, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	writeFileCalled := false
	patches.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		writeFileCalled = true
		return assert.AnError
	})
	defer patches.Reset()

	err = SetTimeZone()
	assert.Error(t, err)
	assert.True(t, writeFileCalled)
}

func TestSetTimeZoneRenameError(t *testing.T) {
	tempDir := t.TempDir()
	zoneInfoPath := filepath.Join(tempDir, "Asia/Shanghai")

	err := os.MkdirAll(filepath.Dir(zoneInfoPath), testFileModeExec)
	assert.NoError(t, err)

	err = os.WriteFile(zoneInfoPath, []byte("tzdata"), testFileModeReadOnly)
	assert.NoError(t, err)

	patches := gomonkey.ApplyFunc(utils.Exists, func(path string) bool {
		return path == timezonePath
	})
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile, func(string) ([]byte, error) {
		return []byte("America/New_York"), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(findTimezoneFile, func(zone string) (string, error) {
		return zoneInfoPath, nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Stat, func(string) (os.FileInfo, error) {
		return nil, nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Rename, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.Symlink, func(string, string) error {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.WriteFile, func(string, []byte, os.FileMode) error {
		return nil
	})
	defer patches.Reset()

	err = SetTimeZone()
	assert.NoError(t, err)
}
