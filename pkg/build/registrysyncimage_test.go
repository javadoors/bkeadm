/*
 *
 * Copyright (c) 2025 Bocloud Technologies Co., Ltd.
 * installer is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          <http://license.coscl.org.cn/MulanPSL2>
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 *
 */

package build

import (
	"fmt"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	reg "gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/pkg/server"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func TestSyncRepo(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	tests := []struct {
		name                    string
		mockRemoveImageRegistry func(string) error
		mockStartImageRegistry  func(string, string, string, string) error
		mockProcessRepoImages   func(Repo, chan struct{}) error
		mockPackImageAndCleanup func() error
		cfg                     *BuildConfig
		expectError             bool
	}{
		{
			name:                    "successful sync",
			mockRemoveImageRegistry: func(name string) error { return nil },
			mockStartImageRegistry:  func(name, image, port, dir string) error { return nil },
			mockProcessRepoImages:   func(repo Repo, stopChan chan struct{}) error { return nil },
			mockPackImageAndCleanup: func() error { return nil },
			cfg: &BuildConfig{
				Repos: []Repo{
					{
						NeedDownload: true,
						SubImages: []SubImage{
							{
								Images: []Image{
									{
										Name: "test-image",
										Tag:  []string{"v1.0"},
									},
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
		{
			name:                    "process repo images fails",
			mockRemoveImageRegistry: func(name string) error { return nil },
			mockStartImageRegistry:  func(name, image, port, dir string) error { return nil },
			mockProcessRepoImages:   func(repo Repo, stopChan chan struct{}) error { return fmt.Errorf("process error") },
			cfg: &BuildConfig{
				Repos: []Repo{
					{
						NeedDownload: true,
						SubImages: []SubImage{
							{
								Images: []Image{
									{
										Name: "test-image",
										Tag:  []string{"v1.0"},
									},
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
		{
			name:                    "pack image and cleanup fails",
			mockRemoveImageRegistry: func(name string) error { return nil },
			mockStartImageRegistry:  func(name, image, port, dir string) error { return nil },
			mockProcessRepoImages:   func(repo Repo, stopChan chan struct{}) error { return nil },
			mockPackImageAndCleanup: func() error { return fmt.Errorf("pack error") },
			cfg: &BuildConfig{
				Repos: []Repo{
					{
						NeedDownload: true,
						SubImages: []SubImage{
							{
								Images: []Image{
									{
										Name: "test-image",
										Tag:  []string{"v1.0"},
									},
								},
							},
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(server.RemoveImageRegistry, tt.mockRemoveImageRegistry)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(server.StartImageRegistry, tt.mockStartImageRegistry)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(processRepoImages, tt.mockProcessRepoImages)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(packImageAndCleanup, tt.mockPackImageAndCleanup)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			// Mock tmpRegistry variable
			patches = gomonkey.ApplyGlobalVar(&tmpRegistry, "/tmp/registry")
			defer patches.Reset()

			err := syncRepo(tt.cfg, stopChan)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessRepoImages(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	tests := []struct {
		name                      string
		repo                      Repo
		mockProcessSingleSubImage func(SubImage, []string, chan struct{}) error
		expectError               bool
	}{
		{
			name: "successful processing",
			repo: Repo{
				Architecture: []string{"amd64"},
				SubImages: []SubImage{
					{
						Images: []Image{
							{
								Name: "test-image",
								Tag:  []string{"v1.0"},
							},
						},
					},
				},
			},
			mockProcessSingleSubImage: func(subImage SubImage, arch []string, stopChan chan struct{}) error { return nil },
			expectError:               false,
		},
		{
			name: "process single sub image fails",
			repo: Repo{
				Architecture: []string{"amd64"},
				SubImages: []SubImage{
					{
						Images: []Image{
							{
								Name: "test-image",
								Tag:  []string{"v1.0"},
							},
						},
					},
				},
			},
			mockProcessSingleSubImage: func(subImage SubImage, arch []string, stopChan chan struct{}) error {
				return fmt.Errorf("process error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(processSingleSubImage, tt.mockProcessSingleSubImage)
			defer patches.Reset()

			err := processRepoImages(tt.repo, stopChan)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessSingleSubImage(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	tests := []struct {
		name                 string
		subImage             SubImage
		architecture         []string
		mockProcessImageTags func(Image, SubImage, []string) error
		expectError          bool
	}{
		{
			name: "successful processing",
			subImage: SubImage{
				Images: []Image{
					{
						Name: "test-image",
						Tag:  []string{"v1.0"},
					},
				},
			},
			architecture:         []string{"amd64"},
			mockProcessImageTags: func(image Image, subImage SubImage, arch []string) error { return nil },
			expectError:          false,
		},
		{
			name: "process image tags fails",
			subImage: SubImage{
				Images: []Image{
					{
						Name: "test-image",
						Tag:  []string{"v1.0"},
					},
				},
			},
			architecture:         []string{"amd64"},
			mockProcessImageTags: func(image Image, subImage SubImage, arch []string) error { return fmt.Errorf("process error") },
			expectError:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(processImageTags, tt.mockProcessImageTags)
			defer patches.Reset()

			err := processSingleSubImage(tt.subImage, tt.architecture, stopChan)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestProcessImageTags(t *testing.T) {
	tests := []struct {
		name              string
		image             Image
		subImage          SubImage
		architecture      []string
		mockImageTrack    func(string, string, string, string, []string) (string, error)
		mockSyncRepoImage func(string, string, []string, bool) error
		expectError       bool
	}{
		{
			name: "successful processing",
			image: Image{
				Name: "test-image",
				Tag:  []string{"v1.0"},
			},
			subImage: SubImage{
				SourceRepo: "source-repo",
				TargetRepo: "target-repo",
			},
			architecture: []string{"amd64"},
			mockImageTrack: func(sourceRepo, imageTrack, imageName, tag string, arch []string) (string, error) {
				return "source:image", nil
			},
			mockSyncRepoImage: func(source, target string, arch []string, srcTLSVerify bool) error { return nil },
			expectError:       false,
		},
		{
			name: "image track fails",
			image: Image{
				Name: "test-image",
				Tag:  []string{"v1.0"},
			},
			subImage: SubImage{
				SourceRepo: "source-repo",
				TargetRepo: "target-repo",
			},
			architecture: []string{"amd64"},
			mockImageTrack: func(sourceRepo, imageTrack, imageName, tag string, arch []string) (string, error) {
				return "", fmt.Errorf("track error")
			},
			expectError: true,
		},
		{
			name: "sync repo image fails",
			image: Image{
				Name: "test-image",
				Tag:  []string{"v1.0"},
			},
			subImage: SubImage{
				SourceRepo: "source-repo",
				TargetRepo: "target-repo",
			},
			architecture: []string{"amd64"},
			mockImageTrack: func(sourceRepo, imageTrack, imageName, tag string, arch []string) (string, error) {
				return "source:image", nil
			},
			mockSyncRepoImage: func(source, target string, arch []string, srcTLSVerify bool) error { return fmt.Errorf("sync error") },
			expectError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(imageTrack, tt.mockImageTrack)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncRepoImage, tt.mockSyncRepoImage)
			defer patches.Reset()

			err := processImageTags(tt.image, tt.subImage, tt.architecture)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncRepoImage(t *testing.T) {
	tests := []struct {
		name                    string
		source                  string
		target                  string
		arch                    []string
		srcTLSVerify            bool
		mockSyncSingleArchImage func(string, string, string, bool) error
		mockSyncMultiArchImage  func(string, string, []string, bool) error
		expectError             bool
	}{
		{
			name:                    "single architecture sync",
			source:                  "source:image",
			target:                  "target:image",
			arch:                    []string{"amd64"},
			srcTLSVerify:            true,
			mockSyncSingleArchImage: func(source, target, arch string, srcTLSVerify bool) error { return nil },
			mockSyncMultiArchImage:  func(source, target string, arch []string, srcTLSVerify bool) error { return nil },
			expectError:             false,
		},
		{
			name:                    "multi architecture sync",
			source:                  "source:image",
			target:                  "target:image",
			arch:                    []string{"amd64", "arm64"},
			srcTLSVerify:            true,
			mockSyncSingleArchImage: func(source, target, arch string, srcTLSVerify bool) error { return nil },
			mockSyncMultiArchImage:  func(source, target string, arch []string, srcTLSVerify bool) error { return nil },
			expectError:             false,
		},
		{
			name:                    "single arch sync fails",
			source:                  "source:image",
			target:                  "target:image",
			arch:                    []string{"amd64"},
			srcTLSVerify:            true,
			mockSyncSingleArchImage: func(source, target, arch string, srcTLSVerify bool) error { return fmt.Errorf("sync error") },
			expectError:             true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(syncSingleArchImage, tt.mockSyncSingleArchImage)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncMultiArchImage, tt.mockSyncMultiArchImage)
			defer patches.Reset()

			err := syncRepoImage(tt.source, tt.target, tt.arch, tt.srcTLSVerify)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncSingleArchImage(t *testing.T) {
	tests := []struct {
		name             string
		source           string
		target           string
		arch             string
		srcTLSVerify     bool
		mockCopyRegistry func(reg.Options) error
		expectError      bool
	}{
		{
			name:         "successful sync",
			source:       "source:image",
			target:       "target:image",
			arch:         "amd64",
			srcTLSVerify: true,
			mockCopyRegistry: func(opts reg.Options) error {
				// Verify the options are set correctly
				assert.Equal(t, false, opts.MultiArch)
				assert.Equal(t, true, opts.SrcTLSVerify)
				assert.Equal(t, false, opts.DestTLSVerify)
				assert.Equal(t, "amd64", opts.Arch)
				assert.Equal(t, "source:image", opts.Source)
				assert.Equal(t, "target:image", opts.Target)
				return nil
			},
			expectError: false,
		},
		{
			name:         "copy registry fails",
			source:       "source:image",
			target:       "target:image",
			arch:         "amd64",
			srcTLSVerify: true,
			mockCopyRegistry: func(opts reg.Options) error {
				return fmt.Errorf("copy error")
			},
			expectError: true,
		},
		{
			name:         "retry succeeds",
			source:       "source:image",
			target:       "target:image",
			arch:         "amd64",
			srcTLSVerify: true,
			mockCopyRegistry: func(opts reg.Options) error {
				// First call fails, second succeeds
				if opts.Source == "source:image" {
					return fmt.Errorf("first attempt fails")
				}
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(reg.CopyRegistry, tt.mockCopyRegistry)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := syncSingleArchImage(tt.source, tt.target, tt.arch, tt.srcTLSVerify)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRetrySyncWithArchSuffix(t *testing.T) {
	// Test the retry function with a mock
	patches := gomonkey.ApplyFunc(reg.CopyRegistry, func(opts reg.Options) error {
		// Verify that the source has the arch suffix
		assert.Equal(t, "source:image-amd64", opts.Source)
		assert.Equal(t, "target:image", opts.Target)
		return nil
	})
	defer patches.Reset()

	patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
	defer patches.Reset()

	op := reg.Options{
		Source: "source:image",
		Target: "target:image",
	}

	err := retrySyncWithArchSuffix("source:image", "target:image", "amd64", op)
	assert.NoError(t, err)
}

func TestSyncMultiArchImage(t *testing.T) {
	tests := []struct {
		name                                string
		source                              string
		target                              string
		arch                                []string
		srcTLSVerify                        bool
		mockTryDirectMultiArchSync          func(string, string, bool) error
		mockSyncArchImagesAndCreateManifest func(string, string, []string, bool) error
		expectError                         bool
	}{
		{
			name:                                "direct sync succeeds",
			source:                              "source:image",
			target:                              "target:image",
			arch:                                []string{"amd64", "arm64"},
			srcTLSVerify:                        true,
			mockTryDirectMultiArchSync:          func(source, target string, srcTLSVerify bool) error { return nil },
			mockSyncArchImagesAndCreateManifest: func(source, target string, arch []string, srcTLSVerify bool) error { return nil },
			expectError:                         false,
		},
		{
			name:                                "fallback to arch images sync",
			source:                              "source:image",
			target:                              "target:image",
			arch:                                []string{"amd64", "arm64"},
			srcTLSVerify:                        true,
			mockTryDirectMultiArchSync:          func(source, target string, srcTLSVerify bool) error { return fmt.Errorf("direct sync fails") },
			mockSyncArchImagesAndCreateManifest: func(source, target string, arch []string, srcTLSVerify bool) error { return nil },
			expectError:                         false,
		},
		{
			name:                       "both methods fail",
			source:                     "source:image",
			target:                     "target:image",
			arch:                       []string{"amd64", "arm64"},
			srcTLSVerify:               true,
			mockTryDirectMultiArchSync: func(source, target string, srcTLSVerify bool) error { return fmt.Errorf("direct sync fails") },
			mockSyncArchImagesAndCreateManifest: func(source, target string, arch []string, srcTLSVerify bool) error {
				return fmt.Errorf("arch sync fails")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(tryDirectMultiArchSync, tt.mockTryDirectMultiArchSync)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncArchImagesAndCreateManifest, tt.mockSyncArchImagesAndCreateManifest)
			defer patches.Reset()

			err := syncMultiArchImage(tt.source, tt.target, tt.arch, tt.srcTLSVerify)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTryDirectMultiArchSync(t *testing.T) {
	tests := []struct {
		name                     string
		source                   string
		target                   string
		srcTLSVerify             bool
		mockIsMultiArchManifests func(bool, string) bool
		mockCopyRegistry         func(reg.Options) error
		expectError              bool
	}{
		{
			name:                     "multi-arch image sync succeeds",
			source:                   "source:image",
			target:                   "target:image",
			srcTLSVerify:             true,
			mockIsMultiArchManifests: func(srcTLSVerify bool, imageAddress string) bool { return true },
			mockCopyRegistry: func(opts reg.Options) error {
				assert.Equal(t, true, opts.MultiArch)
				return nil
			},
			expectError: false,
		},
		{
			name:                     "not multi-arch image",
			source:                   "source:image",
			target:                   "target:image",
			srcTLSVerify:             true,
			mockIsMultiArchManifests: func(srcTLSVerify bool, imageAddress string) bool { return false },
			expectError:              true,
		},
		{
			name:                     "copy registry fails",
			source:                   "source:image",
			target:                   "target:image",
			srcTLSVerify:             true,
			mockIsMultiArchManifests: func(srcTLSVerify bool, imageAddress string) bool { return true },
			mockCopyRegistry:         func(opts reg.Options) error { return fmt.Errorf("copy error") },
			expectError:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(reg.IsMultiArchManifests, tt.mockIsMultiArchManifests)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(reg.CopyRegistry, tt.mockCopyRegistry)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := tryDirectMultiArchSync(tt.source, tt.target, tt.srcTLSVerify)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncArchImagesAndCreateManifest(t *testing.T) {
	tests := []struct {
		name                      string
		source                    string
		target                    string
		arch                      []string
		srcTLSVerify              bool
		mockSyncSingleArchVariant func(string, string, string, reg.Options) (reg.ImageArch, error)
		mockCreateMultiArchImage  func([]reg.ImageArch, string) error
		expectError               bool
	}{
		{
			name:         "successful sync and manifest creation",
			source:       "source:image",
			target:       "target:image",
			arch:         []string{"amd64", "arm64"},
			srcTLSVerify: true,
			mockSyncSingleArchVariant: func(source, target, arch string, op reg.Options) (reg.ImageArch, error) {
				return reg.ImageArch{
					Name:         target + "-" + arch,
					OS:           "linux",
					Architecture: arch,
				}, nil
			},
			mockCreateMultiArchImage: func(img []reg.ImageArch, target string) error { return nil },
			expectError:              false,
		},
		{
			name:         "sync single arch variant fails",
			source:       "source:image",
			target:       "target:image",
			arch:         []string{"amd64"},
			srcTLSVerify: true,
			mockSyncSingleArchVariant: func(source, target, arch string, op reg.Options) (reg.ImageArch, error) {
				return reg.ImageArch{}, fmt.Errorf("sync error")
			},
			expectError: true,
		},
		{
			name:         "create multi-arch image fails",
			source:       "source:image",
			target:       "target:image",
			arch:         []string{"amd64"},
			srcTLSVerify: true,
			mockSyncSingleArchVariant: func(source, target, arch string, op reg.Options) (reg.ImageArch, error) {
				return reg.ImageArch{
					Name:         target + "-" + arch,
					OS:           "linux",
					Architecture: arch,
				}, nil
			},
			mockCreateMultiArchImage: func(img []reg.ImageArch, target string) error { return fmt.Errorf("create error") },
			expectError:              true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(syncSingleArchVariant, tt.mockSyncSingleArchVariant)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(reg.CreateMultiArchImage, tt.mockCreateMultiArchImage)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := syncArchImagesAndCreateManifest(tt.source, tt.target, tt.arch, tt.srcTLSVerify)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSyncSingleArchVariant(t *testing.T) {
	op := reg.Options{
		MultiArch:     false,
		SrcTLSVerify:  true,
		DestTLSVerify: false,
	}

	tests := []struct {
		name             string
		source           string
		target           string
		arch             string
		mockCopyRegistry func(reg.Options) error
		expectError      bool
	}{
		{
			name:   "successful sync (no cut, use original source with arch choice)",
			source: "source:image",
			target: "target:image",
			arch:   "amd64",
			mockCopyRegistry: func(opts reg.Options) error {
				// Verify the options are set correctly
				assert.Equal(t, false, opts.MultiArch)
				assert.Equal(t, true, opts.SrcTLSVerify)
				assert.Equal(t, false, opts.DestTLSVerify)
				assert.Equal(t, "amd64", opts.Arch)
				assert.Equal(t, "target:image-amd64", opts.Target)
				// If source has no cut marker, business logic uses original source
				// and relies on SystemContext ArchitectureChoice (opts.Arch) to pick the arch.
				assert.Equal(t, "source:image", opts.Source)
				return nil
			},
			expectError: false,
		},
		{
			name:   "successful sync (has cut, replace cut with -arch-)",
			source: "source:image-*-202112111112",
			target: "target:image",
			arch:   "arm64",
			mockCopyRegistry: func(opts reg.Options) error {
				assert.Equal(t, false, opts.MultiArch)
				assert.Equal(t, true, opts.SrcTLSVerify)
				assert.Equal(t, false, opts.DestTLSVerify)
				assert.Equal(t, "arm64", opts.Arch)
				assert.Equal(t, "target:image-arm64", opts.Target)
				assert.Equal(t, "source:image-arm64-202112111112", opts.Source)
				return nil
			},
			expectError: false,
		},
		{
			name:   "copy registry fails",
			source: "source:image",
			target: "target:image",
			arch:   "amd64",
			mockCopyRegistry: func(opts reg.Options) error {
				return fmt.Errorf("copy error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(reg.CopyRegistry, tt.mockCopyRegistry)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			img, err := syncSingleArchVariant(tt.source, tt.target, tt.arch, op)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, reg.ImageArch{}, img)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "target:image-"+tt.arch, img.Name)
				assert.Equal(t, "linux", img.OS)
				assert.Equal(t, tt.arch, img.Architecture)
			}
		})
	}
}
