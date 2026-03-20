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
	"path/filepath"
	"strings"

	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func downloadUrlContent(files []File, storagePath string) error {
	for _, f := range files {
		prefixUrl := f.Address
		if !strings.HasSuffix(prefixUrl, "/") {
			prefixUrl += "/"
		}
		for _, f1 := range f.Files {
			url := prefixUrl + f1.FileName
			targetFile := path.Join(storagePath, f1.FileName)
			if f1.FileAlias != "" {
				targetFile = path.Join(targetFile, f1.FileAlias)
			}
			err := utils.DownloadSignalFile(url, targetFile)
			if err != nil {
				log.BKEFormat(log.ERROR, fmt.Sprintf("download file %s error %v", f1, err))
				return err
			}
		}
	}
	return nil
}

func downloadFile(cfg *BuildConfig, stopChan <-chan struct{}) error {
	// Collect deployment packages
	err := buildFiles(cfg.Files, tmpPackagesFiles, stopChan)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build package %s", err.Error()))
		return err
	}
	err = buildFiles(cfg.Charts, tmpPackagesCharts, stopChan)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build chart package %s", err.Error()))
		return err
	}
	err = downloadUrlContent(cfg.Patches, tmpPackagesPatches)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to build patch package %s", err.Error()))
		return err
	}
	return nil
}

func buildRpms(cfg *BuildConfig, stopChan <-chan struct{}) error {
	err := downloadFile(cfg, stopChan)
	if err != nil {
		return err
	}
	err = fileVersionAdaptation()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to rewrite the file name %s", err.Error()))
		return err
	}
	// 重新调整charts.tar.gz
	err = buildFileChart()
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to reconstruct charts.tar.gz %s", err.Error()))
		return err
	}
	// tar zxvf rpm.tar.gz
	if err = buildFileRpm(); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to tar zxvf rpm.tar.gz %s", err.Error()))
		return err
	}
	// Build the rmp source package
	// Duo to the operating system, the yum source can be built only under the corresponding system
	// The default build server is centos7.x
	for _, rpm := range cfg.Rpms {
		select {
		case <-stopChan:
			log.BKEFormat(log.WARN, "build rpm be externally terminated. ")
			return errors.New("build rpm be externally terminated. ")
		default:
		}
		url := rpm.Address
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		err := syncPackage(url, rpm.System, rpm.SystemVersion, rpm.SystemArchitecture, rpm.Directory)
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to collect package %s", err.Error()))
			return err
		}
	}

	if err = global.TarGZ(tmpPackages, fmt.Sprintf("%s/%s", bke, utils.SourceDataFile)); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("tar %s error %s",
			fmt.Sprintf("%s/%s", bke, utils.SourceDataFile), err.Error()))
		return err
	}
	return nil
}

func buildFiles(files []File, storagePath string, stopChan <-chan struct{}) error {
	for _, f := range files {
		select {
		case <-stopChan:
			log.BKEFormat(log.WARN, "build files be externally terminated. ")
			return errors.New("build files be externally terminated. ")
		default:
		}
		url := f.Address
		if !strings.HasSuffix(url, "/") {
			url += "/"
		}
		for _, f1 := range f.Files {
			downloadFile := url + f1.FileName
			targetFile := path.Join(storagePath, f1.FileName)
			if f1.FileAlias != "" {
				targetFile = path.Join(storagePath, f1.FileAlias)
			}
			log.BKEFormat(log.INFO, fmt.Sprintf("Collecting file packages %s to %s", downloadFile, targetFile))
			err := utils.DownloadFile(downloadFile, targetFile)
			if err != nil {
				log.BKEFormat(log.ERROR, fmt.Sprintf("download file %s error %v", downloadFile, err))
				return err
			}
		}
	}
	return nil
}

func syncPackage(url string, systems, versions, architectures, directory []string) error {
	for _, s := range systems {
		for _, v := range versions {
			if err := processSystemVersion(url, s, v, architectures, directory); err != nil {
				return err
			}
		}
	}
	return nil
}

func processSystemVersion(url, system, version string, architectures, directory []string) error {
	for _, ar := range architectures {
		if err := processArchitecture(url, system, version, ar, directory); err != nil {
			return err
		}
	}
	return nil
}

func processArchitecture(url, system, version, arch string, directories []string) error {
	for _, d := range directories {
		if err := downloadPackageDirectory(url, system, version, arch, d); err != nil {
			return err
		}
	}
	cmd := fmt.Sprintf("createrepo %s", path.Join(tmpPackages, system, version, arch))
	log.BKEFormat(log.INFO, fmt.Sprintf("Execute the create repo instruction, %s", cmd))
	output, err := global.Command.ExecuteCommandWithOutput("sh", "-c", cmd)
	if err != nil {
		log.BKEFormat(log.ERROR, output)
		return err
	}
	return nil
}

func downloadPackageDirectory(url, system, version, arch, directory string) error {
	downloadUrl := fmt.Sprintf("%s%s/%s/%s/%s/", url, system, version, arch, directory)
	downloadDirectory := path.Join(tmpPackages, system, version, arch, directory)
	if err := os.MkdirAll(downloadDirectory, utils.DefaultDirPermission); err != nil {
		return err
	}
	log.BKEFormat(log.INFO, fmt.Sprintf("Collect all installation packages under the url %s", downloadUrl))
	return utils.DownloadAllFiles(downloadUrl, downloadDirectory)
}

// buildBKEBinary Collect the bke binary file
// The bke naming rules are as follows, example
/*
	bke
	bke_amd64
*/
func buildBkeBinary() (string, error) {
	bkeBinaryList, err := findBkeBinaries()
	if err != nil {
		return "", err
	}
	if len(bkeBinaryList) == 0 {
		log.BKEFormat(log.ERROR, "The files list must contain bke")
		return "", errors.New("the files list must contain bke binary file")
	}

	if len(bkeBinaryList) == 1 {
		return installSingleBkeBinary(bkeBinaryList[0])
	}
	return installMultipleBkeBinaries(bkeBinaryList)
}

func findBkeBinaries() ([]string, error) {
	files, err := os.ReadDir(tmpPackagesFiles)
	if err != nil {
		return nil, err
	}
	var bkeBinaryList []string
	for _, f := range files {
		if f.Name() == "bke" || strings.HasPrefix(f.Name(), "bkeadm_") || strings.HasPrefix(f.Name(), "bke_") {
			bkeBinaryList = append(bkeBinaryList, f.Name())
		}
	}
	return bkeBinaryList, nil
}

func installSingleBkeBinary(bkeName string) (string, error) {
	sourceBKE := fmt.Sprintf("%s/%s", tmpPackagesFiles, bkeName)
	targetBKE := fmt.Sprintf("%s/bke", usrBin)
	if err := utils.CopyFile(sourceBKE, targetBKE); err != nil {
		return "", err
	}
	if err := os.Chmod(targetBKE, utils.ExecutableFilePermission); err != nil {
		log.BKEFormat(log.ERROR, "Failed to modify the file permission")
		return "", err
	}
	version, err := global.Command.ExecuteCommandWithOutput("sh", "-c", fmt.Sprintf("%s version only", targetBKE))
	if err != nil {
		return "", err
	}
	return version, nil
}

func installMultipleBkeBinaries(bkeBinaryList []string) (string, error) {
	var version string
	for _, bkeName := range bkeBinaryList {
		sourceBKE := fmt.Sprintf("%s/%s", tmpPackagesFiles, bkeName)
		targetBKE := fmt.Sprintf("%s/%s", usrBin, bkeName)
		if err := utils.CopyFile(sourceBKE, targetBKE); err != nil {
			return "", err
		}
		if err := os.Chmod(targetBKE, utils.ExecutableFilePermission); err != nil {
			log.BKEFormat(log.ERROR, "Failed to modify the file permission")
			return "", err
		}
		if len(version) == 0 {
			version, _ = global.Command.ExecuteCommandWithOutput("sh", "-c", fmt.Sprintf("%s version only", targetBKE))
		}
	}
	log.BKEFormat(log.INFO, fmt.Sprintf("The bke binary file version is %s", version))
	return version, nil
}

func moveFilesFromSubfolder(parentPath, subfolderPath string) error {
	entries, err := os.ReadDir(subfolderPath)
	if err != nil {
		return fmt.Errorf("moveFilesFromSubfolder read %s fail: %w", subfolderPath, err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			oldPath := filepath.Join(subfolderPath, entry.Name())
			newPath := filepath.Join(parentPath, entry.Name())

			if _, err = os.Stat(newPath); err == nil {
				continue
			}

			if err = utils.CopyFile(oldPath, newPath); err != nil {
				return fmt.Errorf("moveFilesFromSubfolder copyFile %s to %s fail: %w", oldPath, newPath, err)
			}

			if err = os.Remove(oldPath); err != nil {
				return fmt.Errorf("moveFilesFromSubfolder rm %s fail: %w", oldPath, err)
			}
		}
	}
	return nil
}

func removeDir(dirPath string) error {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		fullPath := filepath.Join(dirPath, entry.Name())

		if entry.IsDir() {
			err = removeDir(fullPath)
			if err != nil {
				return err
			}
		} else {
			err = os.Remove(fullPath)
			if err != nil {
				return err
			}
		}
	}

	return os.Remove(dirPath)
}

func processFolder(rootPath string) error {
	entries, err := os.ReadDir(rootPath)
	if err != nil {
		return fmt.Errorf("processFolder read dir %s fail: %w", rootPath, err)
	}

	for _, entry := range entries {
		fullPath := filepath.Join(rootPath, entry.Name())
		if entry.IsDir() {
			if err = moveFilesFromSubfolder(rootPath, fullPath); err != nil {
				return fmt.Errorf("processFolder move %s to %s fail: %w", fullPath, rootPath, err)
			}

			if err = removeDir(fullPath); err != nil {
				return fmt.Errorf("processFolder rm %s fail: %w", fullPath, err)
			}
		}
	}

	return nil
}

func rePackageChart(srcDir, subDir, target string) error {
	err := processFolder(path.Join(srcDir, subDir))
	if err != nil {
		return fmt.Errorf("rePackageChart process folder %s/%s fail: %w", srcDir, subDir, err)
	}
	if err = global.TarGZWithDir(srcDir, subDir, target); err != nil {
		return fmt.Errorf("rePackageChart compress %s/%s to %s fail: %w", srcDir, subDir, target, err)
	}

	return nil
}

func buildFileChart() error {
	entries, err := os.ReadDir(tmpPackagesCharts)
	if err != nil {
		return fmt.Errorf("buildFileChart read %s fail: %w", tmpPackagesCharts, err)
	}
	if len(entries) == 0 {
		log.BKEFormat(log.WARN, fmt.Sprintf("%s has no chart, not need extra operation", tmpPackagesCharts))
		return nil
	}
	target := filepath.Join(tmpPackagesFiles, utils.ChartFile)
	entries, err = os.ReadDir(tmpPackagesFiles)
	if err != nil {
		return fmt.Errorf("buildFileChart read %s fail: %w", tmpPackagesFiles, err)
	}
	for _, entry := range entries {
		if entry.Name() == utils.ChartFile {
			// 解压文件charts.tar.gz到tmpPackagesCharts
			if err = global.UnTarGZ(target, tmpPackagesCharts); err != nil {
				return fmt.Errorf("buildFileChart untar %s to %s fail: %w", target, tmpPackagesCharts, err)
			}
			err = os.Remove(target)
			if err != nil {
				return fmt.Errorf("buildFileChart rm %s fail: %w", target, err)
			}
			break
		}
	}
	// 重新打包charts.tar.gz
	if err = rePackageChart(tmp, "charts", target); err != nil {
		return fmt.Errorf("buildFileChart re tar to %s fail: %w", target, err)
	}
	return nil
}

func buildFileRpm() error {
	entries, err := os.ReadDir(tmpPackagesFiles)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.Name() == utils.RPMDataFile {
			// 解压文件rpm.tar.gz到tmpPackages
			if err = global.UnTarGZ(tmpPackagesFiles+"/"+utils.RPMDataFile, tmpPackages); err != nil {
				log.BKEFormat(log.ERROR, fmt.Sprintf("tar -zxf %s %s error %s",
					tmpPackagesFiles+"/"+utils.RPMDataFile, tmpPackages, err.Error()))
				return err
			}
			err = os.Remove(tmpPackagesFiles + "/" + utils.RPMDataFile)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return nil
}
