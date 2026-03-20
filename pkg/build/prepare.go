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
	"path"
	"strings"

	"gopkg.openfuyao.cn/cluster-api-provider-bke/common/cluster/validation"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

/*
├── bke
│ ├── manifests.yaml
│ └── volumes
│     ├── image.tar.gz
│     ├── registry.image-amd64
│     └── source.tar.gz
└── usr
    └── bin
        └── bke
*/

var (
	pwd                string
	packages           string
	bke                string
	bkeVolumes         string
	usrBin             string
	tmp                string
	tmpRegistry        string
	tmpPackages        string
	tmpPackagesCharts  string
	tmpPackagesFiles   string
	tmpPackagesPatches string
	cut                = "-*-"
)

func init() {
	var err error
	pwd, err = os.Getwd()
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to get current working directory: %s", err.Error()))
	}
	packages = path.Join(pwd, "packages")
	bke = path.Join(packages, "bke")
	bkeVolumes = path.Join(bke, "volumes")
	usrBin = path.Join(packages, "usr", "bin")
	tmp = path.Join(packages, "tmp")
	tmpRegistry = path.Join(tmp, "registry")
	tmpPackages = path.Join(tmp, "packages")
	tmpPackagesCharts = path.Join(tmp, "charts")
	tmpPackagesFiles = path.Join(tmpPackages, "files")
	tmpPackagesPatches = path.Join(tmpPackagesFiles, "patches")
}

func prepare() error {
	var err error
	pathLevel := []string{
		packages,
		bke,
		bkeVolumes,
		usrBin,
		tmp,
		tmpRegistry,
		tmpPackages,
		tmpPackagesFiles,
		tmpPackagesCharts,
		tmpPackagesPatches,
	}
	if err = os.RemoveAll(packages); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Unable to remove file %s %s", packages, err.Error()))
		return err
	}
	for _, pl := range pathLevel {
		if err = os.MkdirAll(pl, utils.DefaultDirPermission); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Unable to create file %s %s", pl, err.Error()))
			return err
		}
	}
	err = os.Chmod(packages, utils.ReadExecutePermission)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Permission change Failed %s 0555 %s", packages, err.Error()))
		return err
	}
	err = os.Chmod(path.Join(packages, "usr"), utils.DefaultDirPermission)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Permission change Failed %s 0755 %s", path.Join(packages, "usr"), err.Error()))
		return err
	}
	err = os.Chmod(usrBin, utils.DefaultDirPermission)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Permission change Failed %s 0755 %s", usrBin, err.Error()))
		return err
	}
	return nil
}

func checkIsChartDownload(cfg *BuildConfig) bool {
	for _, f := range cfg.Charts {
		if len(f.Address) == 0 {
			return false
		}
		if len(f.Files) != 0 {
			return true
		}
	}
	return false
}

func initRequiredDependencies() map[string]bool {
	return map[string]bool{
		"bke":                           false,
		"charts.tar.gz":                 false,
		"nfsshare.tar.gz":               false,
		"containerd":                    false,
		utils.DefaultLocalYumRegistry:   false,
		utils.DefaultLocalChartRegistry: false,
		utils.DefaultLocalNFSRegistry:   false,
		utils.DefaultLocalK3sRegistry:   false,
		utils.DefaultK3sPause:           false,
		utils.CniPluginPrefix:           false,
	}
}

func validateRpms(rpms []Rpm, requiredDepend map[string]bool) error {
	if requiredDepend == nil {
		return errors.New("required dependencies is nil")
	}
	for _, rpm := range rpms {
		if len(rpm.Address) == 0 {
			return errors.New("the sourceAddress is required")
		}
		if len(rpm.System) == 0 {
			return errors.New("the system is required")
		}
		if len(rpm.SystemVersion) == 0 {
			return errors.New("the systemVersion is required")
		}
		if len(rpm.SystemArchitecture) == 0 {
			return errors.New("the systemArchitecture is required")
		}
		if utils.ContainsString(rpm.Directory, "docker-ce") {
			requiredDepend["docker-ce"] = true
		}
	}
	return nil
}

func validateFiles(files []File, requiredDepend map[string]bool) ([]string, error) {
	if requiredDepend == nil {
		return nil, errors.New("requiredDepend is nil")
	}
	var containerdList []string
	for _, f := range files {
		if len(f.Address) == 0 {
			return nil, errors.New("The address is required. ")
		}
		tmpFiles := make([]string, 0)
		for _, subFile := range f.Files {
			tmpFiles = append(tmpFiles, subFile.FileName)
		}
		if utils.ContainsStringPrefix(tmpFiles, "charts.tar.gz") {
			if requiredDepend["charts.tar.gz"] {
				return nil, errors.New("There can only be one charts package. ")
			}
			requiredDepend["charts.tar.gz"] = true
		}
		if utils.ContainsStringPrefix(tmpFiles, "nfsshare.tar.gz") {
			if requiredDepend["nfsshare.tar.gz"] {
				return nil, errors.New("There can be only one nfsshare package. ")
			}
			requiredDepend["nfsshare.tar.gz"] = true
		}
		if utils.ContainsStringPrefix(tmpFiles, "bke") {
			requiredDepend["bke"] = true
		}
		if utils.ContainsStringPrefix(tmpFiles, utils.CniPluginPrefix) {
			requiredDepend[utils.CniPluginPrefix] = true
		}
		for _, f1 := range tmpFiles {
			err := validation.ValidateCustomExtra(map[string]string{"containerd": f1})
			if err != nil {
				continue
			}
			containerdList = append(containerdList, f1)
		}
	}
	return containerdList, nil
}

func checkImageDependencies(images []Image, requiredDepend map[string]bool) {
	if requiredDepend == nil {
		return
	}
	for _, image := range images {
		for _, tag := range image.Tag {
			img := fmt.Sprintf("%s:%s", image.Name, tag)
			switch img {
			case utils.DefaultLocalYumRegistry:
				requiredDepend[utils.DefaultLocalYumRegistry] = true
			case utils.DefaultLocalChartRegistry:
				requiredDepend[utils.DefaultLocalChartRegistry] = true
			case utils.DefaultLocalNFSRegistry:
				requiredDepend[utils.DefaultLocalNFSRegistry] = true
			case utils.DefaultLocalK3sRegistry:
				requiredDepend[utils.DefaultLocalK3sRegistry] = true
			case utils.DefaultK3sPause:
				requiredDepend[utils.DefaultK3sPause] = true
			default:
			}
		}
	}
}

func validateSubImages(subImages []SubImage, requiredDepend map[string]bool) error {
	for _, subRepo := range subImages {
		if len(subRepo.SourceRepo) == 0 {
			return errors.New("the mirror source address is required")
		}
		if len(subRepo.TargetRepo) == 0 {
			return errors.New("the target warehouse address is required")
		}
		if strings.Contains(subRepo.TargetRepo, "//") {
			return errors.New("the target warehouse address is not valid")
		}
		checkImageDependencies(subRepo.Images, requiredDepend)
	}
	return nil
}

func validateRepos(repos []Repo, requiredDepend map[string]bool) error {
	for _, cr := range repos {
		if !cr.NeedDownload {
			continue
		}
		if len(cr.Architecture) == 0 {
			return errors.New("the architecture parameters of the mirror are required")
		}
		if err := validateSubImages(cr.SubImages, requiredDepend); err != nil {
			return err
		}
	}
	return nil
}

func verifyConfigContent(cfg *BuildConfig) error {
	if len(cfg.Registry.ImageAddress) == 0 || len(cfg.Registry.Architecture) == 0 {
		return errors.New("the parameters registry.imageAddress and registry.architecture are required")
	}

	requiredDepend := initRequiredDependencies()
	if err := validateRpms(cfg.Rpms, requiredDepend); err != nil {
		return err
	}

	containerdList, err := validateFiles(cfg.Files, requiredDepend)
	if err != nil {
		return err
	}
	if len(containerdList) > 0 {
		requiredDepend["containerd"] = true
	}

	if !requiredDepend["charts.tar.gz"] {
		requiredDepend["charts.tar.gz"] = checkIsChartDownload(cfg)
	}

	if err = validateRepos(cfg.Repos, requiredDepend); err != nil {
		return err
	}

	for k, v := range requiredDepend {
		if !v {
			return fmt.Errorf("the build lacks required dependencies %s", k)
		}
	}
	return nil
}

// fileVersionAdaptation
// For multiple versions, the following notation is supported
// charts.tar.gz-v4.0 rename charts.tar.gz
// nfsshare.tar.gz-4.0 rename nfsshare.tar.gz
func fileVersionAdaptation() error {
	entries, err := os.ReadDir(tmpPackagesFiles)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "charts.tar.gz") {
			if entry.Name() == "charts.tar.gz" {
				continue
			}
			err = os.Rename(tmpPackagesFiles+"/"+entry.Name(), tmpPackagesFiles+"/charts.tar.gz")
			if err != nil {
				return err
			}
		}
		if strings.HasPrefix(entry.Name(), "nfsshare.tar.gz") {
			if entry.Name() == "nfsshare.tar.gz" {
				continue
			}
			err = os.Rename(tmpPackagesFiles+"/"+entry.Name(), tmpPackagesFiles+"/nfsshare.tar.gz")
			if err != nil {
				return err
			}
		}
		if strings.HasPrefix(entry.Name(), utils.RPMDataFile) {
			if entry.Name() == utils.RPMDataFile {
				continue
			}
			err = os.Rename(tmpPackagesFiles+"/"+entry.Name(), tmpPackagesFiles+"/"+utils.RPMDataFile)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
