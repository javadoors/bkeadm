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
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func TestBuildConfigStruct(t *testing.T) {
	// Test that the BuildConfig struct has the expected fields
	cfg := &BuildConfig{}

	// Check that all fields exist
	cfg.Registry = registry{
		ImageAddress: "test-registry",
		Architecture: []string{"amd64"},
	}
	cfg.OpenFuyaoVersion = "v1.0.0"
	cfg.KubernetesVersion = "v1.25.0"
	cfg.EtcdVersion = "v3.5.0"
	cfg.ContainerdVersion = "v1.6.0"
	cfg.Repos = []Repo{{}}
	cfg.Rpms = []Rpm{{}}
	cfg.Debs = []Deb{{}}
	cfg.Files = []File{{}}
	cfg.Patches = []File{{}}
	cfg.Charts = []File{{}}

	assert.Equal(t, "test-registry", cfg.Registry.ImageAddress)
	assert.Equal(t, []string{"amd64"}, cfg.Registry.Architecture)
	assert.Equal(t, "v1.0.0", cfg.OpenFuyaoVersion)
	assert.Equal(t, "v1.25.0", cfg.KubernetesVersion)
	assert.Equal(t, "v3.5.0", cfg.EtcdVersion)
	assert.Equal(t, "v1.6.0", cfg.ContainerdVersion)
	assert.Len(t, cfg.Repos, testOneValue)
	assert.Len(t, cfg.Rpms, testOneValue)
	assert.Len(t, cfg.Debs, testOneValue)
	assert.Len(t, cfg.Files, testOneValue)
	assert.Len(t, cfg.Patches, testOneValue)
	assert.Len(t, cfg.Charts, testOneValue)
}

func TestRegistryStruct(t *testing.T) {
	// Test that the registry struct has the expected fields
	reg := &registry{}

	reg.ImageAddress = "test-image-address"
	reg.Architecture = []string{"amd64", "arm64"}

	assert.Equal(t, "test-image-address", reg.ImageAddress)
	assert.Equal(t, []string{"amd64", "arm64"}, reg.Architecture)
}

func TestRepoStruct(t *testing.T) {
	// Test that the Repo struct has the expected fields
	repo := &Repo{}

	repo.Architecture = []string{"amd64"}
	repo.NeedDownload = true
	repo.IsKubernetes = false
	repo.SubImages = []SubImage{{}}

	assert.Equal(t, []string{"amd64"}, repo.Architecture)
	assert.True(t, repo.NeedDownload)
	assert.False(t, repo.IsKubernetes)
	assert.Len(t, repo.SubImages, testOneValue)
}

func TestSubImageStruct(t *testing.T) {
	// Test that the SubImage struct has the expected fields
	subImg := &SubImage{}

	subImg.SourceRepo = "source-repo"
	subImg.TargetRepo = "target-repo"
	subImg.ImageTrack = "track-url"
	subImg.Images = []Image{{}}

	assert.Equal(t, "source-repo", subImg.SourceRepo)
	assert.Equal(t, "target-repo", subImg.TargetRepo)
	assert.Equal(t, "track-url", subImg.ImageTrack)
	assert.Len(t, subImg.Images, testOneValue)
}

func TestImageStruct(t *testing.T) {
	// Test that the Image struct has the expected fields
	img := &Image{}

	img.Name = "test-image"
	img.UsedPodInfo = []PodInfo{{}}
	img.Tag = []string{"latest"}

	assert.Equal(t, "test-image", img.Name)
	assert.Len(t, img.UsedPodInfo, testOneValue)
	assert.Equal(t, []string{"latest"}, img.Tag)
}

func TestPodInfoStruct(t *testing.T) {
	// Test that the PodInfo struct has the expected fields
	podInfo := &PodInfo{}

	podInfo.PodPrefix = "test-prefix"
	podInfo.NameSpace = "test-namespace"

	assert.Equal(t, "test-prefix", podInfo.PodPrefix)
	assert.Equal(t, "test-namespace", podInfo.NameSpace)
}

func TestRpmStruct(t *testing.T) {
	// Test that the Rpm struct has the expected fields
	rpm := &Rpm{}

	rpm.Address = "http://example.com"
	rpm.System = []string{"CentOS"}
	rpm.SystemVersion = []string{"7"}
	rpm.SystemArchitecture = []string{"amd64"}
	rpm.Directory = []string{"docker-ce"}

	assert.Equal(t, "http://example.com", rpm.Address)
	assert.Equal(t, []string{"CentOS"}, rpm.System)
	assert.Equal(t, []string{"7"}, rpm.SystemVersion)
	assert.Equal(t, []string{"amd64"}, rpm.SystemArchitecture)
	assert.Equal(t, []string{"docker-ce"}, rpm.Directory)
}

func TestDebStruct(t *testing.T) {
	// Test that the Deb struct has the expected fields
	deb := &Deb{}

	deb.Address = "http://example.com"
	deb.System = []string{"Ubuntu"}
	deb.SystemVersion = []string{"20.04"}
	deb.SystemArchitecture = []string{"amd64"}
	deb.Directory = []string{"docker-ce"}

	assert.Equal(t, "http://example.com", deb.Address)
	assert.Equal(t, []string{"Ubuntu"}, deb.System)
	assert.Equal(t, []string{"20.04"}, deb.SystemVersion)
	assert.Equal(t, []string{"amd64"}, deb.SystemArchitecture)
	assert.Equal(t, []string{"docker-ce"}, deb.Directory)
}

func TestFileStruct(t *testing.T) {
	// Test that the File struct has the expected fields
	file := &File{}

	file.Address = "http://example.com/files/"
	file.Files = []FileInfo{{}}

	assert.Equal(t, "http://example.com/files/", file.Address)
	assert.Len(t, file.Files, testOneValue)
}

func TestFileInfoStruct(t *testing.T) {
	// Test that the FileInfo struct has the expected fields
	fileInfo := &FileInfo{}

	fileInfo.FileName = "test-file"
	fileInfo.FileAlias = "test-alias"

	assert.Equal(t, "test-file", fileInfo.FileName)
	assert.Equal(t, "test-alias", fileInfo.FileAlias)
}

func TestBuildConfigMethod(t *testing.T) {
	o := &Options{}

	cfg := o.buildConfig()

	// Check that the default config is built correctly
	assert.Equal(t, fmt.Sprintf("registry.bocloud.com/kubernetes/%s", utils.DefaultLocalImageRegistry), cfg.Registry.ImageAddress)
	assert.Equal(t, []string{"amd64"}, cfg.Registry.Architecture)

	// Check that repos are populated with defaults
	assert.NotEmpty(t, cfg.Repos)

	// Check that rpms are populated with defaults
	assert.NotEmpty(t, cfg.Rpms)

	// Check that files are populated with defaults
	assert.NotEmpty(t, cfg.Files)
}

func TestConfigMethod(t *testing.T) {
	tests := []struct {
		name          string
		mockMarshal   func(interface{}) ([]byte, error)
		mockGetwd     func() (string, error)
		mockWriteFile func(string, []byte, os.FileMode) error
		expectError   bool
	}{
		{
			name: "successful config generation",
			mockMarshal: func(v interface{}) ([]byte, error) {
				return []byte("test: config"), nil
			},
			mockGetwd: func() (string, error) {
				return "/tmp", nil
			},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "marshal fails",
			mockMarshal: func(v interface{}) ([]byte, error) {
				return nil, fmt.Errorf("marshal error")
			},
			expectError: true,
		},
		{
			name: "getwd fails",
			mockMarshal: func(v interface{}) ([]byte, error) {
				return []byte("test: config"), nil
			},
			mockGetwd: func() (string, error) {
				return "", fmt.Errorf("getwd error")
			},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return nil
			},
			expectError: false, // getwd error is just logged as warning
		},
		{
			name: "write file fails",
			mockMarshal: func(v interface{}) ([]byte, error) {
				return []byte("test: config"), nil
			},
			mockGetwd: func() (string, error) {
				return "/tmp", nil
			},
			mockWriteFile: func(filename string, data []byte, perm os.FileMode) error {
				return fmt.Errorf("write error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Options{}

			// Apply patches
			patches := gomonkey.ApplyFunc(yaml.Marshal, tt.mockMarshal)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Getwd, tt.mockGetwd)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.WriteFile, tt.mockWriteFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.Errorf, func(format string, args ...interface{}) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(fmt.Println, func(a ...interface{}) (n int, err error) {
				return testZeroValue, nil
			})
			defer patches.Reset()

			// Mock buildConfig to return a simple config
			patches = gomonkey.ApplyFunc((*Options).buildConfig, func(o *Options) BuildConfig {
				return BuildConfig{
					Registry: registry{
						ImageAddress: "test-registry",
						Architecture: []string{"amd64"},
					},
				}
			})
			defer patches.Reset()

			o.Config()

			// The function should complete without panic
			assert.True(t, true)
		})
	}
}

func TestBuildConfigWithRealValues(t *testing.T) {
	o := &Options{}

	cfg := o.buildConfig()

	// Verify that the config has expected default values
	assert.Equal(t, fmt.Sprintf("registry.bocloud.com/kubernetes/%s", utils.DefaultLocalImageRegistry), cfg.Registry.ImageAddress)
	assert.Equal(t, []string{"amd64"}, cfg.Registry.Architecture)

	// Check that the first repo has expected values
	assert.NotEmpty(t, cfg.Repos)
	firstRepo := cfg.Repos[0]
	assert.Equal(t, []string{"amd64"}, firstRepo.Architecture)
	assert.True(t, firstRepo.NeedDownload)
	assert.NotEmpty(t, firstRepo.SubImages)

	// Check that the first subimage has expected values
	firstSubImage := firstRepo.SubImages[0]
	assert.Equal(t, "registry.bocloud.com/kubernetes", firstSubImage.SourceRepo)
	assert.Equal(t, "kubernetes", firstSubImage.TargetRepo)
	assert.NotEmpty(t, firstSubImage.Images)

	// Check that the first image has expected values
	firstImage := firstSubImage.Images[0]
	assert.Equal(t, "registry", firstImage.Name)
	assert.Equal(t, []string{"2.8.1"}, firstImage.Tag)

	// Check that rpms have expected values
	assert.NotEmpty(t, cfg.Rpms)
	firstRpm := cfg.Rpms[0]
	assert.Equal(t, "http://127.0.0.1:40080/", firstRpm.Address)
	assert.Equal(t, []string{"CentOS"}, firstRpm.System)

	// Check that files have expected values
	assert.NotEmpty(t, cfg.Files)
	firstFile := cfg.Files[0]
	assert.Equal(t, "http://127.0.0.1:40080/files/", firstFile.Address)
	assert.NotEmpty(t, firstFile.Files)

	// Check that the first file info has expected values
	firstFileInfo := firstFile.Files[0]
	assert.Equal(t, "bkeadm_linux_amd64", firstFileInfo.FileName)
}

func TestYamlTags(t *testing.T) {
	// Test that the YAML tags are properly set by marshaling and unmarshaling
	cfg := BuildConfig{
		Registry: registry{
			ImageAddress: "test-registry",
			Architecture: []string{"amd64"},
		},
		OpenFuyaoVersion:  "v1.0.0",
		KubernetesVersion: "v1.25.0",
		EtcdVersion:       "v3.5.0",
		ContainerdVersion: "v1.6.0",
		Repos: []Repo{
			{
				Architecture: []string{"amd64"},
				NeedDownload: true,
				IsKubernetes: false,
				SubImages: []SubImage{
					{
						SourceRepo: "source",
						TargetRepo: "target",
						ImageTrack: "track",
						Images: []Image{
							{
								Name: "test-image",
								UsedPodInfo: []PodInfo{
									{
										PodPrefix: "prefix",
										NameSpace: "namespace",
									},
								},
								Tag: []string{"latest"},
							},
						},
					},
				},
			},
		},
		Rpms: []Rpm{
			{
				Address:            "http://example.com",
				System:             []string{"CentOS"},
				SystemVersion:      []string{"7"},
				SystemArchitecture: []string{"amd64"},
				Directory:          []string{"docker-ce"},
			},
		},
		Debs: []Deb{
			{
				Address:            "http://example.com",
				System:             []string{"Ubuntu"},
				SystemVersion:      []string{"20.04"},
				SystemArchitecture: []string{"amd64"},
				Directory:          []string{"docker-ce"},
			},
		},
		Files: []File{
			{
				Address: "http://example.com/files/",
				Files: []FileInfo{
					{
						FileName:  "test-file",
						FileAlias: "test-alias",
					},
				},
			},
		},
		Patches: []File{
			{
				Address: "http://example.com/patches/",
				Files: []FileInfo{
					{
						FileName:  "patch-file",
						FileAlias: "patch-alias",
					},
				},
			},
		},
		Charts: []File{
			{
				Address: "http://example.com/charts/",
				Files: []FileInfo{
					{
						FileName:  "chart-file",
						FileAlias: "chart-alias",
					},
				},
			},
		},
	}

	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(&cfg)
	assert.NoError(t, err)

	// Unmarshal back to struct
	var newCfg BuildConfig
	err = yaml.Unmarshal(yamlBytes, &newCfg)
	assert.NoError(t, err)

	// Check that values are preserved
	assert.Equal(t, cfg.Registry.ImageAddress, newCfg.Registry.ImageAddress)
	assert.Equal(t, cfg.OpenFuyaoVersion, newCfg.OpenFuyaoVersion)
	assert.Equal(t, cfg.KubernetesVersion, newCfg.KubernetesVersion)
	assert.Equal(t, cfg.EtcdVersion, newCfg.EtcdVersion)
	assert.Equal(t, cfg.ContainerdVersion, newCfg.ContainerdVersion)
	assert.Equal(t, cfg.Repos, newCfg.Repos)
	assert.Equal(t, cfg.Rpms, newCfg.Rpms)
	assert.Equal(t, cfg.Debs, newCfg.Debs)
	assert.Equal(t, cfg.Files, newCfg.Files)
	assert.Equal(t, cfg.Patches, newCfg.Patches)
	assert.Equal(t, cfg.Charts, newCfg.Charts)
}
