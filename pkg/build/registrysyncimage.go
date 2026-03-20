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
	"fmt"
	"strings"

	reg "gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/pkg/server"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func syncRepo(cfg *BuildConfig, stopChan chan struct{}) error {
	var err error
	_ = server.RemoveImageRegistry(utils.LocalImageRegistryName)
	if err = server.StartImageRegistry(utils.LocalImageRegistryName, cfg.Registry.ImageAddress, "5000", tmpRegistry); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("The mirror warehouse fails to be started, %s", err.Error()))
		return err
	}

	for _, cr := range cfg.Repos {
		if !cr.NeedDownload {
			continue
		}
		if err = processRepoImages(cr, stopChan); err != nil {
			return err
		}
	}

	return packImageAndCleanup()
}

func processRepoImages(cr Repo, stopChan chan struct{}) error {
	for _, subImage := range cr.SubImages {
		if err := processSingleSubImage(subImage, cr.Architecture, stopChan); err != nil {
			return err
		}
	}
	return nil
}

func processSingleSubImage(subImage SubImage, architecture []string, stopChan chan struct{}) error {
	for _, image := range subImage.Images {
		select {
		case <-stopChan:
			log.BKEFormat(log.ERROR, "sync repo be externally terminated. ")
			return nil
		default:
		}
		if err := processImageTags(image, subImage, architecture); err != nil {
			return err
		}
	}
	return nil
}

func processImageTags(image Image, subImage SubImage, architecture []string) error {
	for _, tag := range image.Tag {
		source, err := imageTrack(subImage.SourceRepo, subImage.ImageTrack, image.Name, tag, architecture)
		if err != nil {
			return err
		}
		targetTag := tag
		if strings.Contains(tag, cut) {
			targetTag = strings.Split(tag, cut)[0]
		}
		target := fmt.Sprintf("127.0.0.1:5000/%s/%s:%s", subImage.TargetRepo, image.Name, targetTag)
		if subImage.TargetRepo == "/" {
			target = strings.ReplaceAll(target, "///", "/")
		} else {
			target = strings.ReplaceAll(target, "//", "/")
		}
		if err = syncRepoImage(source, target, architecture, true); err != nil {
			return err
		}
	}
	return nil
}

// syncRepoImage Synchronize an image between mirror repositories
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
func syncRepoImage(source, target string, arch []string, srcTLSVerify bool) error {
	if len(arch) == 1 {
		return syncSingleArchImage(source, target, arch[0], srcTLSVerify)
	}
	return syncMultiArchImage(source, target, arch, srcTLSVerify)
}

func syncSingleArchImage(source, target, arch string, srcTLSVerify bool) error {
	imageAddress := source
	if strings.Contains(imageAddress, cut) {
		imageAddress = strings.ReplaceAll(imageAddress, cut, fmt.Sprintf("-%s-", arch))
	}
	op := reg.Options{
		MultiArch:     false,
		SrcTLSVerify:  srcTLSVerify,
		DestTLSVerify: false,
		Arch:          arch,
		Source:        imageAddress,
		Target:        target,
	}

	log.BKEFormat(log.INFO, fmt.Sprintf("Sync image %s to %s", imageAddress, target))
	if err := reg.CopyRegistry(op); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Sync image %s to %s error %s", imageAddress, target, err.Error()))
		if strings.Contains(imageAddress, fmt.Sprintf("-%s-", arch)) {
			return err
		}
		return retrySyncWithArchSuffix(imageAddress, target, arch, op)
	}
	return nil
}

func retrySyncWithArchSuffix(imageAddress, target, arch string, op reg.Options) error {
	imageAddress = imageAddress + "-" + arch
	log.BKEFormat(log.INFO, fmt.Sprintf("Sync image %s to %s", imageAddress, target))
	op.Source = imageAddress
	if err := reg.CopyRegistry(op); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Sync image %s to %s error %s", imageAddress, target, err.Error()))
		return err
	}
	return nil
}

func syncMultiArchImage(source, target string, arch []string, srcTLSVerify bool) error {
	// 为了避免直接使用 CopyAllImages 拉取镜像清单中所有架构的镜像，
	// 这里不再走 tryDirectMultiArchSync，而是始终按传入的 arch 列表
	// 逐个架构拉取并重新创建仅包含这些架构的多架构 manifest。
	return syncArchImagesAndCreateManifest(source, target, arch, srcTLSVerify)
}

func tryDirectMultiArchSync(source, target string, srcTLSVerify bool) error {
	if strings.Contains(source, cut) || !reg.IsMultiArchManifests(srcTLSVerify, source) {
		return fmt.Errorf("not a direct multi-arch image")
	}
	op := reg.Options{
		MultiArch:     true,
		SrcTLSVerify:  srcTLSVerify,
		DestTLSVerify: false,
		Source:        source,
		Target:        target,
	}
	err := reg.CopyRegistry(op)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Sync image %s to %s error %s", source, target, err.Error()))
	}
	return err
}

func syncArchImagesAndCreateManifest(source, target string, arch []string, srcTLSVerify bool) error {
	var img []reg.ImageArch
	op := reg.Options{
		MultiArch:     false,
		SrcTLSVerify:  srcTLSVerify,
		DestTLSVerify: false,
	}
	for _, ar := range arch {
		archImg, err := syncSingleArchVariant(source, target, ar, op)
		if err != nil {
			return err
		}
		img = append(img, archImg)
	}
	if err := reg.CreateMultiArchImage(img, target); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf(
			"The creation of multiple schema images manifests fails %s %s", target, err.Error()))
		return err
	}
	return nil
}

func syncSingleArchVariant(source, target, arch string, op reg.Options) (reg.ImageArch, error) {
	op.Arch = arch
	op.Target = target + "-" + arch
	if strings.Contains(source, cut) {
		op.Source = strings.ReplaceAll(source, cut, fmt.Sprintf("-%s-", arch))
	}
	// 对于常见的多架构镜像（如 hub.oepkgs.net/openfuyao/registry:2.8.1），
	// 上游只提供无架构后缀的 manifest list，此时应直接使用原始 source，
	// 通过在 SystemContext 中设置 ArchitectureChoice（op.Arch）来选择具体架构，
	// 而不是构造不存在的 tag（例如 :2.8.1-amd64）。
	if op.Source == "" {
		op.Source = source
	}
	log.Debugf(op.Source + "-->" + op.Target)
	if err := reg.CopyRegistry(op); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Sync image %s to %s error %s", source, target, err.Error()))
		return reg.ImageArch{}, err
	}
	return reg.ImageArch{
		Name:         op.Target,
		OS:           "linux",
		Architecture: arch,
	}, nil
}
