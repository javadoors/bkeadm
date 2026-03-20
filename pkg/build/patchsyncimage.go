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
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"

	"gopkg.openfuyao.cn/bkeadm/pkg/common"
	reg "gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/pkg/server"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func loadLocalRepository(imageFile string) error {
	return common.LoadLocalRepositoryFromFile(imageFile)
}

func SpecificSync(source, target string) {
	newTarget := normalizeTargetPath(target)
	if !validateSourceFiles(source) {
		return
	}

	cfg, err := loadManifestConfig(source)
	if err != nil {
		log.BKEFormat(log.ERROR, err.Error())
		return
	}

	mountPath := path.Join(source, "mount")
	ensureMountDirectoryExists(mountPath)

	// 检测补丁包格式
	format := DetectPatchFormat(source)
	log.BKEFormat(log.INFO, fmt.Sprintf("Detected patch format: %s", format))

	switch format {
	case "oci":
		log.BKEFormat(log.INFO, "Detected OCI format, using direct OCI-to-target conversion")

		if err = syncImagesFromOCI(source, cfg, newTarget); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to sync images from OCI: %s", err.Error()))
			return
		}

		log.BKEFormat(log.INFO, "OCI images synced successfully to target registry")

	case "registry":
		if err = prepareImageData(source); err != nil {
			log.BKEFormat(log.ERROR, err.Error())
			return
		}

		if err = loadAndStartRegistry(source); err != nil {
			log.BKEFormat(log.ERROR, err.Error())
			if err = server.RemoveImageRegistry(utils.PatchImageRegistryName); err != nil {
				log.BKEFormat(log.ERROR, err.Error())
			}
			return
		}

		if err = syncImages(cfg, newTarget); err != nil {
			log.BKEFormat(log.ERROR, err.Error())
		}

		if err = server.RemoveImageRegistry(utils.PatchImageRegistryName); err != nil {
			log.BKEFormat(log.ERROR, err.Error())
		}

	default:
		log.BKEFormat(log.ERROR, fmt.Sprintf("Unknown patch format: %s", format))
		return
	}
}

func normalizeTargetPath(target string) string {
	if !strings.HasSuffix(target, "/") {
		return target + "/"
	}
	return target
}

func validateSourceFiles(source string) bool {
	if !utils.IsDir(source) {
		log.BKEFormat(log.ERROR, fmt.Sprintf("The %s is not a directory", source))
		return false
	}

	manifestFile := path.Join(source, "manifests.yaml")
	if !utils.IsFile(manifestFile) {
		log.BKEFormat(log.ERROR, fmt.Sprintf("The manifests.yaml is not found in %s", source))
		return false
	}

	hasRegistry := utils.IsFile(path.Join(source, utils.ImageDataFile))
	hasOCI := utils.IsDir(path.Join(source, "volumes", "oci-layout"))

	if !hasRegistry && !hasOCI {
		log.BKEFormat(log.ERROR, "Neither image.tar.gz nor oci-layout directory found")
		return false
	}

	// 只在 registry 格式时检查 registry.image 文件
	if hasRegistry && !hasOCI {
		imageFile := path.Join(source, utils.ImageFile+"-"+runtime.GOARCH)
		if !utils.IsFile(imageFile) {
			log.BKEFormat(log.WARN, fmt.Sprintf("The %s is not found, may cause issues", utils.ImageFile+"-"+runtime.GOARCH))
		}
	}

	return true
}

func loadManifestConfig(source string) (*BuildConfig, error) {
	manifests := path.Join(source, "manifests.yaml")
	yamlFile, err := os.ReadFile(manifests)
	if err != nil {
		return nil, fmt.Errorf("failed to read %s, %s", manifests, err.Error())
	}

	cfg := &BuildConfig{}
	if err = yaml.Unmarshal(yamlFile, cfg); err != nil {
		return nil, fmt.Errorf("unable to serialize file, %s", err.Error())
	}
	return cfg, nil
}

func ensureMountDirectoryExists(mountPath string) {
	if !utils.Exists(mountPath) {
		if err := os.MkdirAll(mountPath, utils.DefaultDirPermission); err != nil {
			log.BKEFormat(log.ERROR, err.Error())
		}
	}
}

func prepareImageData(source string) error {
	imageDataDirectory := path.Join(source, utils.ImageDataDirectory)
	if !utils.Exists(imageDataDirectory) || utils.DirectoryIsEmpty(imageDataDirectory) {
		log.BKEFormat(log.INFO, "Decompressing the image package...")
		if err := utils.UnTar(path.Join(source, utils.ImageDataFile), imageDataDirectory); err != nil {
			return err
		}
		if !utils.Exists(imageDataDirectory) {
			log.BKEFormat(log.ERROR, fmt.Sprintf("%s, not found", imageDataDirectory))
			return fmt.Errorf("image data directory not found")
		}
	} else {
		log.BKEFormat(log.WARN, "If the image file already exists, skip decompressing the volumes/image.tar.gz file")
	}
	return nil
}

func loadAndStartRegistry(source string) error {
	imageFile := path.Join(source, utils.ImageFile+"-"+runtime.GOARCH)
	if err := loadLocalRepository(imageFile); err != nil {
		log.BKEFormat(log.ERROR, err.Error())
		return err
	}

	return server.StartImageRegistry(
		utils.PatchImageRegistryName,
		utils.DefaultLocalImageRegistry,
		"40448",
		path.Join(source, utils.ImageDataDirectory),
	)
}

func syncImages(cfg *BuildConfig, target string) error {
	for _, repos := range cfg.Repos {
		opts := reg.Options{
			MultiArch:     len(repos.Architecture) > 1,
			Arch:          getArchitecture(repos.Architecture),
			SrcTLSVerify:  false,
			DestTLSVerify: false,
		}

		for _, subImage := range repos.SubImages {
			if err := syncSubImage(subImage, opts, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func syncSubImage(subImage SubImage, opts reg.Options, target string) error {
	sourcePrefix := normalizeRegistryPath("0.0.0.0:40448/" + subImage.TargetRepo)
	targetPrefix := normalizeRegistryPath(target + subImage.TargetRepo)

	for _, image := range subImage.Images {
		if err := syncImageTags(image, sourcePrefix, targetPrefix, opts); err != nil {
			return err
		}
	}
	return nil
}

func syncImageTags(image Image, sourcePrefix, targetPrefix string, opts reg.Options) error {
	for _, tag := range image.Tag {
		tagName := strings.Split(tag, cut)[0]
		opts.Source = sourcePrefix + image.Name + ":" + tagName
		opts.Target = targetPrefix + image.Name + ":" + tagName

		if err := reg.CopyRegistry(opts); err != nil {
			log.BKEFormat(log.ERROR, err.Error())
			return err
		}
	}
	return nil
}

func getArchitecture(archs []string) string {
	if len(archs) == 1 {
		return archs[0]
	}
	return ""
}

func normalizeRegistryPath(path string) string {
	if !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return strings.ReplaceAll(path, "//", "/")
}

// 直接从OCI layout同步镜像到目标Registry
func syncImagesFromOCI(source string, cfg *BuildConfig, target string) error {
	ociDir := filepath.Join(source, "volumes", "oci-layout")

	absOciDir, err := filepath.Abs(ociDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for OCI directory: %v", err)
	}

	for _, repos := range cfg.Repos {
		opts := reg.Options{
			MultiArch:     len(repos.Architecture) > 1,
			Arch:          getArchitecture(repos.Architecture),
			SrcTLSVerify:  false,
			DestTLSVerify: false,
		}

		for _, subImage := range repos.SubImages {
			if err := syncSubImageFromOCI(absOciDir, subImage, opts, target); err != nil {
				return err
			}
		}
	}
	return nil
}

func syncSubImageFromOCI(ociDir string, subImage SubImage, opts reg.Options, target string) error {
	targetPrefix := normalizeRegistryPath(target + subImage.TargetRepo)

	for _, image := range subImage.Images {
		if err := syncImageTagsFromOCI(ociDir, image, targetPrefix, opts); err != nil {
			return err
		}
	}
	return nil
}

func copyImageFromOCI(ociSource, dockerTarget string, opts reg.Options) error {
	return reg.CopyRegistry(reg.Options{
		Source:        ociSource,
		Target:        dockerTarget,
		Arch:          opts.Arch,
		MultiArch:     opts.MultiArch,
		SrcTLSVerify:  false,
		DestTLSVerify: opts.DestTLSVerify,
	})
}

func syncImageTagsFromOCI(ociDir string, image Image, targetPrefix string, opts reg.Options) error {
	for _, tag := range image.Tag {
		tagName := strings.Split(tag, cut)[0]

		fullTargetRef := targetPrefix + image.Name + ":" + tagName
		ociImageRef := fullTargetRef
		if idx := strings.Index(ociImageRef, "/"); idx != -1 {
			ociImageRef = ociImageRef[idx+1:]
		}

		ociSource := fmt.Sprintf("oci:%s:%s", ociDir, ociImageRef)
		dockerTarget := fmt.Sprintf("docker://%s", fullTargetRef)

		log.BKEFormat(log.INFO, fmt.Sprintf("Syncing from OCI: %s -> %s", ociImageRef, fullTargetRef))

		if err := copyImageFromOCI(ociSource, dockerTarget, opts); err != nil {
			log.BKEFormat(log.ERROR, err.Error())
			log.BKEFormat(log.ERROR, "=== Error Details ===")
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to copy from: %s", ociSource))
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to copy to: %s", dockerTarget))
			log.BKEFormat(log.ERROR, "Possible causes:")
			log.BKEFormat(log.ERROR, "  1. OCI layout directory does not exist or is corrupted")
			log.BKEFormat(log.ERROR, "  2. Reference name in index.json does not match")
			log.BKEFormat(log.ERROR, "  3. Target registry is not accessible")
			log.BKEFormat(log.ERROR, "====================")
			return err
		}

		log.BKEFormat(log.INFO, fmt.Sprintf("Successfully synced: %s", ociImageRef))
	}
	return nil
}
