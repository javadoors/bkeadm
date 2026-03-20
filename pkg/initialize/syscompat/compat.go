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

package syscompat

import (
	"errors"
	"fmt"
	"strings"

	"github.com/shirou/gopsutil/v3/host"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

// Compat Compatible with multiple system Settings
func Compat() error {
	h, _ := host.Info()
	stopFirewall()

	output, err := verifyAndInstallIptables(h.Platform)
	if err != nil {
		return err
	}

	if strings.Contains(output, "nf_tables") {
		return switchToLegacyIptables(h.Platform)
	}
	return nil
}

// stopFirewall 停止防火墙
func stopFirewall() {
	_ = global.Command.ExecuteCommand("sudo", "systemctl", "stop", "firewalld")
	_ = global.Command.ExecuteCommand("sudo", "systemctl", "disable", "firewalld")
	_ = global.Command.ExecuteCommand("sudo", "ufw", "disable")
}

// verifyAndInstallIptables 验证并安装 iptables
func verifyAndInstallIptables(platform string) (string, error) {
	output, err := global.Command.ExecuteCommandWithOutput("iptables", "-V")
	if err == nil {
		log.BKEFormat(log.INFO, fmt.Sprintf("iptables -V output: %s", output))
		return output, nil
	}

	log.BKEFormat(log.WARN, fmt.Sprintf("iptables -V error: %s", err.Error()))

	if !strings.Contains(err.Error(), "not found") {
		return "", err
	}

	return installIptables(platform)
}

// installIptables 根据平台安装 iptables
func installIptables(platform string) (string, error) {
	log.BKEFormat(log.INFO, "install iptables...")

	switch strings.ToLower(platform) {
	case "centos", "kylin", "openeuler":
		return installIptablesYum()
	case "ubuntu", "debian", "fedora":
		return installIptablesApt()
	default:
		log.BKEFormat(log.ERROR, "Unsupported platform, only support centos, ubuntu, debian, fedora")
		return "", nil
	}
}

// verifyIptablesInstallation 验证 iptables 安装
func verifyIptablesInstallation() (string, error) {
	output, err := global.Command.ExecuteCommandWithOutput("iptables", "-V")
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to exec iptables -V, %s", err.Error()))
		return "", nil
	}
	log.BKEFormat(log.INFO, fmt.Sprintf("iptables -V output: %s", output))
	return output, nil
}

// installIptablesYum 使用 yum 安装 iptables
func installIptablesYum() (string, error) {
	if err := global.Command.ExecuteCommand("yum", "-y", "install", "iptables"); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install iptables, %s", err.Error()))
		return "", nil
	}
	return verifyIptablesInstallation()
}

// installIptablesApt 使用 apt 安装 iptables
func installIptablesApt() (string, error) {
	_ = global.Command.ExecuteCommand("apt", "-y", "clean")
	_ = global.Command.ExecuteCommand("apt", "-y", "update")
	if err := global.Command.ExecuteCommand("apt", "-y", "install", "iptables"); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install iptables, %s", err.Error()))
		return "", nil
	}
	return verifyIptablesInstallation()
}

// switchToLegacyIptables 切换到传统 iptables
func switchToLegacyIptables(platform string) error {
	switch strings.ToLower(platform) {
	case "centos", "kylin", "openeuler":
		return reinstallIptablesYum()
	case "ubuntu", "debian":
		return updateAlternativesDebian()
	case "fedora":
		return updateAlternativesFedora()
	default:
		return errors.New(fmt.Sprintf("%s is not supported", platform))
	}
}

// reinstallIptablesYum 使用 yum 重新安装 iptables
func reinstallIptablesYum() error {
	if err := global.Command.ExecuteCommand("yum", "-y", "remove", "iptables"); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to remove iptables, %s", err.Error()))
		return err
	}
	if err := global.Command.ExecuteCommand("yum", "-y", "install", "iptables"); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to install iptables, %s", err.Error()))
		return err
	}
	return nil
}

// updateAlternativesDebian 更新 Debian/Ubuntu 的 iptables 替代方案
func updateAlternativesDebian() error {
	alternatives := [][]string{
		{"iptables", "/usr/sbin/iptables-legacy"},
		{"ip6tables", "/usr/sbin/ip6tables-legacy"},
		{"arptables", "/usr/sbin/arptables-legacy"},
		{"ebtables", "/usr/sbin/ebtables-legacy"},
	}
	for _, alt := range alternatives {
		if err := global.Command.ExecuteCommand("update-alternatives", "--set", alt[0], alt[1]); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to update %s, %s", alt[0], err.Error()))
			return nil
		}
	}
	return nil
}

// updateAlternativesFedora 更新 Fedora 的 iptables 替代方案
func updateAlternativesFedora() error {
	if err := global.Command.ExecuteCommand("update-alternatives",
		"--set", "iptables", "/usr/sbin/iptables-legacy"); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to update iptables, %s", err.Error()))
		return nil
	}
	return nil
}

func RepoUpdate() error {
	packageManager := ""
	h, _, _, err := host.PlatformInformation()
	if err != nil {
		log.Errorf("get host platform information failed, err: %v", err)
	}
	switch h {
	case "ubuntu", "debian":
		packageManager = "apt"
	case "centos", "kylin", "redhat", "fedora", "openeuler", "hopeos":
		packageManager = "yum"
	default:
		packageManager = "unknown"
	}
	switch packageManager {
	case "apt":
		output, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", fmt.Sprintf("%s -y clean", packageManager))
		if err != nil {
			log.Errorf("update packages failed, err: %v, out: %s", err, output)
			return nil
		}

		output, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", fmt.Sprintf("%s -y update", packageManager))
		if err != nil {
			log.Errorf("update packages failed, err: %v, out: %s", err, output)
			return nil
		}
	case "yum":
		// yum clean all
		output, err := global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", fmt.Sprintf("%s clean all", packageManager))
		if err != nil {
			log.Errorf("update packages failed, err: %v, out: %s", err, output)
			return nil
		}
		// yum makecache
		output, err = global.Command.ExecuteCommandWithCombinedOutput("/bin/sh", "-c", fmt.Sprintf("%s makecache", packageManager))
		if err != nil {
			return nil
		}
	default:
		log.Errorf("package manager %q not supported", packageManager)
		return nil
	}
	return nil
}
