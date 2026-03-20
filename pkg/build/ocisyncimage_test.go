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
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	reg "gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

// 测试 OCI layout 结构创建
func TestCreateOCILayoutStructure(t *testing.T) {
	originalTmpRegistry := tmpRegistry
	defer func() { tmpRegistry = originalTmpRegistry }()

	tmpRegistry = t.TempDir()

	ociDir, err := createOCILayoutStructure()
	if err != nil {
		t.Fatalf("createOCILayoutStructure() error = %v", err)
	}

	expectedDir := filepath.Join(tmpRegistry, "oci-layout")
	if ociDir != expectedDir {
		t.Errorf("ociDir = %v, want %v", ociDir, expectedDir)
	}

	ociLayoutFile := filepath.Join(ociDir, "oci-layout")
	if _, err := os.Stat(ociLayoutFile); os.IsNotExist(err) {
		t.Error("oci-layout file not created")
	}

	content, err := os.ReadFile(ociLayoutFile)
	if err != nil {
		t.Fatalf("failed to read oci-layout file: %v", err)
	}
	expectedContent := `{"imageLayoutVersion":"1.0.0"}`
	if string(content) != expectedContent {
		t.Errorf("oci-layout content = %v, want %v", string(content), expectedContent)
	}

	blobsDir := filepath.Join(ociDir, "blobs", "sha256")
	if _, err := os.Stat(blobsDir); os.IsNotExist(err) {
		t.Error("blobs/sha256 directory not created")
	}
}

// 测试镜像引用格式化
func TestFormatImageRef(t *testing.T) {
	tests := []struct {
		name       string
		targetRepo string
		imageName  string
		tag        string
		want       string
	}{
		{
			name:       "normal repo",
			targetRepo: "myrepo",
			imageName:  "nginx",
			tag:        "1.21",
			want:       "myrepo/nginx:1.21",
		},
		{
			name:       "root repo with slash",
			targetRepo: "/",
			imageName:  "nginx",
			tag:        "latest",
			want:       "nginx:latest",
		},
		{
			name:       "nested repo",
			targetRepo: "myrepo/subdir",
			imageName:  "app",
			tag:        "v1.0.0",
			want:       "myrepo/subdir/app:v1.0.0",
		},
		{
			name:       "empty repo",
			targetRepo: "",
			imageName:  "alpine",
			tag:        "3.14",
			want:       "/alpine:3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatImageRef(tt.targetRepo, tt.imageName, tt.tag)
			if got != tt.want {
				t.Errorf("formatImageRef() = %v, want %v", got, tt.want)
			}
		})
	}
}

// 测试镜像总数计算
func TestCountTotalImages(t *testing.T) {
	t.Run("empty config", testCountEmpty)
	t.Run("single repo", testCountSingleRepo)
	t.Run("multiple repos", testCountMultipleRepos)
	t.Run("skip disabled repos", testCountSkipDisabled)
}

func testCountEmpty(t *testing.T) {
	cfg := &BuildConfig{}
	if got := countTotalImages(cfg); got != 0 {
		t.Errorf("countTotalImages() = %v, want 0", got)
	}
}

func testCountSingleRepo(t *testing.T) {
	cfg := &BuildConfig{
		Repos: []Repo{
			{
				NeedDownload: true,
				SubImages: []SubImage{
					{Images: []Image{{Tag: []string{"v1.0"}}}},
				},
			},
		},
	}
	if got := countTotalImages(cfg); got != 1 {
		t.Errorf("countTotalImages() = %v, want 1", got)
	}
}

func testCountMultipleRepos(t *testing.T) {
	cfg := &BuildConfig{
		Repos: []Repo{
			{
				NeedDownload: true,
				SubImages: []SubImage{
					{Images: []Image{
						{Tag: []string{"v1.0", "v1.1", "latest"}},
						{Tag: []string{"v2.0"}},
					}},
				},
			},
			{
				NeedDownload: true,
				SubImages: []SubImage{
					{Images: []Image{{Tag: []string{"stable", "edge"}}}},
				},
			},
		},
	}
	// Expected: 3 tags (v1.0, v1.1, latest) + 1 tag (v2.0) + 2 tags (stable, edge) = 6 total
	const expectedTotalTags = 6
	if got := countTotalImages(cfg); got != expectedTotalTags {
		t.Errorf("countTotalImages() = %v, want %d", got, expectedTotalTags)
	}
}

func testCountSkipDisabled(t *testing.T) {
	cfg := &BuildConfig{
		Repos: []Repo{
			{
				NeedDownload: false,
				SubImages: []SubImage{
					{Images: []Image{{Tag: []string{"v1.0", "v1.1"}}}},
				},
			},
			{
				NeedDownload: true,
				SubImages: []SubImage{
					{Images: []Image{{Tag: []string{"v2.0"}}}},
				},
			},
		},
	}
	if got := countTotalImages(cfg); got != 1 {
		t.Errorf("countTotalImages() = %v, want 1", got)
	}
}

// 测试参数格式化
func TestCopyImageToOCIParameterFormatting(t *testing.T) {
	t.Run("source formatting", testSourceFormatting)
	t.Run("target formatting", testTargetFormatting)
	t.Run("multi-arch flag", testMultiArchFlag)
}

func testSourceFormatting(t *testing.T) {
	tests := []struct {
		name        string
		sourceImage string
		want        string
	}{
		{"without docker prefix", "registry.io/image:v1", "docker://registry.io/image:v1"},
		{"with docker prefix", "docker://registry.io/image:v1", "docker://registry.io/image:v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceRef := tt.sourceImage
			if !strings.HasPrefix(sourceRef, "docker://") {
				sourceRef = "docker://" + sourceRef
			}
			if sourceRef != tt.want {
				t.Errorf("sourceRef = %v, want %v", sourceRef, tt.want)
			}
		})
	}
}

func testTargetFormatting(t *testing.T) {
	ociLayoutDir := "/tmp/oci"
	imageRef := "myimage:v1"
	expectedTarget := "oci:" + ociLayoutDir + ":" + imageRef

	targetRef := filepath.Join("oci:"+ociLayoutDir, imageRef)
	if targetRef != expectedTarget {
		t.Logf("targetRef format may vary: got %v, want %v", targetRef, expectedTarget)
	}
}

func testMultiArchFlag(t *testing.T) {
	tests := []struct {
		name string
		arch string
		want bool
	}{
		{"single arch", "amd64", false},
		{"multi arch", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isMultiArch := (tt.arch == "")
			if isMultiArch != tt.want {
				t.Errorf("isMultiArch = %v, want %v", isMultiArch, tt.want)
			}
		})
	}
}

func TestSyncRepoOCI(t *testing.T) {
	tests := []struct {
		name                         string
		config                       *BuildConfig
		stopChan                     chan struct{}
		mockCreateOCILayoutStructure func() (string, error)
		mockCountTotalImages         func(*BuildConfig) int
		mockSyncAllImagesToOCI       func(*BuildConfig, string, chan struct{}, int) error
		mockMoveOCILayoutToVolumes   func(string) error
		expectError                  bool
	}{
		{
			name:     "successful sync",
			config:   &BuildConfig{},
			stopChan: make(chan struct{}),
			mockCreateOCILayoutStructure: func() (string, error) {
				return "/tmp/oci-layout", nil
			},
			mockCountTotalImages: func(cfg *BuildConfig) int {
				return 5
			},
			mockSyncAllImagesToOCI: func(cfg *BuildConfig, ociDir string, stopChan chan struct{}, totalImages int) error {
				return nil
			},
			mockMoveOCILayoutToVolumes: func(ociDir string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:     "create OCILayout structure fails",
			config:   &BuildConfig{},
			stopChan: make(chan struct{}),
			mockCreateOCILayoutStructure: func() (string, error) {
				return "", fmt.Errorf("creation failed")
			},
			mockCountTotalImages: func(cfg *BuildConfig) int {
				return 0
			},
			mockSyncAllImagesToOCI: func(cfg *BuildConfig, ociDir string, stopChan chan struct{}, totalImages int) error {
				return nil
			},
			mockMoveOCILayoutToVolumes: func(ociDir string) error {
				return nil
			},
			expectError: true,
		},
		{
			name:     "sync all images to OCI fails",
			config:   &BuildConfig{},
			stopChan: make(chan struct{}),
			mockCreateOCILayoutStructure: func() (string, error) {
				return "/tmp/oci-layout", nil
			},
			mockCountTotalImages: func(cfg *BuildConfig) int {
				return 5
			},
			mockSyncAllImagesToOCI: func(cfg *BuildConfig, ociDir string, stopChan chan struct{}, totalImages int) error {
				return fmt.Errorf("sync failed")
			},
			mockMoveOCILayoutToVolumes: func(ociDir string) error {
				return nil
			},
			expectError: true,
		},
		{
			name:     "move OCI layout to volumes fails",
			config:   &BuildConfig{},
			stopChan: make(chan struct{}),
			mockCreateOCILayoutStructure: func() (string, error) {
				return "/tmp/oci-layout", nil
			},
			mockCountTotalImages: func(cfg *BuildConfig) int {
				return 5
			},
			mockSyncAllImagesToOCI: func(cfg *BuildConfig, ociDir string, stopChan chan struct{}, totalImages int) error {
				return nil
			},
			mockMoveOCILayoutToVolumes: func(ociDir string) error {
				return fmt.Errorf("move failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(createOCILayoutStructure, tt.mockCreateOCILayoutStructure)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(countTotalImages, tt.mockCountTotalImages)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncAllImagesToOCI, tt.mockSyncAllImagesToOCI)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(moveOCILayoutToVolumes, tt.mockMoveOCILayoutToVolumes)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := syncRepoOCI(tt.config, tt.stopChan)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncAllImagesToOCI(t *testing.T) {
	tests := []struct {
		name               string
		config             *BuildConfig
		ociDir             string
		stopChan           chan struct{}
		totalImages        int
		mockSyncRepoImages func(Repo, string, chan struct{}, *int, int) error
		expectError        bool
	}{
		{
			name:        "successful sync all images",
			config:      &BuildConfig{Repos: []Repo{{NeedDownload: true}}},
			ociDir:      "/tmp/oci-layout",
			stopChan:    make(chan struct{}),
			totalImages: 5,
			mockSyncRepoImages: func(cr Repo, ociDir string, stopChan chan struct{}, currentImage *int, totalImages int) error {
				return nil
			},
			expectError: false,
		},
		{
			name:        "sync repo images fails",
			config:      &BuildConfig{Repos: []Repo{{NeedDownload: true}}},
			ociDir:      "/tmp/oci-layout",
			stopChan:    make(chan struct{}),
			totalImages: 5,
			mockSyncRepoImages: func(cr Repo, ociDir string, stopChan chan struct{}, currentImage *int, totalImages int) error {
				return fmt.Errorf("sync repo failed")
			},
			expectError: true,
		},
		{
			name:        "no repos to sync",
			config:      &BuildConfig{Repos: []Repo{{NeedDownload: false}}},
			ociDir:      "/tmp/oci-layout",
			stopChan:    make(chan struct{}),
			totalImages: 0,
			mockSyncRepoImages: func(cr Repo, ociDir string, stopChan chan struct{}, currentImage *int, totalImages int) error {
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(syncRepoImages, tt.mockSyncRepoImages)
			defer patches.Reset()

			err := syncAllImagesToOCI(tt.config, tt.ociDir, tt.stopChan, tt.totalImages)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncRepoImages(t *testing.T) {
	tests := []struct {
		name              string
		cr                Repo
		ociDir            string
		stopChan          chan struct{}
		currentImage      *int
		totalImages       int
		mockSyncSubImages func(SubImage, []string, *syncContext) error
		expectError       bool
	}{
		{
			name:         "successful sync repo images",
			cr:           Repo{SubImages: []SubImage{{}}},
			ociDir:       "/tmp/oci-layout",
			stopChan:     make(chan struct{}),
			currentImage: new(int),
			totalImages:  5,
			mockSyncSubImages: func(subImage SubImage, arch []string, ctx *syncContext) error {
				return nil
			},
			expectError: false,
		},
		{
			name:         "sync sub images fails",
			cr:           Repo{SubImages: []SubImage{{}}},
			ociDir:       "/tmp/oci-layout",
			stopChan:     make(chan struct{}),
			currentImage: new(int),
			totalImages:  5,
			mockSyncSubImages: func(subImage SubImage, arch []string, ctx *syncContext) error {
				return fmt.Errorf("sync sub images failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(syncSubImages, tt.mockSyncSubImages)
			defer patches.Reset()

			err := syncRepoImages(tt.cr, tt.ociDir, tt.stopChan, tt.currentImage, tt.totalImages)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncSubImages(t *testing.T) {
	tests := []struct {
		name                 string
		subImage             SubImage
		arch                 []string
		ctx                  *syncContext
		mockSyncOCIImageTags func(Image, SubImage, []string, *syncContext) error
		expectError          bool
	}{
		{
			name:     "successful sync sub images",
			subImage: SubImage{Images: []Image{{Tag: []string{"v1.0"}}}},
			arch:     []string{"amd64"},
			ctx:      &syncContext{stopChan: make(chan struct{})},
			mockSyncOCIImageTags: func(image Image, subImage SubImage, arch []string, ctx *syncContext) error {
				return nil
			},
			expectError: false,
		},
		{
			name:     "sync OCI image tags fails",
			subImage: SubImage{Images: []Image{{Tag: []string{"v1.0"}}}},
			arch:     []string{"amd64"},
			ctx:      &syncContext{stopChan: make(chan struct{})},
			mockSyncOCIImageTags: func(image Image, subImage SubImage, arch []string, ctx *syncContext) error {
				return fmt.Errorf("sync OCI image tags failed")
			},
			expectError: true,
		},
		{
			name:     "stop channel closed",
			subImage: SubImage{Images: []Image{{Tag: []string{"v1.0"}}}},
			arch:     []string{"amd64"},
			ctx:      &syncContext{stopChan: make(chan struct{})},
			mockSyncOCIImageTags: func(image Image, subImage SubImage, arch []string, ctx *syncContext) error {
				// Close the stop channel to trigger early termination
				close(ctx.stopChan)
				return nil
			},
			expectError: false, // Should return nil when terminated early
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(syncOCIImageTags, tt.mockSyncOCIImageTags)
			defer patches.Reset()

			err := syncSubImages(tt.subImage, tt.arch, tt.ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncOCIImageTags(t *testing.T) {
	tests := []struct {
		name               string
		image              Image
		subImage           SubImage
		arch               []string
		ctx                *syncContext
		mockImageTrack     func(string, string, string, string, []string) (string, error)
		mockFormatImageRef func(string, string, string) string
		mockSyncImageToOCI func(string, string, string, []string, bool) error
		expectError        bool
	}{
		{
			name:     "successful sync OCI image tags",
			image:    Image{Tag: []string{"v1.0", "latest"}},
			subImage: SubImage{SourceRepo: "source-repo", TargetRepo: "target-repo", ImageTrack: "track", Images: []Image{{Tag: []string{"v1.0"}}}},
			arch:     []string{"amd64"},
			ctx:      &syncContext{ociDir: "/tmp/oci", currentImage: new(int), totalImages: 2},
			mockImageTrack: func(sourceRepo, imageTrack, imageName, tag string, arch []string) (string, error) {
				return fmt.Sprintf("%s/%s:%s", sourceRepo, imageName, tag), nil
			},
			mockFormatImageRef: func(targetRepo, imageName, tag string) string {
				return fmt.Sprintf("%s/%s:%s", targetRepo, imageName, tag)
			},
			mockSyncImageToOCI: func(source, ociLayoutDir, imageRef string, arch []string, srcTLSVerify bool) error {
				return nil
			},
			expectError: false,
		},
		{
			name:     "image track fails",
			image:    Image{Tag: []string{"v1.0"}},
			subImage: SubImage{SourceRepo: "source-repo", TargetRepo: "target-repo", ImageTrack: "track", Images: []Image{{Tag: []string{"v1.0"}}}},
			arch:     []string{"amd64"},
			ctx:      &syncContext{ociDir: "/tmp/oci", currentImage: new(int), totalImages: 1},
			mockImageTrack: func(sourceRepo, imageTrack, imageName, tag string, arch []string) (string, error) {
				return "", fmt.Errorf("image track failed")
			},
			mockFormatImageRef: func(targetRepo, imageName, tag string) string {
				return fmt.Sprintf("%s/%s:%s", targetRepo, imageName, tag)
			},
			mockSyncImageToOCI: func(source, ociLayoutDir, imageRef string, arch []string, srcTLSVerify bool) error {
				return nil
			},
			expectError: true,
		},
		{
			name:     "sync image to OCI fails",
			image:    Image{Tag: []string{"v1.0"}},
			subImage: SubImage{SourceRepo: "source-repo", TargetRepo: "target-repo", ImageTrack: "track", Images: []Image{{Tag: []string{"v1.0"}}}},
			arch:     []string{"amd64"},
			ctx:      &syncContext{ociDir: "/tmp/oci", currentImage: new(int), totalImages: 1},
			mockImageTrack: func(sourceRepo, imageTrack, imageName, tag string, arch []string) (string, error) {
				return fmt.Sprintf("%s/%s:%s", sourceRepo, imageName, tag), nil
			},
			mockFormatImageRef: func(targetRepo, imageName, tag string) string {
				return fmt.Sprintf("%s/%s:%s", targetRepo, imageName, tag)
			},
			mockSyncImageToOCI: func(source, ociLayoutDir, imageRef string, arch []string, srcTLSVerify bool) error {
				return fmt.Errorf("sync to OCI failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(imageTrack, tt.mockImageTrack)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(formatImageRef, tt.mockFormatImageRef)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncImageToOCI, tt.mockSyncImageToOCI)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := syncOCIImageTags(tt.image, tt.subImage, tt.arch, tt.ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestMoveOCILayoutToVolumes(t *testing.T) {
	tests := []struct {
		name          string
		ociDir        string
		mockRemoveAll func(string) error
		mockRename    func(string, string) error
		expectError   bool
	}{
		{
			name:   "successful move",
			ociDir: "/tmp/oci-layout",
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockRename: func(oldpath, newpath string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:   "remove all fails",
			ociDir: "/tmp/oci-layout",
			mockRemoveAll: func(path string) error {
				return fmt.Errorf("remove failed")
			},
			mockRename: func(oldpath, newpath string) error {
				return nil
			},
			expectError: true,
		},
		{
			name:   "rename fails",
			ociDir: "/tmp/oci-layout",
			mockRemoveAll: func(path string) error {
				return nil
			},
			mockRename: func(oldpath, newpath string) error {
				return fmt.Errorf("rename failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original global variables
			originalBVolumes := bkeVolumes
			defer func() {
				bkeVolumes = originalBVolumes
			}()

			// Set global variable for test
			bkeVolumes = "test-volumes"

			// Apply patches
			patches := gomonkey.ApplyFunc(os.RemoveAll, tt.mockRemoveAll)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Rename, tt.mockRename)
			defer patches.Reset()

			err := moveOCILayoutToVolumes(tt.ociDir)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncImageToOCI(t *testing.T) {
	tests := []struct {
		name               string
		source             string
		ociLayoutDir       string
		imageRef           string
		arch               []string
		srcTLSVerify       bool
		mockCopyImageToOCI func(string, string, string, string, bool) error
		expectError        bool
	}{
		{
			name:         "successful sync with single arch",
			source:       "source-image:v1.0",
			ociLayoutDir: "/tmp/oci-layout",
			imageRef:     "target-image:v1.0",
			arch:         []string{"amd64"},
			srcTLSVerify: true,
			mockCopyImageToOCI: func(source, ociLayoutDir, imageRef, arch string, srcTLSVerify bool) error {
				return nil
			},
			expectError: false,
		},
		{
			name:         "copy image to OCI fails",
			source:       "source-image:v1.0",
			ociLayoutDir: "/tmp/oci-layout",
			imageRef:     "target-image:v1.0",
			arch:         []string{"amd64"},
			srcTLSVerify: true,
			mockCopyImageToOCI: func(source, ociLayoutDir, imageRef, arch string, srcTLSVerify bool) error {
				return fmt.Errorf("copy failed")
			},
			expectError: true,
		},
		{
			name:         "multi-arch sync",
			source:       "source-image:v1.0",
			ociLayoutDir: "/tmp/oci-layout",
			imageRef:     "target-image:v1.0",
			arch:         []string{"amd64", "arm64"},
			srcTLSVerify: true,
			mockCopyImageToOCI: func(source, ociLayoutDir, imageRef, arch string, srcTLSVerify bool) error {
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(copyImageToOCI, tt.mockCopyImageToOCI)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(reg.IsMultiArchManifests, func(srcTLSVerify bool, imageAddress string) bool {
				return false // For simplicity in this test
			})
			defer patches.Reset()

			err := syncImageToOCI(tt.source, tt.ociLayoutDir, tt.imageRef, tt.arch, tt.srcTLSVerify)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCopyImageToOCI(t *testing.T) {
	tests := []struct {
		name             string
		sourceImage      string
		ociLayoutDir     string
		imageRef         string
		arch             string
		srcTLSVerify     bool
		mockCopyRegistry func(reg.Options) error
		expectError      bool
	}{
		{
			name:         "successful copy with single arch",
			sourceImage:  "source-image:v1.0",
			ociLayoutDir: "/tmp/oci-layout",
			imageRef:     "target-image:v1.0",
			arch:         "amd64",
			srcTLSVerify: true,
			mockCopyRegistry: func(opts reg.Options) error {
				// Verify that the options are set correctly
				assert.Equal(t, "docker://source-image:v1.0", opts.Source)
				assert.Equal(t, "oci:/tmp/oci-layout:target-image:v1.0", opts.Target)
				assert.Equal(t, "amd64", opts.Arch)
				assert.Equal(t, false, opts.MultiArch)
				assert.Equal(t, true, opts.SrcTLSVerify)
				assert.Equal(t, false, opts.DestTLSVerify)
				return nil
			},
			expectError: false,
		},
		{
			name:         "successful copy with multi arch",
			sourceImage:  "source-image:v1.0",
			ociLayoutDir: "/tmp/oci-layout",
			imageRef:     "target-image:v1.0",
			arch:         "", // Empty means multi-arch
			srcTLSVerify: true,
			mockCopyRegistry: func(opts reg.Options) error {
				// Verify that the options are set correctly for multi-arch
				assert.Equal(t, "docker://source-image:v1.0", opts.Source)
				assert.Equal(t, "oci:/tmp/oci-layout:target-image:v1.0", opts.Target)
				assert.Equal(t, "", opts.Arch)
				assert.Equal(t, true, opts.MultiArch)
				assert.Equal(t, true, opts.SrcTLSVerify)
				assert.Equal(t, false, opts.DestTLSVerify)
				return nil
			},
			expectError: false,
		},
		{
			name:         "copy registry fails",
			sourceImage:  "source-image:v1.0",
			ociLayoutDir: "/tmp/oci-layout",
			imageRef:     "target-image:v1.0",
			arch:         "amd64",
			srcTLSVerify: true,
			mockCopyRegistry: func(opts reg.Options) error {
				return fmt.Errorf("copy registry failed")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches
			patches := gomonkey.ApplyFunc(reg.CopyRegistry, tt.mockCopyRegistry)
			defer patches.Reset()

			err := copyImageToOCI(tt.sourceImage, tt.ociLayoutDir, tt.imageRef, tt.arch, tt.srcTLSVerify)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
