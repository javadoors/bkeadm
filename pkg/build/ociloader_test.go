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
	"testing"

	"gopkg.openfuyao.cn/bkeadm/utils"
)

// TestIsValidOCILayout 测试 OCI layout 验证
func TestIsValidOCILayout(t *testing.T) {
	t.Run("valid", testValidOCILayout)
	t.Run("missing files", testMissingOCIFiles)
	t.Run("invalid json", testInvalidOCIJSON)
}

func testValidOCILayout(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "valid")
	if err := os.MkdirAll(filepath.Join(dir, "blobs", "sha256"), utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	ociLayoutContent := []byte(`{"imageLayoutVersion":"1.0.0"}`)
	if err := os.WriteFile(filepath.Join(dir, "oci-layout"), ociLayoutContent, utils.DefaultFilePermission); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	indexContent := []byte(`{"schemaVersion":2,"manifests":[]}`)
	if err := os.WriteFile(filepath.Join(dir, "index.json"), indexContent, utils.DefaultFilePermission); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if !isValidOCILayout(dir) {
		t.Error("valid OCI layout should return true")
	}
}

func testMissingOCIFiles(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "invalid")
	if err := os.MkdirAll(dir, utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	if isValidOCILayout(dir) {
		t.Error("incomplete OCI layout should return false")
	}
}

func testInvalidOCIJSON(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "badjson")
	if err := os.MkdirAll(filepath.Join(dir, "blobs", "sha256"), utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "oci-layout"), []byte("invalid"), utils.DefaultFilePermission); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	indexContent := []byte(`{"schemaVersion":2,"manifests":[]}`)
	if err := os.WriteFile(filepath.Join(dir, "index.json"), indexContent, utils.DefaultFilePermission); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if isValidOCILayout(dir) {
		t.Error("invalid JSON should return false")
	}
}

// TestDetectPatchFormat 测试补丁格式检测
func TestDetectPatchFormat(t *testing.T) {
	tmpDir := t.TempDir()

	// OCI 格式
	t.Run("oci", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "oci")
		ociDir := filepath.Join(dir, "volumes", "oci-layout")
		if err := os.MkdirAll(filepath.Join(ociDir, "blobs", "sha256"), utils.DefaultDirPermission); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}
		ociLayoutContent := []byte(`{"imageLayoutVersion":"1.0.0"}`)
		if err := os.WriteFile(filepath.Join(ociDir, "oci-layout"), ociLayoutContent, utils.DefaultFilePermission); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}
		indexContent := []byte(`{"schemaVersion":2,"manifests":[]}`)
		if err := os.WriteFile(filepath.Join(ociDir, "index.json"), indexContent, utils.DefaultFilePermission); err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		if got := DetectPatchFormat(dir); got != "oci" {
			t.Errorf("DetectPatchFormat() = %q, want %q", got, "oci")
		}
	})

	// Registry 格式
	t.Run("registry", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "registry")
		if err := os.MkdirAll(filepath.Join(dir, "volumes", "registry", "docker"), utils.DefaultDirPermission); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		if got := DetectPatchFormat(dir); got != "registry" {
			t.Errorf("DetectPatchFormat() = %q, want %q", got, "registry")
		}
	})

	// 未知格式
	t.Run("unknown", func(t *testing.T) {
		unknownDir := t.TempDir()
		// 不创建任何子目录或文件，完全空的目录
		if got := DetectPatchFormat(unknownDir); got != "unknown" {
			t.Errorf("DetectPatchFormat() = %q, want %q (dir: %s)", got, "unknown", unknownDir)
		}
	})
}
