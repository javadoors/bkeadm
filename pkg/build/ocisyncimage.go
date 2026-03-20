/*
 * Copyright (c) 2025 Huawei Technologies Co., Ltd.
 * openFuyao is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          http://license.coscl.org.cn/MulanPSL2
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	reg "gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

// syncContext 封装同步上下文参数
type syncContext struct {
	ociDir       string
	stopChan     chan struct{}
	currentImage *int
	totalImages  int
}

// 使用OCI layout格式同步镜像，无需Docker/Containerd
func syncRepoOCI(cfg *BuildConfig, stopChan chan struct{}) error {
	log.BKEFormat(log.INFO, "Using OCI layout strategy (no Docker required)")

	ociDir, err := createOCILayoutStructure()
	if err != nil {
		return err
	}

	totalImages := countTotalImages(cfg)
	log.BKEFormat(log.INFO, fmt.Sprintf("Total images to sync: %d", totalImages))

	if err := syncAllImagesToOCI(cfg, ociDir, stopChan, totalImages); err != nil {
		return err
	}

	return moveOCILayoutToVolumes(ociDir)
}

func createOCILayoutStructure() (string, error) {
	ociDir := filepath.Join(tmpRegistry, "oci-layout")
	if err := os.MkdirAll(ociDir, utils.DefaultDirPermission); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to create OCI directory: %s", err.Error()))
		return "", err
	}

	blobsDir := filepath.Join(ociDir, "blobs", "sha256")
	if err := os.MkdirAll(blobsDir, utils.DefaultDirPermission); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to create blobs directory: %s", err.Error()))
		return "", err
	}

	ociLayoutFile := filepath.Join(ociDir, "oci-layout")
	ociLayoutContent := `{"imageLayoutVersion":"1.0.0"}`
	if err := os.WriteFile(ociLayoutFile, []byte(ociLayoutContent), utils.DefaultFilePermission); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to create oci-layout file: %s", err.Error()))
		return "", err
	}
	log.BKEFormat(log.INFO, "Created standard OCI layout structure")
	return ociDir, nil
}

func countTotalImages(cfg *BuildConfig) int {
	totalImages := 0
	for _, cr := range cfg.Repos {
		if !cr.NeedDownload {
			continue
		}
		for _, subImage := range cr.SubImages {
			for _, image := range subImage.Images {
				totalImages += len(image.Tag)
			}
		}
	}
	return totalImages
}

func syncAllImagesToOCI(cfg *BuildConfig, ociDir string, stopChan chan struct{}, totalImages int) error {
	currentImage := 0
	for _, cr := range cfg.Repos {
		if !cr.NeedDownload {
			continue
		}
		if err := syncRepoImages(cr, ociDir, stopChan, &currentImage, totalImages); err != nil {
			return err
		}
	}
	return nil
}

func syncRepoImages(cr Repo, ociDir string, stopChan chan struct{}, currentImage *int, totalImages int) error {
	for _, subImage := range cr.SubImages {
		ctx := &syncContext{
			ociDir:       ociDir,
			stopChan:     stopChan,
			currentImage: currentImage,
			totalImages:  totalImages,
		}
		if err := syncSubImages(subImage, cr.Architecture, ctx); err != nil {
			return err
		}
	}
	return nil
}

func syncSubImages(subImage SubImage, arch []string, ctx *syncContext) error {
	for _, image := range subImage.Images {
		select {
		case <-ctx.stopChan:
			log.BKEFormat(log.ERROR, "sync repo be externally terminated.")
			return nil
		default:
		}
		if err := syncOCIImageTags(image, subImage, arch, ctx); err != nil {
			return err
		}
	}
	return nil
}

func syncOCIImageTags(image Image, subImage SubImage, arch []string, ctx *syncContext) error {
	for _, tag := range image.Tag {
		*ctx.currentImage++
		log.BKEFormat(log.INFO, fmt.Sprintf("Progress: [%d/%d] Syncing image",
			*ctx.currentImage, ctx.totalImages))

		source, err := imageTrack(subImage.SourceRepo, subImage.ImageTrack, image.Name, tag, arch)
		if err != nil {
			return err
		}

		targetTag := tag
		if strings.Contains(tag, cut) {
			targetTag = strings.Split(tag, cut)[0]
		}

		imageRef := formatImageRef(subImage.TargetRepo, image.Name, targetTag)

		if err := syncImageToOCI(source, ctx.ociDir, imageRef, arch, true); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to sync image %s: %s", source, err.Error()))
			return err
		}
	}
	return nil
}

func formatImageRef(targetRepo, imageName, tag string) string {
	if targetRepo == "/" {
		return fmt.Sprintf("%s:%s", imageName, tag)
	}
	return fmt.Sprintf("%s/%s:%s", targetRepo, imageName, tag)
}

func moveOCILayoutToVolumes(ociDir string) error {
	log.BKEFormat(log.INFO, "Moving OCI layout to volumes directory...")
	volumesOCIDir := filepath.Join(bkeVolumes, "oci-layout")

	if err := os.RemoveAll(volumesOCIDir); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to remove old OCI directory: %s", err.Error()))
		return err
	}

	if err := os.Rename(ociDir, volumesOCIDir); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to move OCI layout: %s", err.Error()))
		return err
	}

	log.BKEFormat(log.INFO, "OCI layout created successfully at volumes/oci-layout/")
	return nil
}

func syncImageToOCI(source, ociLayoutDir, imageRef string, arch []string, srcTLSVerify bool) error {
	imageAddress := source

	// 单架构处理
	if len(arch) == 1 {
		if strings.Contains(imageAddress, cut) {
			imageAddress = strings.ReplaceAll(imageAddress, cut, fmt.Sprintf("-%s-", arch[0]))
		}

		log.BKEFormat(log.INFO, fmt.Sprintf("Syncing image %s to OCI layout as %s", imageAddress, imageRef))

		if err := copyImageToOCI(imageAddress, ociLayoutDir, imageRef, arch[0], srcTLSVerify); err != nil {
			log.BKEFormat(log.DEBUG, fmt.Sprintf("Sync image %s failed, retrying: %s", imageAddress, err.Error()))
			if strings.Contains(imageAddress, fmt.Sprintf("-%s-", arch[0])) {
				return err
			}
			imageAddress = imageAddress + "-" + arch[0]
			log.BKEFormat(log.DEBUG, fmt.Sprintf("Retrying with %s", imageAddress))
			if err = copyImageToOCI(imageAddress, ociLayoutDir, imageRef, arch[0], srcTLSVerify); err != nil {
				log.BKEFormat(log.ERROR, fmt.Sprintf("Sync image %s failed: %s", imageAddress, err.Error()))
				return err
			}
		}
		return nil
	}

	// 多架构处理
	log.BKEFormat(log.INFO, fmt.Sprintf("Syncing multi-arch image %s to OCI layout", imageAddress))

	if !strings.Contains(imageAddress, cut) && reg.IsMultiArchManifests(srcTLSVerify, imageAddress) {
		if err := copyImageToOCI(imageAddress, ociLayoutDir, imageRef, "", srcTLSVerify); err == nil {
			return nil
		}
		log.BKEFormat(log.INFO, fmt.Sprintf("Direct multi-arch sync failed, trying per-arch sync"))
	}

	for _, ar := range arch {
		archImageAddress := imageAddress
		if strings.Contains(archImageAddress, cut) {
			archImageAddress = strings.ReplaceAll(archImageAddress, cut, fmt.Sprintf("-%s-", ar))
		} else {
			archImageAddress = imageAddress + "-" + ar
		}

		archImageRef := imageRef + "-" + ar
		log.BKEFormat(log.INFO, fmt.Sprintf("Syncing %s arch: %s as %s", ar, archImageAddress, archImageRef))

		if err := copyImageToOCI(archImageAddress, ociLayoutDir, archImageRef, ar, srcTLSVerify); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Sync image %s failed: %s", archImageAddress, err.Error()))
			return err
		}
	}

	return nil
}

func copyImageToOCI(sourceImage, ociLayoutDir, imageRef, arch string, srcTLSVerify bool) error {
	sourceRef := sourceImage
	if !strings.HasPrefix(sourceRef, "docker://") {
		sourceRef = "docker://" + sourceRef
	}

	targetRef := fmt.Sprintf("oci:%s:%s", ociLayoutDir, imageRef)

	isMultiArch := (arch == "")
	if isMultiArch {
		log.BKEFormat(log.DEBUG, fmt.Sprintf("Using CopyAllImages mode for multi-arch image %s", sourceImage))
	} else {
		log.BKEFormat(log.DEBUG, fmt.Sprintf("Using CopySystemImage mode for %s arch of %s", arch, sourceImage))
	}

	// 使用 registry.CopyRegistry 统一的镜像复制逻辑
	err := reg.CopyRegistry(reg.Options{
		Source:        sourceRef,
		Target:        targetRef,
		Arch:          arch,
		MultiArch:     isMultiArch,
		SrcTLSVerify:  srcTLSVerify,
		DestTLSVerify: false,
	})

	if err == nil {
		log.BKEFormat(log.DEBUG, fmt.Sprintf("Successfully copied %s to OCI layout with ref: %s", sourceImage, imageRef))
	}

	return err
}
