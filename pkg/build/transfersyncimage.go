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
	"strings"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/docker"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/server"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

var (
	needRemoveImage = []string{}
	pushImageCount  = 0
)

// syncChannels 封装同步过程中使用的通道
type syncChannels struct {
	stopChan         <-chan struct{}
	internalStopChan chan struct{}
	pullCompleteChan chan struct{}
	imageChan        chan<- docker.ImageRef
}

// buildRegistry Package the image to a local file
// image collection of multiple architectures is supported in the following scenarios
// 1. The image itself is multi-architecture
// 2. The image is of a single architecture, and the suffix contains the -ARCH architecture
/*
multi-architecture image alpine:3.15
or
alpine:3.15-amd64
alpine:3.15-arm64
*/
func buildRegistry(source string, arch []string) error {
	var err error
	for _, ar := range arch {
		imageAddress := source
		log.BKEFormat(log.INFO, fmt.Sprintf("Try pulling away the mirror image %s", imageAddress))
		if err = global.Docker.Pull(docker.ImageRef{Image: imageAddress, Platform: ar},
			utils.RetryOptions{MaxRetry: 3, Delay: 1}); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to pull the mirror %s", err.Error()))
			imageAddress = source + "-" + ar
			log.BKEFormat(log.INFO, fmt.Sprintf("Try pulling away the mirror image %s", imageAddress))
			if err = global.Docker.Pull(docker.ImageRef{Image: imageAddress, Platform: ar},
				utils.RetryOptions{MaxRetry: 3, Delay: 1}); err != nil {
				log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to pull the mirror %s", err.Error()))
				return err
			}
		}
		if err = global.Docker.Tag(imageAddress, utils.DefaultLocalImageRegistry); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("docker tag %s %s error: %v", imageAddress,
				utils.DefaultLocalImageRegistry, err))
			return err
		}
		imageName := fmt.Sprintf("%s/%s-%s", bke, utils.ImageFile, ar)
		if err = global.Docker.Save(utils.DefaultLocalImageRegistry, imageName); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("docker save %s %s error: %v", utils.DefaultLocalImageRegistry,
				fmt.Sprintf("%s/%s-%s", bke, utils.ImageFile, ar), err))
			return err
		}
		if err = global.Docker.Remove(docker.ImageRef{Image: imageAddress}); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("docker rmi %s error: %v", imageAddress, err))
			return err
		}
		if err = global.Docker.Remove(docker.ImageRef{Image: utils.DefaultLocalImageRegistry}); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("docker rmi %s error: %v", utils.DefaultLocalImageRegistry, err))
			return err
		}
	}
	needRemoveImage = append(needRemoveImage, source)
	return nil
}

// syncImageTag 同步单个镜像标签
func syncImageTag(subImage SubImage, image Image, imageTag string, cr Repo, imageChan chan<- docker.ImageRef) error {
	sou, err := imageTrack(subImage.SourceRepo, subImage.ImageTrack, image.Name, imageTag, cr.Architecture)
	if err != nil {
		return err
	}
	targetTag := imageTag
	if strings.Contains(imageTag, cut) {
		targetTag = strings.Split(imageTag, cut)[0]
	}
	tgt := fmt.Sprintf("127.0.0.1:5000/%s/%s:%s", subImage.TargetRepo, image.Name, targetTag)
	if subImage.TargetRepo == "/" {
		tgt = strings.ReplaceAll(tgt, "///", "/")
	} else {
		tgt = strings.ReplaceAll(tgt, "//", "/")
	}
	return syncImage(sou, tgt, cr.Architecture, imageChan)
}

// collectRepo Collect the images listed in the configuration file
func collectRepo(cfg *BuildConfig, stopChan <-chan struct{}) error {
	var err error
	_ = server.RemoveImageRegistry(utils.LocalImageRegistryName)
	if err = server.StartImageRegistry(utils.LocalImageRegistryName, cfg.Registry.ImageAddress, "5000", tmpRegistry); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("The mirror warehouse fails to be started, %s", err.Error()))
		return err
	}

	imageChan := make(chan docker.ImageRef, 100)
	internalStopChan := make(chan struct{})
	pullCompleteChan := make(chan struct{})
	pushCompleteChan := make(chan string)
	defer func() {
		if !utils.IsChanClosed(imageChan) {
			close(imageChan)
		}
	}()
	defer closeChanStruct(pullCompleteChan)

	go pushImage(imageChan, pullCompleteChan, pushCompleteChan, internalStopChan)

	channels := &syncChannels{
		stopChan:         stopChan,
		internalStopChan: internalStopChan,
		pullCompleteChan: pullCompleteChan,
		imageChan:        imageChan,
	}
	if err = syncAllRepoImages(cfg, channels); err != nil {
		return err
	}

	closeChanStruct(pullCompleteChan)
	pushResult := <-pushCompleteChan
	if len(pushResult) > 0 {
		return errors.New(pushResult)
	}

	return packImageAndCleanup()
}

// syncRepoImageTags 同步单个仓库的所有镜像标签
func syncRepoImageTags(cr Repo, subImage SubImage, channels *syncChannels) error {
	for _, image := range subImage.Images {
		select {
		case <-channels.stopChan:
			closeChanStruct(channels.internalStopChan)
			log.BKEFormat(log.ERROR, "pull image be externally terminated. ")
			return nil
		default:
		}
		for _, imageTag := range image.Tag {
			if err := syncImageTag(subImage, image, imageTag, cr, channels.imageChan); err != nil {
				closeChanStruct(channels.internalStopChan)
				closeChanStruct(channels.pullCompleteChan)
				return err
			}
		}
	}
	return nil
}

// syncAllRepoImages 同步所有仓库镜像
func syncAllRepoImages(cfg *BuildConfig, channels *syncChannels) error {
	for _, cr := range cfg.Repos {
		if !cr.NeedDownload {
			continue
		}
		for _, subImage := range cr.SubImages {
			if err := syncRepoImageTags(cr, subImage, channels); err != nil {
				return err
			}
		}
	}
	return nil
}

// pullAndTagSingleArchImage 拉取并标记单架构镜像
func pullAndTagSingleArchImage(source, target, arch string) error {
	imageAddress := source
	if strings.Contains(imageAddress, cut) {
		imageAddress = strings.ReplaceAll(imageAddress, cut, fmt.Sprintf("-%s-", arch))
	}
	log.BKEFormat(log.INFO, fmt.Sprintf("Try pulling away the mirror image %s", imageAddress))
	if err := global.Docker.Pull(docker.ImageRef{
		Image:    imageAddress,
		Platform: arch,
	}, utils.RetryOptions{MaxRetry: 3, Delay: 1}); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to pull the mirror %s", err.Error()))
		if strings.Contains(imageAddress, fmt.Sprintf("-%s-", arch)) {
			return err
		}
		imageAddress = source + "-" + arch
		log.BKEFormat(log.INFO, fmt.Sprintf("Try pulling away the mirror image %s", imageAddress))
		if err = global.Docker.Pull(docker.ImageRef{
			Image:    imageAddress,
			Platform: arch,
		}, utils.RetryOptions{MaxRetry: 3, Delay: 1}); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to pull the mirror %s", err.Error()))
			return err
		}
	}
	if err := global.Docker.Tag(imageAddress, target); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("docker tag %s %s error: %v", imageAddress, target, err))
		return err
	}
	needRemoveImage = append(needRemoveImage, imageAddress)
	needRemoveImage = append(needRemoveImage, target)
	return nil
}

// pullAndPushMultiArchImage 拉取并推送多架构镜像
func pullAndPushMultiArchImage(source, target string, arch []string) (string, []string, error) {
	manifestCreateCmd := fmt.Sprintf("docker manifest create --insecure %s", target)
	var manifestAnnotate []string

	for _, ar := range arch {
		imageAddress := source
		if strings.Contains(imageAddress, cut) {
			imageAddress = strings.ReplaceAll(imageAddress, cut, fmt.Sprintf("-%s-", ar))
		}
		log.BKEFormat(log.INFO, fmt.Sprintf("Try pulling away the mirror image %s", imageAddress))
		if err := global.Docker.Pull(docker.ImageRef{
			Image:    imageAddress,
			Platform: ar,
		}, utils.RetryOptions{MaxRetry: 3, Delay: 1}); err != nil {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to pull the mirror %s", err.Error()))
			if strings.Contains(imageAddress, fmt.Sprintf("-%s-", ar)) {
				return "", nil, err
			}
			imageAddress = source + "-" + ar
			log.BKEFormat(log.INFO, fmt.Sprintf("Try pulling away the mirror image %s", imageAddress))
			if err = global.Docker.Pull(docker.ImageRef{
				Image:    imageAddress,
				Platform: ar,
			}, utils.RetryOptions{MaxRetry: 3, Delay: 1}); err != nil {
				log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to pull the mirror %s", err.Error()))
				return "", nil, err
			}
		}

		targetArch := fmt.Sprintf("%s-%s", target, ar)
		if err := global.Docker.Tag(imageAddress, targetArch); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("docker tag %s %s error: %v", imageAddress, target, err))
			return "", nil, err
		}
		log.Infof("docker tag %s %s", imageAddress, targetArch)
		if err := global.Docker.Remove(docker.ImageRef{Image: imageAddress}); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Image cannot be removed %s %s", imageAddress, err.Error()))
			return "", nil, err
		}
		if err := global.Docker.Push(docker.ImageRef{Image: targetArch}); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Image push faile %s %s", targetArch, err.Error()))
			return "", nil, err
		}
		needRemoveImage = append(needRemoveImage, targetArch)

		manifestCreateCmd += fmt.Sprintf("  --amend %s", targetArch)
		manifestAnnotate = append(manifestAnnotate, fmt.Sprintf("docker manifest annotate %s --os linux --arch %s %s", target, ar, targetArch))
	}
	return manifestCreateCmd, manifestAnnotate, nil
}

// executeManifestCommands 执行 manifest 相关命令
func executeManifestCommands(target, manifestCreateCmd string, manifestAnnotate []string) error {
	manifestEnv := os.Environ()
	manifestEnv = append(manifestEnv, "DOCKER_CLI_EXPERIMENTAL=enabled")

	if err := global.Command.ExecuteCommandWithEnv(manifestEnv, "/bin/sh", "-c", manifestCreateCmd); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("%s, %v", manifestCreateCmd, err))
		return err
	}
	log.Infof("%s", manifestCreateCmd)

	for _, cmd := range manifestAnnotate {
		if err := global.Command.ExecuteCommandWithEnv(manifestEnv, "/bin/sh", "-c", cmd); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("%s, %v", cmd, err))
			return err
		}
		log.BKEFormat(log.INFO, fmt.Sprintf("%s", cmd))
	}

	manifestPushCmd := fmt.Sprintf("docker manifest push --purge --insecure %s", target)
	if err := global.Command.ExecuteCommandWithEnv(manifestEnv, "/bin/sh", "-c", manifestPushCmd); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("%s, %v", manifestPushCmd, err))
		return err
	}
	log.BKEFormat(log.INFO, fmt.Sprintf("%s", manifestPushCmd))
	return nil
}

// syncImage Synchronize an image between mirror repositories
// image collection of multiple architectures is supported in the following scenarios
// 1. The image itself is multi-architecture
// 2. The image is of a single architecture, and the suffix contains the -ARCH architecture
/*
multi-architecture image alpine:3.15
or
alpine:3.15-amd64
alpine:3.15-arm64
alpine:3.15-*-202112111112
*/
func syncImage(source, target string, arch []string, imageChan chan<- docker.ImageRef) error {
	// Single architecture
	if len(arch) == 1 {
		if err := pullAndTagSingleArchImage(source, target, arch[0]); err != nil {
			return err
		}
		pushImageCount += 1
		imageChan <- docker.ImageRef{Image: target, Platform: arch[0]}
		return nil
	}

	// More than architecture
	manifestCreateCmd, manifestAnnotate, err := pullAndPushMultiArchImage(source, target, arch)
	if err != nil {
		return err
	}

	return executeManifestCommands(target, manifestCreateCmd, manifestAnnotate)
}

func cleanBuildImage() {
	for _, image := range needRemoveImage {
		_ = global.Docker.Remove(docker.ImageRef{
			Image: image,
		})
	}
}

func pushImage(imageChan <-chan docker.ImageRef, pullCompleteChan <-chan struct{},
	pushCompleteChan chan<- string, stopChan <-chan struct{}) {
	pullCompleteFlag := false
	pushCompleteCount := 0
	manifestEnv := os.Environ()
	manifestEnv = append(manifestEnv, "DOCKER_CLI_EXPERIMENTAL=enabled")
	for {
		if pullCompleteFlag && pushImageCount == pushCompleteCount {
			log.BKEFormat(log.INFO, "push to complete. ")
			pushCompleteChan <- ""
			return
		}
		select {
		case _, ok := <-pullCompleteChan:
			if !ok {
				log.BKEFormat(log.WARN, "pull complete channel closed")
			}
			pullCompleteFlag = true
		case image := <-imageChan:
			log.BKEFormat(log.INFO, fmt.Sprintf("docker push %s", image.Image))
			err := global.Docker.Push(image)
			if err != nil {
				log.BKEFormat(log.ERROR, fmt.Sprintf("docker push %s error: %v", image.Image, err))
				pushCompleteChan <- err.Error()
			}
			pushCompleteCount += 1
		case _, ok := <-stopChan:
			if !ok {
				log.BKEFormat(log.WARN, "stop channel closed")
			}
			log.BKEFormat(log.INFO, "push image be external termination. ")
			pushCompleteChan <- "push image be external termination."
		}
	}
}

func closeChanStruct(ch chan struct{}) {
	if !utils.IsChanClosed(ch) {
		close(ch)
	}
}

// packImageAndCleanup 打包镜像文件并清理
func packImageAndCleanup() error {
	log.BKEFormat(log.INFO, "The system starts to pack the image file.")
	if err := global.TarGZ(tmpRegistry, fmt.Sprintf("%s/%s", bke, utils.ImageDataFile)); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("tar %s error %s",
			fmt.Sprintf("%s/%s", bke, utils.ImageDataFile), err.Error()))
		return err
	}
	_ = server.RemoveImageRegistry(utils.LocalImageRegistryName)
	return nil
}
