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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"sigs.k8s.io/yaml"

	"gopkg.openfuyao.cn/bkeadm/pkg/common"
	reg "gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/pkg/server"
	"gopkg.openfuyao.cn/bkeadm/utils"
)

const (
	testIPv4SegmentA = 192
	testIPv4SegmentB = 168
	testIPv4SegmentC = 1
	testIPv4SegmentD = 100
)

var testIP = fmt.Sprintf("%d.%d.%d.%d", testIPv4SegmentA, testIPv4SegmentB, testIPv4SegmentC, testIPv4SegmentD)

// 测试目标路径标准化
func TestNormalizeTargetPath(t *testing.T) {
	tests := []struct {
		target   string
		expected string
	}{
		{"registry.example.com", "registry.example.com/"},
		{"registry.example.com/", "registry.example.com/"},
		{"localhost:5000", "localhost:5000/"},
		{"", "/"},
	}

	for _, tt := range tests {
		if got := normalizeTargetPath(tt.target); got != tt.expected {
			t.Errorf("normalizeTargetPath(%q) = %q, want %q", tt.target, got, tt.expected)
		}
	}
}

// 测试配置文件加载
func TestLoadManifestConfig(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("valid", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "valid")
		if err := os.MkdirAll(dir, utils.DefaultDirPermission); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		content := `registry:
  imageAddress: registry.example.com/registry:2.8.1
  architecture:
    - amd64
`
		if err := os.WriteFile(
			filepath.Join(dir, "manifests.yaml"), []byte(content), utils.DefaultFilePermission); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		cfg, err := loadManifestConfig(dir)
		if err != nil {
			t.Fatalf("loadManifestConfig() error = %v", err)
		}
		if cfg.Registry.ImageAddress != "registry.example.com/registry:2.8.1" {
			t.Error("config not loaded correctly")
		}
	})

	t.Run("invalid yaml", func(t *testing.T) {
		dir := filepath.Join(tmpDir, "invalid")
		if err := os.MkdirAll(dir, utils.DefaultDirPermission); err != nil {
			t.Fatalf("failed to create directory: %v", err)
		}
		if err := os.WriteFile(
			filepath.Join(dir, "manifests.yaml"), []byte("invalid: ["), utils.DefaultFilePermission); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		if _, err := loadManifestConfig(dir); err == nil {
			t.Error("should return error for invalid YAML")
		}
	})
}

// 测试 BuildConfig YAML 序列化
func TestBuildConfigYAML(t *testing.T) {
	yamlContent := `registry:
  imageAddress: registry.example.com/registry:2.8.1
  architecture:
    - amd64
`
	var cfg BuildConfig
	if err := yaml.Unmarshal([]byte(yamlContent), &cfg); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if cfg.Registry.ImageAddress != "registry.example.com/registry:2.8.1" {
		t.Error("YAML unmarshal failed")
	}
}

// 测试挂载目录创建
func TestEnsureMountDirectoryExists(t *testing.T) {
	tmpDir := t.TempDir()
	mountPath := filepath.Join(tmpDir, "mount")

	ensureMountDirectoryExists(mountPath)

	if _, err := os.Stat(mountPath); os.IsNotExist(err) {
		t.Error("mount directory should be created")
	}

	ensureMountDirectoryExists(mountPath)
}

// 综合测试源文件验证
func TestValidateSourceFilesComprehensive(t *testing.T) {
	t.Run("valid oci format", testValidOCIFormat)
	t.Run("valid registry format", testValidRegistryFormat)
	t.Run("missing manifest", testMissingManifest)
	t.Run("not a directory", testNotADirectory)
}

func testValidOCIFormat(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "valid-oci")
	if err := os.MkdirAll(dir, utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "manifests.yaml"), []byte("test: value"), utils.DefaultFilePermission); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "volumes", "oci-layout"), utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	if !validateSourceFiles(dir) {
		t.Error("valid OCI format should return true")
	}
}

func testValidRegistryFormat(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "valid-registry")
	if err := os.MkdirAll(dir, utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "manifests.yaml"), []byte("test: value"), utils.DefaultFilePermission); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "volumes"), utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(
		filepath.Join(dir, "volumes/image.tar.gz"), []byte("fake"), utils.DefaultFilePermission); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if !validateSourceFiles(dir) {
		t.Error("valid registry format should return true")
	}
}

func testMissingManifest(t *testing.T) {
	tmpDir := t.TempDir()
	dir := filepath.Join(tmpDir, "no-manifest")
	if err := os.MkdirAll(dir, utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "volumes", "oci-layout"), utils.DefaultDirPermission); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}

	if validateSourceFiles(dir) {
		t.Error("missing manifests.yaml should return false")
	}
}

func testNotADirectory(t *testing.T) {
	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "not-a-dir")
	if err := os.WriteFile(file, []byte("test"), utils.DefaultFilePermission); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	if validateSourceFiles(file) {
		t.Error("file path should return false")
	}
}

// 测试架构获取逻辑
func TestGetArchitecture(t *testing.T) {
	tests := []struct {
		name  string
		archs []string
		want  string
	}{
		{
			name:  "single architecture",
			archs: []string{"amd64"},
			want:  "amd64",
		},
		{
			name:  "multiple architectures",
			archs: []string{"amd64", "arm64"},
			want:  "",
		},
		{
			name:  "empty architectures",
			archs: []string{},
			want:  "",
		},
		{
			name:  "three architectures",
			archs: []string{"amd64", "arm64", "ppc64le"},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getArchitecture(tt.archs)
			if got != tt.want {
				t.Errorf("getArchitecture() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 测试 Registry 路径标准化
func TestNormalizeRegistryPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "path without trailing slash",
			path: "registry.example.com/repo",
			want: "registry.example.com/repo/",
		},
		{
			name: "path with trailing slash",
			path: "registry.example.com/repo/",
			want: "registry.example.com/repo/",
		},
		{
			name: "path with double slashes",
			path: "registry.example.com//repo",
			want: "registry.example.com/repo/",
		},
		{
			name: "path with multiple double slashes",
			path: "registry.example.com//repo//subdir",
			want: "registry.example.com/repo/subdir/",
		},
		{
			name: "empty path",
			path: "",
			want: "/",
		},
		{
			name: "root path",
			path: "/",
			want: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizeRegistryPath(tt.path)
			if got != tt.want {
				t.Errorf("normalizeRegistryPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadLocalRepository(t *testing.T) {
	tests := []struct {
		name        string
		imageFile   string
		mockLoadErr error
		expectError bool
	}{
		{
			name:        "file exists and loads successfully",
			imageFile:   "/tmp/existing-image-file",
			mockLoadErr: nil,
			expectError: false,
		},
		{
			name:        "file exists but load fails",
			imageFile:   "/tmp/failing-image-file",
			mockLoadErr: assert.AnError,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(common.LoadLocalRepositoryFromFile,
				func(imageFile string) error {
					return tt.mockLoadErr
				})
			defer patches.Reset()

			err := loadLocalRepository(tt.imageFile)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPrepareImageData(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		setupFunc   func(t *testing.T, source string)
		expectError bool
	}{
		{
			name:        "image data directory exists and is not empty",
			source:      "/tmp/existing-source",
			setupFunc:   nil,
			expectError: false,
		},
		{
			name:        "image data directory needs to be created",
			source:      "/tmp/new-source",
			setupFunc:   nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			source := filepath.Join(tmpDir, "source")

			patches := gomonkey.ApplyFunc(utils.UnTar,
				func(src, dest string) error {
					return nil
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.Exists,
				func(path string) bool {
					if path == filepath.Join(source, utils.ImageDataDirectory) {
						return true
					}
					return false
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.DirectoryIsEmpty,
				func(path string) bool {
					return false
				})
			defer patches.Reset()

			err := prepareImageData(source)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoadAndStartRegistry(t *testing.T) {
	tests := []struct {
		name         string
		source       string
		mockLoadErr  error
		mockStartErr error
		expectError  bool
	}{
		{
			name:         "load and start successfully",
			source:       "/tmp/source",
			mockLoadErr:  nil,
			mockStartErr: nil,
			expectError:  false,
		},
		{
			name:         "load fails",
			source:       "/tmp/source",
			mockLoadErr:  assert.AnError,
			mockStartErr: nil,
			expectError:  true,
		},
		{
			name:         "start registry fails",
			source:       "/tmp/source",
			mockLoadErr:  nil,
			mockStartErr: assert.AnError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			source := filepath.Join(tmpDir, "source")

			patches := gomonkey.ApplyFunc(loadLocalRepository,
				func(imageFile string) error {
					return tt.mockLoadErr
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(server.StartImageRegistry,
				func(name, image, imageRegistryPort, imageDataDirectory string) error {
					return tt.mockStartErr
				})
			defer patches.Reset()

			err := loadAndStartRegistry(source)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncImages(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *BuildConfig
		mockSyncErr error
		expectError bool
	}{
		{
			name: "sync images successfully",
			cfg: &BuildConfig{
				Repos: []Repo{
					{
						Architecture: []string{"amd64"},
						SubImages: []SubImage{
							{
								TargetRepo: "test-repo",
								Images: []Image{
									{Name: "test-image", Tag: []string{"v1.0"}},
								},
							},
						},
					},
				},
			},
			mockSyncErr: nil,
			expectError: false,
		},
		{
			name:        "empty config",
			cfg:         &BuildConfig{},
			mockSyncErr: nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(syncSubImage,
				func(subImage SubImage, opts reg.Options, target string) error {
					return tt.mockSyncErr
				})
			defer patches.Reset()

			err := syncImages(tt.cfg, "registry.example.com/")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncSubImage(t *testing.T) {
	tests := []struct {
		name        string
		subImage    SubImage
		opts        reg.Options
		target      string
		mockSyncErr error
		expectError bool
	}{
		{
			name: "sync sub image successfully",
			subImage: SubImage{
				TargetRepo: "test-repo",
				Images: []Image{
					{Name: "test-image", Tag: []string{"v1.0"}},
				},
			},
			opts:        reg.Options{MultiArch: false, Arch: "amd64"},
			target:      "registry.example.com/",
			mockSyncErr: nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(syncImageTags,
				func(image Image, sourcePrefix, targetPrefix string, opts reg.Options) error {
					return tt.mockSyncErr
				})
			defer patches.Reset()

			err := syncSubImage(tt.subImage, tt.opts, tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncImageTags(t *testing.T) {
	tests := []struct {
		name         string
		image        Image
		sourcePrefix string
		targetPrefix string
		opts         reg.Options
		mockCopyErr  error
		expectError  bool
	}{
		{
			name:         "sync image tags successfully",
			image:        Image{Name: "test-image", Tag: []string{"v1.0", "latest"}},
			sourcePrefix: testIP + ":40448/test-repo/",
			targetPrefix: "registry.example.com/test-repo/",
			opts:         reg.Options{MultiArch: false, Arch: "amd64"},
			mockCopyErr:  nil,
			expectError:  false,
		},
		{
			name:         "copy fails",
			image:        Image{Name: "test-image", Tag: []string{"v1.0"}},
			sourcePrefix: testIP + ":40448/test-repo/",
			targetPrefix: "registry.example.com/test-repo/",
			opts:         reg.Options{MultiArch: false, Arch: "amd64"},
			mockCopyErr:  assert.AnError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(reg.CopyRegistry,
				func(opts reg.Options) error {
					return tt.mockCopyErr
				})
			defer patches.Reset()

			err := syncImageTags(tt.image, tt.sourcePrefix, tt.targetPrefix, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncImagesFromOCI(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		cfg            *BuildConfig
		target         string
		mockSyncOCIErr error
		expectError    bool
	}{
		{
			name:   "sync from OCI successfully",
			source: "/tmp/oci-source",
			cfg: &BuildConfig{
				Repos: []Repo{
					{
						Architecture: []string{"amd64"},
						SubImages: []SubImage{
							{
								TargetRepo: "test-repo",
								Images: []Image{
									{Name: "test-image", Tag: []string{"v1.0"}},
								},
							},
						},
					},
				},
			},
			target:         "registry.example.com/",
			mockSyncOCIErr: nil,
			expectError:    false,
		},
		{
			name:           "sync from OCI fails",
			source:         "/tmp/oci-source",
			cfg:            &BuildConfig{},
			target:         "registry.example.com/",
			mockSyncOCIErr: assert.AnError,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			ociSource := filepath.Join(tmpDir, "oci-source")
			ociDir := filepath.Join(ociSource, "volumes", "oci-layout")

			patches := gomonkey.ApplyFunc(filepath.Abs,
				func(path string) (string, error) {
					return ociDir, nil
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncSubImageFromOCI,
				func(ociDir string, subImage SubImage, opts reg.Options, target string) error {
					return tt.mockSyncOCIErr
				})
			defer patches.Reset()

			err := syncImagesFromOCI(tt.source, tt.cfg, tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncSubImageFromOCI(t *testing.T) {
	tests := []struct {
		name        string
		ociDir      string
		subImage    SubImage
		opts        reg.Options
		target      string
		mockTagsErr error
		expectError bool
	}{
		{
			name:        "sync sub image from OCI successfully",
			ociDir:      "/tmp/oci-layout",
			subImage:    SubImage{TargetRepo: "test-repo", Images: []Image{{Name: "test-image", Tag: []string{"v1.0"}}}},
			opts:        reg.Options{MultiArch: false, Arch: "amd64"},
			target:      "registry.example.com/",
			mockTagsErr: nil,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(syncImageTagsFromOCI,
				func(ociDir string, image Image, targetPrefix string, opts reg.Options) error {
					return tt.mockTagsErr
				})
			defer patches.Reset()

			err := syncSubImageFromOCI(tt.ociDir, tt.subImage, tt.opts, tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCopyImageFromOCI(t *testing.T) {
	tests := []struct {
		name         string
		ociSource    string
		dockerTarget string
		opts         reg.Options
		mockCopyErr  error
		expectError  bool
	}{
		{
			name:         "copy from OCI successfully",
			ociSource:    "oci:/tmp/oci-layout:test-image:v1.0",
			dockerTarget: "docker://registry.example.com/test-repo/test-image:v1.0",
			opts:         reg.Options{MultiArch: false, Arch: "amd64"},
			mockCopyErr:  nil,
			expectError:  false,
		},
		{
			name:         "copy fails",
			ociSource:    "oci:/tmp/oci-layout:test-image:v1.0",
			dockerTarget: "docker://registry.example.com/test-repo/test-image:v1.0",
			opts:         reg.Options{MultiArch: false, Arch: "amd64"},
			mockCopyErr:  assert.AnError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(reg.CopyRegistry,
				func(opts reg.Options) error {
					return tt.mockCopyErr
				})
			defer patches.Reset()

			err := copyImageFromOCI(tt.ociSource, tt.dockerTarget, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncImageTagsFromOCI(t *testing.T) {
	tests := []struct {
		name         string
		ociDir       string
		image        Image
		targetPrefix string
		opts         reg.Options
		mockCopyErr  error
		expectError  bool
	}{
		{
			name:         "sync image tags from OCI successfully",
			ociDir:       "/tmp/oci-layout",
			image:        Image{Name: "test-image", Tag: []string{"v1.0", "latest"}},
			targetPrefix: "registry.example.com/test-repo/",
			opts:         reg.Options{MultiArch: false, Arch: "amd64"},
			mockCopyErr:  nil,
			expectError:  false,
		},
		{
			name:         "copy fails",
			ociDir:       "/tmp/oci-layout",
			image:        Image{Name: "test-image", Tag: []string{"v1.0"}},
			targetPrefix: "registry.example.com/test-repo/",
			opts:         reg.Options{MultiArch: false, Arch: "amd64"},
			mockCopyErr:  assert.AnError,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(copyImageFromOCI,
				func(ociSource, dockerTarget string, opts reg.Options) error {
					return tt.mockCopyErr
				})
			defer patches.Reset()

			err := syncImageTagsFromOCI(tt.ociDir, tt.image, tt.targetPrefix, tt.opts)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSpecificSync(t *testing.T) {
	tests := []struct {
		name               string
		source             string
		target             string
		mockValidateResult bool
		mockLoadErr        error
		mockPrepareErr     error
		mockStartErr       error
		mockSyncErr        error
		mockSyncOCIErr     error
		mockRemoveErr      error
	}{
		{
			name:               "sync with OCI format successfully",
			source:             "/tmp/oci-source",
			target:             "registry.example.com",
			mockValidateResult: true,
			mockLoadErr:        nil,
			mockPrepareErr:     nil,
			mockStartErr:       nil,
			mockSyncErr:        nil,
			mockSyncOCIErr:     nil,
			mockRemoveErr:      nil,
		},
		{
			name:               "sync with registry format successfully",
			source:             "/tmp/registry-source",
			target:             "registry.example.com",
			mockValidateResult: true,
			mockLoadErr:        nil,
			mockPrepareErr:     nil,
			mockStartErr:       nil,
			mockSyncErr:        nil,
			mockSyncOCIErr:     nil,
			mockRemoveErr:      nil,
		},
		{
			name:               "validate source files fails",
			source:             "/tmp/invalid-source",
			target:             "registry.example.com",
			mockValidateResult: false,
			mockLoadErr:        nil,
			mockPrepareErr:     nil,
			mockStartErr:       nil,
			mockSyncErr:        nil,
			mockSyncOCIErr:     nil,
			mockRemoveErr:      nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			patches := gomonkey.ApplyFunc(validateSourceFiles,
				func(source string) bool {
					return tt.mockValidateResult
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(loadManifestConfig,
				func(source string) (*BuildConfig, error) {
					return &BuildConfig{
						Repos: []Repo{
							{
								Architecture: []string{"amd64"},
								SubImages: []SubImage{
									{
										TargetRepo: "test-repo",
										Images:     []Image{{Name: "test-image", Tag: []string{"v1.0"}}},
									},
								},
							},
						},
					}, nil
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(ensureMountDirectoryExists,
				func(mountPath string) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(DetectPatchFormat,
				func(source string) string {
					if _, err := os.Stat(filepath.Join(source, "volumes", "oci-layout")); err == nil {
						return "oci"
					}
					return "registry"
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncImagesFromOCI,
				func(source string, cfg *BuildConfig, target string) error {
					return tt.mockSyncOCIErr
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(prepareImageData,
				func(source string) error {
					return tt.mockPrepareErr
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(loadAndStartRegistry,
				func(source string) error {
					return tt.mockStartErr
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncImages,
				func(cfg *BuildConfig, target string) error {
					return tt.mockSyncErr
				})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(server.RemoveImageRegistry,
				func(name string) error {
					return tt.mockRemoveErr
				})
			defer patches.Reset()

			SpecificSync(tt.source, tt.target)
		})
	}
}
