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
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/strslice"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type RpmOptions struct {
	root.Options
	Source   string `json:"source"`
	Add      string `json:"add"`
	Registry string `json:"registry"`
	Package  string `json:"package"`
}

const (
	// addListMinParts 表示 adds 路径拆分后的最小部分数 (OS/Version/Arch)
	addListMinParts = 3
)

var adds = map[string]string{
	"centos/7/amd64":  "CentOS/7/amd64",
	"centos/7/arm64":  "CentOS/7/arm64",
	"centos/8/amd64":  "CentOS/8/amd64",
	"centos/8/arm64":  "CentOS/8/arm64",
	"ubuntu/22/amd64": "Ubuntu/22/amd64",
	"ubuntu/22/arm64": "Ubuntu/22/arm64",
	"kylin/v10/arm64": "Kylin/V10/arm64",
	"kylin/v10/amd64": "Kylin/V10/amd64",
}

// compressAndCleanupRpm handles the common rpm packaging workflow:
// removes old archive, compresses packages, sets permissions, and cleans up
func compressAndCleanupRpm(targetFile string, successMsg string) error {
	if err := os.RemoveAll(targetFile); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to remove file %s %s", targetFile, err.Error()))
		return err
	}
	if err := global.TarGZ(packages, targetFile); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm %s", err.Error()))
		return err
	}
	if err := os.Chmod(targetFile, utils.DefaultDirPermission); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to chmod %s: %s", targetFile, err.Error()))
	}
	if err := os.RemoveAll(packages); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to remove file %s %s", packages, err.Error()))
		return err
	}
	log.BKEFormat(log.INFO, fmt.Sprintf("%s %s", successMsg, targetFile))
	return nil
}

// cleanRepodata 清理指定目录下的 repodata 相关文件
func cleanRepodata(mnt string) error {
	entries, err := os.ReadDir(mnt)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to read the directory, %s", err.Error()))
		return err
	}

	if err := os.RemoveAll(path.Join(mnt, "repodata")); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove repodata: %s", err.Error()))
	}
	if err := os.RemoveAll(path.Join(mnt, ".repodata")); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove .repodata: %s", err.Error()))
	}

	for _, entry := range entries {
		if err := os.RemoveAll(path.Join(mnt, entry.Name(), "repodata")); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove %s/repodata: %s", entry.Name(), err.Error()))
		}
		if err := os.RemoveAll(path.Join(mnt, ".repodata")); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove .repodata: %s", err.Error()))
		}
	}
	return nil
}

// runBuildContainer 在指定目录中运行构建命令
func runBuildContainer(image, mnt, containerName, cmd string) error {
	return global.Docker.Run(
		&container.Config{
			Image:      image,
			WorkingDir: "/opt/mnt",
			Cmd: strslice.StrSlice{
				"sh",
				"-c",
				cmd,
			},
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:   mount.TypeBind,
					Source: mnt,
					Target: "/opt/mnt",
				},
			},
			RestartPolicy: container.RestartPolicy{
				Name:              "no",
				MaximumRetryCount: 0,
			},
		}, nil, nil, containerName)
}

// waitForContainerComplete waits for a container to finish running
func waitForContainerComplete(containerName string) {
	for {
		time.Sleep(utils.ContainerWaitSeconds * time.Second)
		containerInfo, _ := global.Docker.GetClient().ContainerInspect(context.TODO(), containerName)
		if containerInfo.ContainerJSONBase != nil {
			if !containerInfo.State.Running {
				break
			}
		} else {
			break
		}
	}
}

// ensureRpmBuildImage ensures the RPM build image exists
func ensureRpmBuildImage(registry, imageTag string) (string, error) {
	image := registry + "/" + imageTag
	err := global.Docker.EnsureImageExists(docker.ImageRef{
		Image: image,
	}, utils.RetryOptions{MaxRetry: 3, Delay: 1})
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to pull the mirror %s", err.Error()))
		return "", err
	}
	return image, nil
}

// executeRpmBuildContainer executes RPM build container and waits for completion
func executeRpmBuildContainer(image, mnt, containerName, cmd string) error {
	_ = global.Docker.ContainerRemove(containerName)
	err := runBuildContainer(image, mnt, containerName, cmd)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm %s", err.Error()))
		return err
	}
	defer func() {
		_ = global.Docker.ContainerRemove(containerName)
	}()
	waitForContainerComplete(containerName)
	return nil
}

// verifyRpmBuildResult verifies that required files exist after build
func verifyRpmBuildResult(mnt, osInfo string, requiredFiles ...string) error {
	for _, file := range requiredFiles {
		if !utils.Exists(path.Join(mnt, file)) {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build the %s %s rpm package", osInfo, mnt))
			return fmt.Errorf("%s not found", file)
		}
	}
	return nil
}

// rpmBuildConfig holds the configuration for building rpm packages
type rpmBuildConfig struct {
	registry      string
	mnt           string
	image         string
	containerName string
	cmd           string
	checkFile     string
	osInfo        string
}

// validateAddOption 验证 add 选项是否有效
func validateAddOption(add string) bool {
	_, ok := adds[add]
	return ok
}

// validatePackageDirectory 验证 package 目录是否有效
func validatePackageDirectory(pack string, add string) error {
	if !utils.IsDir(pack) {
		return fmt.Errorf("the %s is not a directory", pack)
	}

	entries, err := os.ReadDir(pack)
	if err != nil {
		return fmt.Errorf("failed to read the directory, %s", err.Error())
	}

	for _, entry := range entries {
		if strings.HasPrefix(add, "centos") && entry.Name() == "modules.yaml" {
			continue
		}
		if strings.HasPrefix(add, "ubuntu") && entry.Name() == "Packages.gz" {
			continue
		}
		if !entry.IsDir() {
			return fmt.Errorf("the %s/%s is not a directory", pack, entry.Name())
		}
	}
	return nil
}

// getAbsolutePath 获取绝对路径
func getAbsolutePath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get the absolute path, %s", err.Error())
	}
	return absPath, nil
}

func (ro *RpmOptions) Build() {
	if len(ro.Source) == 0 && len(ro.Add) == 0 && len(ro.Package) == 0 {
		consoleOutputStruct()
		return
	}

	if len(ro.Add) != 0 {
		ro.Add = strings.ToLower(ro.Add)
		if !validateAddOption(ro.Add) {
			log.BKEFormat(log.ERROR, fmt.Sprintf("The %s is not a valid add. ", ro.Add))
			log.BKEFormat(log.INFO, "Valid add: centos/7/amd64, centos/7/arm64, centos/8/amd64, centos/8/arm64, ubuntu/22/amd64, ubuntu/22/arm64, kylin/v10/arm64, kylin/v10/amd64")
			return
		}
	}

	if len(ro.Package) != 0 {
		if err := validatePackageDirectory(ro.Package, ro.Add); err != nil {
			log.BKEFormat(log.ERROR, err.Error())
			return
		}
		absPath, err := getAbsolutePath(ro.Package)
		if err != nil {
			log.BKEFormat(log.ERROR, err.Error())
			return
		}
		ro.Package = absPath
	}

	if len(ro.Source) != 0 {
		absPath, err := getAbsolutePath(ro.Source)
		if err != nil {
			log.BKEFormat(log.ERROR, err.Error())
			return
		}
		ro.Source = absPath
	}

	if !infrastructure.IsDocker() {
		log.BKEFormat(log.ERROR, "This build instruction only supports running in docker environment.")
		return
	}

	ro.executeBuild()
}

// executeBuild 执行具体的构建逻辑
func (ro *RpmOptions) executeBuild() {
	// 为指定包构建rpm
	if len(ro.Source) == 0 {
		if err := rmpBuild(ro.Registry, ro.Add, ro.Package); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm, %s", err.Error()))
		}
		return
	}

	// 整个包裹构建rpm包
	if len(ro.Source) != 0 && len(ro.Add) == 0 && len(ro.Package) == 0 {
		rpmBuildPackage(ro.Source, ro.Registry)
		return
	}

	// 为压缩包添加指定rpm包
	if len(ro.Source) != 0 && len(ro.Add) != 0 && len(ro.Package) != 0 {
		rpmPackageAddOne(ro.Source, ro.Registry, ro.Add, ro.Package)
		return
	}
}

func prepareWorkspace() error {
	pathLevel := []string{
		packages,
		path.Join(packages, "files"),
		path.Join(packages, "CentOS", "7", "amd64"),
		path.Join(packages, "CentOS", "7", "arm64"),
		path.Join(packages, "CentOS", "8", "amd64"),
		path.Join(packages, "CentOS", "8", "arm64"),
		path.Join(packages, "Ubuntu", "22", "amd64"),
		path.Join(packages, "Ubuntu", "22", "arm64"),
		path.Join(packages, "Kylin", "V10", "amd64"),
		path.Join(packages, "Kylin", "V10", "arm64"),
	}
	if err := os.RemoveAll(packages); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Unable to remove file %s %s", packages, err.Error()))
		return err
	}
	for _, pl := range pathLevel {
		if err := os.MkdirAll(pl, utils.DefaultDirPermission); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Unable to create file %s %s", pl, err.Error()))
			return err
		}
	}
	return nil
}

func consoleOutputStruct() {
	content := `rpm
├── CentOS
│   ├── 7
│   │   ├── amd64
│   │   └── arm64
│   └── 8
│       ├── amd64
│       └── arm64
├── files
├── Kylin
│   └── V10
│       ├── amd64
│       └── arm64
└── Ubuntu
    └── 22
        ├── amd64
        └── arm64
`
	fmt.Print(content)
}

// rpmBuild
// 为当前目录制作rpm源文件
func rmpBuild(registry string, add string, absPath string) error {
	switch add {
	case "centos/7/amd64", "centos/7/arm64":
		err := rpmCentos7Build(registry, absPath)
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm %s", err.Error()))
			return err
		}
	case "centos/8/amd64", "centos/8/arm64":
		err := rpmCentos8Build(registry, absPath)
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm %s", err.Error()))
			return err
		}
	case "ubuntu/22/amd64", "ubuntu/22/arm64":
		err := rpmUbuntu22Build(registry, absPath)
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm %s", err.Error()))
			return err
		}
	case "kylin/v10/amd64", "kylin/v10/arm64":
		err := rpmKylinV10Build(registry, absPath)
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm %s", err.Error()))
			return err
		}
	default:
		log.BKEFormat(log.ERROR, fmt.Sprintf("The %s is not supported", add))
		return errors.New("not supported")
	}
	log.BKEFormat(log.INFO, fmt.Sprintf("Successfully build rpm %s", add))
	return nil
}

// target 表示一个构建目标配置
type target struct {
	osName  string
	version string
	arch    string
	builder func(string, string) error
}

// getTargets 返回所有构建目标的配置
func getTargets() []target {
	return []target{
		{"Centos", "7", "amd64", rpmCentos7Build},
		{"Centos", "7", "arm64", rpmCentos7Build},
		{"Centos", "8", "amd64", rpmCentos8Build},
		{"Centos", "8", "arm64", rpmCentos8Build},
		{"Ubuntu", "22", "amd64", rpmUbuntu22Build},
		{"Ubuntu", "22", "arm64", rpmUbuntu22Build},
		{"Kylin", "V10", "amd64", rpmKylinV10Build},
		{"Kylin", "V10", "arm64", rpmKylinV10Build},
	}
}

// executeSingleTarget 构建单个目标
func executeSingleTarget(registry string, t target) error {
	targetPath := path.Join(packages, t.osName, t.version, t.arch)
	return t.builder(registry, targetPath)
}

// rpmBuildAllArchitectures 为所有架构构建 rpm 包
func rpmBuildAllArchitectures(registry string) error {
	targets := getTargets()
	for _, t := range targets {
		if err := executeSingleTarget(registry, t); err != nil {
			return fmt.Errorf("failed to build rpm for %s/%s/%s: %w",
				t.osName, t.version, t.arch, err)
		}
	}
	return nil
}

// rpmBuildPackage build all package of rpm
// 为整个目录构建一个完整的rpm.tar.gz
func rpmBuildPackage(source string, registry string) {
	if !utils.IsDir(source) {
		log.BKEFormat(log.ERROR, fmt.Sprintf("The %s is not a directory. ", source))
		return
	}

	if err := prepareWorkspace(); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to prepare workspace %s", err.Error()))
		return
	}

	if err := utils.CopyDir(source, packages); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to copy file %s %s", source, err.Error()))
		return
	}

	if err := rpmBuildAllArchitectures(registry); err != nil {
		log.BKEFormat(log.ERROR, err.Error())
		return
	}

	if err := compressAndCleanupRpm(path.Join(pwd, "rpm.tar.gz.1"), "Build rpm success."); err != nil {
		return
	}
}

// rpmPackageAddOne 添加一个rpm包
// 为已经打包号的rpm.tar.gz，增加一个包
func rpmPackageAddOne(source string, registry string, add string, pack string) {
	if !utils.IsFile(source) {
		log.BKEFormat(log.ERROR, fmt.Sprintf("The %s is not a file. ", source))
		return
	}
	err := os.RemoveAll(packages)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to remove file %s %s", packages, err.Error()))
		return
	}
	err = os.MkdirAll(packages, utils.DefaultDirPermission)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to create file %s %s", packages, err.Error()))
		return
	}

	err = utils.UnTar(source, packages)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to unTar %s %s", source, err.Error()))
		return
	}
	addList := strings.Split(adds[add], "/")
	if len(addList) < addListMinParts {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Invalid add format, expected at least 3 parts separated by '/'"))
		return
	}
	err = utils.CopyDir(pack, path.Join(packages, addList[0], addList[1], addList[2]))
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to copy file %s %s", pack, err.Error()))
		return
	}

	err = rmpBuild(registry, add, path.Join(packages, addList[0], addList[1], addList[2]))
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm %s", err.Error()))
		return
	}

	// 打包
	if err = compressAndCleanupRpm(
		path.Join(pwd, "rpm.tar.gz.1"), "The rpm package has been successfully built"); err != nil {
		return
	}
}

func rpmCentos8Build(registry string, mnt string) error {
	if utils.DirectoryIsEmpty(mnt) {
		return nil
	}

	image, err := ensureRpmBuildImage(registry, "centos:8-amd64-build")
	if err != nil {
		return err
	}

	if err := cleanCentos8Modules(mnt); err != nil {
		return err
	}

	cmd := "createrepo ./ && repo2module -s stable . modules.yaml && " +
		"modifyrepo_c --mdtype=modules modules.yaml repodata/"
	if err := executeRpmBuildContainer(image, mnt, "build-centos8-rpm", cmd); err != nil {
		return err
	}

	return verifyRpmBuildResult(mnt, "centos/8/amd64", "modules.yaml", "repodata")
}

// cleanCentos8Modules cleans modules.yaml and repodata for CentOS 8 builds
func cleanCentos8Modules(mnt string) error {
	entries, err := os.ReadDir(mnt)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to read the directory, %s", err.Error()))
		return err
	}

	// Clean root level files
	for _, f := range []string{"modules.yaml", "repodata", ".repodata"} {
		if err := os.RemoveAll(path.Join(mnt, f)); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove %s: %s", f, err.Error()))
		}
	}

	// Clean subdirectory files
	for _, entry := range entries {
		for _, f := range []string{"modules.yaml", "repodata"} {
			if err := os.RemoveAll(path.Join(mnt, entry.Name(), f)); err != nil {
				log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove %s/%s: %s", entry.Name(), f, err.Error()))
			}
		}
	}
	return nil
}

func executeGenericRpmBuild(cfg rpmBuildConfig) error {
	if utils.DirectoryIsEmpty(cfg.mnt) {
		return nil
	}

	image, err := ensureRpmBuildImage(cfg.registry, cfg.image)
	if err != nil {
		return err
	}

	if err := cleanRepodata(cfg.mnt); err != nil {
		return err
	}

	if err := executeRpmBuildContainer(image, cfg.mnt, cfg.containerName, cfg.cmd); err != nil {
		return err
	}

	return verifyRpmBuildResult(cfg.mnt, cfg.osInfo, cfg.checkFile)
}

func rpmCentos7Build(registry string, mnt string) error {
	return executeGenericRpmBuild(rpmBuildConfig{
		registry:      registry,
		mnt:           mnt,
		image:         "centos:7-amd64-build",
		containerName: "build-centos7-rpm",
		cmd:           "createrepo ./",
		osInfo:        "centos/7/amd64",
		checkFile:     "repodata",
	})
}

func rpmUbuntu22Build(registry string, mnt string) error {
	if utils.DirectoryIsEmpty(mnt) {
		return nil
	}
	image := registry + "/ubuntu:22-amd64-build"
	err := global.Docker.EnsureImageExists(docker.ImageRef{
		Image: image,
	}, utils.RetryOptions{MaxRetry: 3, Delay: 1})
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to pull the mirror %s", err.Error()))
		return err
	}
	if err := os.RemoveAll(path.Join(mnt, "Packages.gz")); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove Packages.gz: %s", err.Error()))
	}
	if err := os.RemoveAll(path.Join(mnt, "archives", "Packages.gz")); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove archives/Packages.gz: %s", err.Error()))
	}

	name := "build-ubuntu22-rpm"
	_ = global.Docker.ContainerRemove(name)
	// Starting the image repository
	err = runBuildContainer(image, mnt, name,
		"dpkg-scanpackages -m . /dev/null | gzip -9c > Packages.gz && cp Packages.gz ./archives")
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build rpm %s", err.Error()))
		return err
	}
	defer func() {
		_ = global.Docker.ContainerRemove(name)
	}()
	waitForContainerComplete(name)

	if !utils.Exists(path.Join(mnt, "Packages.gz")) {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build the ubuntu/22/amd64 %s rpm package", mnt))
		return errors.New("packages.gz not found")
	}
	return nil
}

func rpmKylinV10Build(registry string, mnt string) error {
	return executeGenericRpmBuild(rpmBuildConfig{
		registry:      registry,
		mnt:           mnt,
		image:         "centos:7-amd64-build",
		containerName: "build-kylin10-rpm",
		cmd:           "createrepo ./",
		osInfo:        "kylin/v10/amd64",
		checkFile:     "repodata",
	})
}
