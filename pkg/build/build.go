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

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/pkg/infrastructure"
	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type Options struct {
	root.Options
	File     string `json:"file"`
	Target   string `json:"target"`
	Strategy string `json:"strategy"`
	Arch     string `json:"Arch"`
}

// Build builds the bke package according to the configuration file.
func (o *Options) Build() {
	if !infrastructure.IsDocker() {
		log.BKEFormat(log.ERROR, "This build instruction only supports running in docker environment.")
		return
	}

	cfg, err := loadAndVerifyBuildConfig(o.File)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to load and verify build config: %s", err.Error()))
		return
	}

	if err := prepareBuildWorkspace(); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to prepare build workspace: %s", err.Error()))
		return
	}

	version, err := o.collectDependenciesAndImages(cfg)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to collect dependencies and images: %s", err.Error()))
		return
	}

	if err := o.createFinalPackage(cfg, version); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to create final package: %s", err.Error()))
		return
	}

	log.BKEFormat("step.8", fmt.Sprintf("Packaging complete %s", o.Target))
}

func loadAndVerifyBuildConfig(file string) (*BuildConfig, error) {
	log.BKEFormat("step.1", "Configuration file check")
	cfg := &BuildConfig{}
	yamlFile, err := os.ReadFile(file)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to read the file, %s", err.Error()))
		return nil, err
	}
	if err = yaml.Unmarshal(yamlFile, cfg); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Unable to serialize file, %s", err.Error()))
		return nil, err
	}
	err = verifyConfigContent(cfg)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Configuration verification fails %s", err.Error()))
		return nil, err
	}
	return cfg, nil
}

func prepareBuildWorkspace() error {
	log.BKEFormat("step.2", "Creates a workspace in the current directory")
	if err := prepare(); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to prepare workspace: %s", err.Error()))
		return err
	}
	return nil
}

func (o *Options) collectDependenciesAndImages(cfg *BuildConfig) (string, error) {
	var version string
	var errNumber uint64
	stopChan := make(chan struct{})
	defer closeChanStruct(stopChan)
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		var err error
		version, err = collectRpmsAndBinary(cfg, stopChan, &errNumber)
		if err != nil {
			closeChanStruct(stopChan)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := collectRegistryImages(cfg, stopChan, &errNumber); err != nil {
			closeChanStruct(stopChan)
		}
	}()

	wg.Wait()
	if errNumber > 0 {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures %d", errNumber))
		return "", fmt.Errorf("build failures: %d", errNumber)
	}
	if len(version) == 0 {
		log.BKEFormat(log.ERROR, "Failed to get bke version number, please check")
		return "", fmt.Errorf("failed to get bke version")
	}
	return version, nil
}

func collectRpmsAndBinary(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) (string, error) {
	log.BKEFormat("step.3", "Collect host dependency packages and package files")
	if err := buildRpms(cfg, stopChan); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures %s when collecting host dependency packages "+
			"and package files", err.Error()))
		*errNumber++
		return "", err
	}

	log.BKEFormat("step.4", "Collect the bke binary file")
	version, err := buildBkeBinary()
	if err != nil || len(version) == 0 {
		log.BKEFormat(log.ERROR, "Collect the bke binary file failed, get bke version number failed")
		if err != nil {
			log.BKEFormat(log.ERROR, err.Error())
		}
		*errNumber++
		return "", err
	}
	return version, nil
}

func collectRegistryImages(cfg *BuildConfig, stopChan chan struct{}, errNumber *uint64) error {
	log.BKEFormat("step.5", "Collect the required image files")
	if err := buildRegistry(cfg.Registry.ImageAddress, cfg.Registry.Architecture); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures %s when collecting image files", err.Error()))
		*errNumber++
		return err
	}

	log.BKEFormat("step.6", "Collect images from the source repository to the target repository")
	if err := syncRepo(cfg, stopChan); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Build failures %s when collecting images from the "+
			"source repository to the target repository", err.Error()))
		*errNumber++
		return err
	}
	return nil
}

func (o *Options) createFinalPackage(cfg *BuildConfig, version string) error {
	log.BKEFormat("step.7", "Build the bke package, please wait for the larger package...")
	if len(o.Target) == 0 {
		fileInfo, err := os.Stat(o.File)
		if err != nil {
			log.BKEFormat(log.ERROR, err.Error())
			return err
		}
		o.Target = path.Join(pwd, fmt.Sprintf("bke-%s-%s-%s-%s.tar.gz", version,
			strings.TrimSuffix(fileInfo.Name(), ".yaml"),
			strings.Join(cfg.Registry.Architecture, "-"), time.Now().Format("20060102150405")))
	}
	if err := compressedPackage(cfg, o.Target); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to compress the package error is%s", err.Error()))
		return err
	}
	return nil
}

func compressedPackage(cfg *BuildConfig, target string) error {
	err := writeManifestsFile(cfg, path.Join(bke, "manifests.yaml"))
	if err != nil {
		return err
	}
	return finalizeAndCompress(target)
}

// writeManifestsFile marshals the config and writes it to the specified path
func writeManifestsFile(cfg *BuildConfig, manifestPath string) error {
	b, err := yaml.Marshal(cfg)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Description Failed to parse the configuration file %v", err))
		return err
	}
	err = os.WriteFile(manifestPath, b, utils.DefaultFilePermission)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to write the configuration file. Procedure %v", err))
		return err
	}
	return nil
}

// finalizeAndCompress removes tmp directory, resolves target path, and compresses the package
func finalizeAndCompress(target string) error {
	if err := os.RemoveAll(tmp); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Unable to delete temporary file %s", err.Error()))
		return err
	}
	if !strings.Contains(target, "/") {
		target = filepath.Join(pwd, target)
	}
	if err := global.TaeGZWithoutChangeFile(packages, target); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to compress the package to %s", target))
		return err
	}
	return nil
}
