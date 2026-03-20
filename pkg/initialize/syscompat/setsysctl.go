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
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func SetSysctl() {
	file, err := os.OpenFile("/etc/sysctl.conf", os.O_WRONLY|os.O_CREATE|os.O_APPEND, utils.DefaultReadWritePermission)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Unable to open file %s %s", "/etc/sysctl.conf", err.Error()))
		return
	} else {
		defer file.Close()
		sysconf, err := os.ReadFile("/etc/sysctl.conf")
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Unable to read sysctl.conf: %s", err.Error()))
		}
		if !strings.Contains(string(sysconf), "fs.file-max") {
			if _, err := file.WriteString("fs.file-max = 9000000\n"); err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("Unable to write fs.file-max: %s", err.Error()))
			}
		}
		if !strings.Contains(string(sysconf), "fs.inotify.max_user_watches") {
			if _, err := file.WriteString("fs.inotify.max_user_watches = 1000000\n"); err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("Unable to write fs.inotify.max_user_watches: %s", err.Error()))
			}
		}
		if !strings.Contains(string(sysconf), "fs.inotify.max_user_instances") {
			if _, err := file.WriteString("fs.inotify.max_user_instances = 1000000\n"); err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("Unable to write fs.inotify.max_user_instances: %s", err.Error()))
			}
		}
		if !strings.Contains(string(sysconf), "net.ipv4.ip_forward") {
			if _, err := file.WriteString("net.ipv4.ip_forward = 1\n"); err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("Unable to write net.ipv4.ip_forward: %s", err.Error()))
			}
		}
		err = exec.Command("sudo", "sysctl", "-p").Run()
		if err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Unable to set sysctl %s", err.Error()))
			return
		}
	}
}
