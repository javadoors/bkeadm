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
	"os"
	"path/filepath"
	"strings"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

// DetectPatchFormat 检测补丁包格式（oci、registry 或 unknown）
func DetectPatchFormat(source string) string {
	ociLayoutDir := filepath.Join(source, "volumes", "oci-layout")
	if isValidOCILayout(ociLayoutDir) {
		return "oci"
	}

	registryDir := filepath.Join(source, "volumes", "registry", "docker")
	if utils.IsDir(registryDir) {
		return "registry"
	}

	registryTarGz := filepath.Join(source, utils.ImageDataFile)
	if utils.IsFile(registryTarGz) {
		return "registry"
	}

	return "unknown"
}

// 验证OCI layout的完整性
func isValidOCILayout(ociLayoutDir string) bool {
	ociLayoutFile := filepath.Join(ociLayoutDir, "oci-layout")
	if !utils.IsFile(ociLayoutFile) {
		return false
	}

	indexFile := filepath.Join(ociLayoutDir, "index.json")
	if !utils.IsFile(indexFile) {
		return false
	}

	blobsDir := filepath.Join(ociLayoutDir, "blobs", "sha256")
	if !utils.IsDir(blobsDir) {
		return false
	}

	content, err := os.ReadFile(ociLayoutFile)
	if err != nil {
		return false
	}

	if !strings.Contains(string(content), "imageLayoutVersion") {
		log.BKEFormat(log.WARN, "oci-layout file exists but missing imageLayoutVersion")
		return false
	}

	return true
}
