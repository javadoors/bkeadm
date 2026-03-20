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

package timezone

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/zh-five/xdaemon"
	configinit "gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/initialize"
	"gopkg.openfuyao.cn/cluster-api-provider-bke/common/ntp"

	"gopkg.openfuyao.cn/bkeadm/pkg/server"
	"gopkg.openfuyao.cn/bkeadm/utils"
)

var (
	localtimePath = "/etc/localtime"
	timezone      = "Asia/Shanghai"
	timezonePath  = "/etc/timezone"
	zoneDirs      = []string{
		"/usr/share/zoneinfo",
		"/usr/lib/zoneinfo",
		"/usr/local/share/zoneinfo",
		"/system/usr/share/zoneinfo",
	}
)

func findTimezoneFile(zone string) (string, error) {
	for _, dir := range zoneDirs {
		path := filepath.Join(dir, zone)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return "", fmt.Errorf("timezone file not found for %s", zone)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, utils.DefaultFilePermission)
}

func SetTimeZone() error {
	if utils.Exists(timezonePath) {
		tz, err := os.ReadFile(timezonePath)
		if err != nil {
			return err
		}
		if string(tz) == timezone {
			return nil
		}
	}

	zoneInfoPath, err := findTimezoneFile(timezone)
	if err != nil {
		return err
	}

	if _, err = os.Stat(localtimePath); err == nil {
		if err = os.Rename(localtimePath, localtimePath+".bak"); err != nil {
			fmt.Printf("rename file err: %v", err)
		}
	}

	if err = os.Symlink(zoneInfoPath, localtimePath); err == nil {
		return os.WriteFile(timezonePath, []byte(timezone), utils.DefaultFilePermission)
	}

	if err = copyFile(zoneInfoPath, localtimePath); err != nil {
		return fmt.Errorf("sym link and copy file both fail: %v", err)
	}

	return os.WriteFile(timezonePath, []byte(timezone), utils.DefaultFilePermission)
}

func NTPServer(ntpServer, hostIP string, online bool) (string, error) {
	if ntpServer != utils.LocalNTPName {
		if server, err := tryConnectExternalNTP(ntpServer, online); err == nil {
			return server, nil
		} else if online {
			return "", err
		}
	}

	if ntpServer != utils.LocalNTPName && ntpServer != configinit.DefaultNTPServer {
		return "", errors.New(fmt.Sprintf("ntp server %s is unreachable", ntpServer))
	}

	return startLocalNTPServer(hostIP)
}

// tryConnectExternalNTP 尝试连接外部 NTP 服务器
func tryConnectExternalNTP(ntpServer string, online bool) (string, error) {
	if err := ntp.Date(ntpServer); err == nil {
		return ntpServer, nil
	}

	if !online {
		return "", errors.New(fmt.Sprintf("ntp server %s is unreachable", ntpServer))
	}

	return retryNTPConnection(ntpServer)
}

// retryNTPConnection 重试 NTP 连接
func retryNTPConnection(ntpServer string) (string, error) {
	const maxRetries = 2
	for i := 0; i < maxRetries; i++ {
		time.Sleep(utils.DefaultSleepSeconds * time.Second)
		if err := ntp.Date(ntpServer); err == nil {
			return ntpServer, nil
		}
	}
	return "", errors.New(fmt.Sprintf("Failed to connect to the ntp server %s", ntpServer))
}

// startLocalNTPServer 启动本地 NTP 服务器
func startLocalNTPServer(hostIP string) (string, error) {
	bin, err := utils.ExecPath()
	if err != nil {
		return "", errors.New(fmt.Sprintf("Failed to obtain the execution path %s", err.Error()))
	}

	env := os.Environ()
	env = append(env, "bke="+bin)
	cmd := &exec.Cmd{
		Path:        bin,
		Args:        []string{bin, "start", "ntpserver", "--systemd"},
		Env:         env,
		SysProcAttr: xdaemon.NewSysProcAttr(),
	}
	if err = cmd.Start(); err != nil {
		return "", err
	}

	localAddr := fmt.Sprintf("127.0.0.1:%d", utils.DefaultNTPServerPort)
	if !server.TryConnectNTPServer(localAddr) {
		return "", errors.New(fmt.Sprintf("Failed to connect to the ntp server %s", localAddr))
	}
	return fmt.Sprintf("%s:%d", hostIP, utils.DefaultNTPServerPort), nil
}
