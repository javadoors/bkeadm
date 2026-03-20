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
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"

	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func (o *Options) Patch() {
	if err := o.checkEnvironment(); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to check environment: %s", err.Error()))
		return
	}

	cfg, err := o.loadAndValidateConfig()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to load and validate config: %s", err.Error()))
		return
	}

	if err := o.prepareWorkspace(); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to prepare workspace: %s", err.Error()))
		return
	}

	if err := o.collectFilesAndImages(cfg); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to collect files and images: %s", err.Error()))
		return
	}

	if err := o.createPatchPackage(cfg); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to create patch package: %s", err.Error()))
		return
	}

	log.BKEFormat("step.7", fmt.Sprintf("Packaging complete %s", o.Target))
}

func (o *Options) checkEnvironment() error {
	if o.Strategy != "oci" && !infrastructure.IsDocker() {
		log.BKEFormat(log.ERROR, "This build instruction only supports running in docker environment.")
		return fmt.Errorf("docker environment required")
	}
	return nil
}

func (o *Options) loadAndValidateConfig() (*BuildConfig, error) {
	log.BKEFormat("step.1", "Configuration file check")
	cfg := &BuildConfig{}
	yamlFile, err := os.ReadFile(o.File)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to read the file, %s", err.Error()))
		return nil, err
	}
	if err = yaml.Unmarshal(yamlFile, cfg); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Unable to serialize file, %s", err.Error()))
		return nil, err
	}
	if (o.Strategy == "registry") && (len(cfg.Registry.ImageAddress) == 0 || len(cfg.Registry.Architecture) == 0) {
		log.BKEFormat(log.ERROR, "The parameters registry.imageAddress and registry.architecture are required. ")
		return nil, fmt.Errorf("missing required parameters")
	}
	return cfg, nil
}

func (o *Options) prepareWorkspace() error {
	log.BKEFormat("step.2", "Creates a workspace in the current directory")
	if err := prepare(); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to prepare workspace: %s", err.Error()))
		return err
	}
	if err := os.RemoveAll(path.Join(packages, "usr")); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to remove usr directory: %v", err))
	}
	return nil
}

func (o *Options) collectFilesAndImages(cfg *BuildConfig) error {
	var errNumber uint64
	stopChan := make(chan struct{})
	defer closeChanStruct(stopChan)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := o.collectHostFiles(cfg, stopChan, &errNumber); err != nil {
			closeChanStruct(stopChan)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := o.collectPatchFiles(cfg, stopChan, &errNumber); err != nil {
			closeChanStruct(stopChan)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := o.collectChartFiles(cfg, stopChan, &errNumber); err != nil {
			closeChanStruct(stopChan)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := o.collectImages(cfg, stopChan, &errNumber); err != nil {
			closeChanStruct(stopChan)
		}
	}()

	wg.Wait()
	if errNumber > 0 {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures %d", errNumber))
		return fmt.Errorf("build failures: %d", errNumber)
	}
	return nil
}

func (o *Options) collectHostFiles(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
	log.BKEFormat("step.3.1", "Collect host dependency packages and package files")
	err := buildFiles(cfg.Files, tmpPackagesFiles, stopChan)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build package %s", err.Error()))
		*errNumber++
		return err
	}
	err = transferFile(tmpPackagesFiles, bkeVolumes)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to transfer file %s", err.Error()))
		*errNumber++
		return err
	}
	return nil
}

func (o *Options) collectPatchFiles(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
	log.BKEFormat("step.3.2", "Collect patch files")
	err := buildFiles(cfg.Patches, tmpPackagesPatches, stopChan)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build package %s", err.Error()))
		*errNumber++
		return err
	}
	err = transferFile(tmpPackagesPatches, path.Join(bkeVolumes, "patches"))
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to transfer file %s", err.Error()))
		*errNumber++
		return err
	}
	return nil
}

func (o *Options) collectChartFiles(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
	log.BKEFormat("step.3.3", "Collect chart files")
	err := buildFiles(cfg.Charts, tmpPackagesCharts, stopChan)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build package %s", err.Error()))
		*errNumber++
		return err
	}
	err = transferFile(tmpPackagesCharts, path.Join(bkeVolumes, "charts"))
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to transfer file %s", err.Error()))
		*errNumber++
		return err
	}
	return nil
}

func (o *Options) collectImages(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
	log.BKEFormat("step.4", "Collect the required image files")
	// OCI策略不需要buildRegistry
	if o.Strategy != "oci" {
		if err := buildRegistry(cfg.Registry.ImageAddress, cfg.Registry.Architecture); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures %s", err.Error()))
			*errNumber++
			return err
		}
	}

	log.BKEFormat("step.5", "Collect images from the source repository to the target repository")
	return o.syncImagesByStrategy(cfg, stopChan, errNumber)
}

func (o *Options) syncImagesByStrategy(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
	switch o.Strategy {
	case "registry":
		if err := syncRepo(cfg, stopChan); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures %s", err.Error()))
			*errNumber++
			return err
		}
	case "oci":
		if err := syncRepoOCI(cfg, stopChan); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures (OCI strategy) %s", err.Error()))
			*errNumber++
			return err
		}
	default:
		log.BKEFormat(log.ERROR, fmt.Sprintf("Unknown strategy: %s, use 'registry' or 'oci'", o.Strategy))
		*errNumber++
		return fmt.Errorf("unknown strategy: %s", o.Strategy)
	}
	return nil
}

func (o *Options) createPatchPackage(cfg *BuildConfig) error {
	log.BKEFormat("step.6", "Build the bke package, please wait for the larger package...")
	fileInfo, err := os.Stat(o.File)
	if err != nil {
		log.BKEFormat(log.ERROR, err.Error())
		return err
	}
	fileName := strings.TrimSuffix(fileInfo.Name(), ".yaml")
	if len(o.Target) == 0 {
		o.Target = fmt.Sprintf("bke-patch-%s-%s-%s.tar.gz", fileName,
			strings.Join(cfg.Registry.Architecture, "-"), time.Now().Format("20060102150405"))
	}
	return compressedPatch(cfg, o.Target)
}

// transferFile 将指定源目录下的文件和目录结构转移到目标目录下，保留目录结构
func transferFile(sourceDir, targetDir string) error {
	// 确保目标目录存在
	if err := os.MkdirAll(targetDir, utils.DefaultDirPermission); err != nil {
		return fmt.Errorf("failed to create target directory %s: %w", targetDir, err)
	}

	// 使用 filepath.Walk 递归遍历源目录，保留目录结构
	return filepath.Walk(sourceDir, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算相对于源目录的路径
		relPath, err := filepath.Rel(sourceDir, srcPath)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// 跳过源目录本身
		if relPath == "." {
			return nil
		}

		// 构建目标路径
		dstPath := filepath.Join(targetDir, relPath)

		if info.IsDir() {
			// 如果是目录，创建目标目录
			if err := os.MkdirAll(dstPath, utils.DefaultDirPermission); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", dstPath, err)
			}
		} else {
			// 如果是文件，确保目标目录存在，然后移动文件
			if err := os.MkdirAll(filepath.Dir(dstPath), utils.DefaultDirPermission); err != nil {
				return fmt.Errorf("failed to create parent directory for %s: %w", dstPath, err)
			}
			// 如果目标文件已存在，先删除
			if utils.Exists(dstPath) {
				if err := os.Remove(dstPath); err != nil {
					return fmt.Errorf("failed to remove existing file %s: %w", dstPath, err)
				}
			}
			// 移动文件
			if err := os.Rename(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to move file from %s to %s: %w", srcPath, dstPath, err)
			}
		}

		return nil
	})
}

func compressedPatch(cfg *BuildConfig, target string) error {
	bkePatch := strings.TrimSuffix(target, ".tar.gz")
	err := os.Rename(bke, path.Join(packages, bkePatch))
	if err != nil {
		log.BKEFormat(log.ERROR, err.Error())
		return err
	}

	err = writeManifestsFile(cfg, path.Join(packages, bkePatch, "manifests.yaml"))
	if err != nil {
		return err
	}

	err = finalizeAndCompress(target)
	if err != nil {
		return err
	}

	if err = os.RemoveAll(packages); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to remove the package %s", packages))
		return err
	}
	return nil
}
