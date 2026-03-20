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

package build

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func (o *Options) BuildOnlineImage() {
	var err error
	// 需要构建镜像，不自己实现，直接调用docker构建镜像的命令
	if !infrastructure.IsDocker() {
		log.BKEFormat(log.ERROR, "This build instruction only supports running in docker environment.")
		return
	}

	log.BKEFormat("step.1", "Configuration file check")
	cfg := &BuildConfig{}
	yamlFile, err := os.ReadFile(o.File)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to read the file, %s", err.Error()))
		return
	}
	if err = yaml.Unmarshal(yamlFile, cfg); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Unable to serialize file, %s", err.Error()))
		return
	}

	log.BKEFormat("step.2", "Creates a workspace in the current directory")
	if err = prepare(); err != nil {
		return
	}

	stopChan := make(chan struct{})
	log.BKEFormat("step.3", "Collect host dependency packages and package files")
	if err = buildRpms(cfg, stopChan); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures %s", err.Error()))
		closeChanStruct(stopChan)
		return
	}
	// 构建镜像
	log.BKEFormat("step.4", fmt.Sprintf("Build the image %s ...", o.Target))
	err = buildImage(o.Target, o.Arch)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Build image failures %s", err.Error()))
		return
	}
	if err := os.RemoveAll(packages); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove packages directory: %s", err.Error()))
	}
	log.BKEFormat("step.5", "Push the image to the registry completed")
}

func buildImage(imageName string, arch string) error {
	err := os.Mkdir(pwd+"/bkesource", utils.DefaultDirPermission)
	if err != nil {
		return err
	}
	defer os.RemoveAll(pwd + "/bkesource")
	dockerfile := `
FROM scratch
COPY source.tar.gz /bkesource/source.tar.gz
`
	err = os.WriteFile(pwd+"/bkesource/Dockerfile", []byte(dockerfile), utils.DefaultFilePermission)
	if err != nil {
		return err
	}
	err = os.Rename(fmt.Sprintf("%s/%s", bke, utils.SourceDataFile), pwd+"/bkesource/source.tar.gz")
	if err != nil {
		return err
	}
	// 构建镜像并推送到仓库
	if strings.Contains(arch, runtime.GOARCH) && !strings.Contains(arch, ",") {
		log.BKEFormat(log.INFO, fmt.Sprintf("docker build -t %s .", imageName))
		output, err := global.Command.ExecuteCommandWithOutput("sh", "-c",
			fmt.Sprintf("cd %s/bkesource && docker build -t %s .", pwd, imageName))
		if err != nil {
			return errors.New(output + err.Error())
		}
		log.BKEFormat(log.INFO, fmt.Sprintf("docker push %s", imageName))
		output, err = global.Command.ExecuteCommandWithOutput("sh", "-c",
			fmt.Sprintf("docker push %s", imageName))
		if err != nil {
			return errors.New(output + err.Error())
		}
	} else {
		platform := ""
		for _, ar := range strings.Split(arch, ",") {
			if strings.HasPrefix("linux", ar) {
				platform += ar + ","
			} else {
				platform += "linux/" + ar + ","
			}
		}
		platform = strings.TrimSuffix(platform, ",")
		log.BKEFormat(log.INFO, fmt.Sprintf("docker buildx build --platform=%s -t %s . --push", platform, imageName))
		output, err := global.Command.ExecuteCommandWithOutput("sh", "-c",
			fmt.Sprintf("cd %s/bkesource && docker buildx build --platform=%s -t %s . --push", pwd, platform, imageName))
		if err != nil {
			return errors.New(output + err.Error())
		}
	}
	return nil
}
