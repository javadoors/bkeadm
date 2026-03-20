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

	"gopkg.in/yaml.v3"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type BuildConfig struct {
	Registry          registry `yaml:"registry"`
	OpenFuyaoVersion  string   `yaml:"openFuyaoVersion"`
	KubernetesVersion string   `yaml:"kubernetesVersion"`
	EtcdVersion       string   `yaml:"etcdVersion"`
	ContainerdVersion string   `yaml:"containerdVersion"`
	Repos             []Repo   `yaml:"repos"`
	Rpms              []Rpm    `yaml:"rpms"`
	Debs              []Deb    `yaml:"debs"`
	Files             []File   `yaml:"files"`
	Patches           []File   `yaml:"patches"`
	Charts            []File   `yaml:"charts"`
}

// The image files of the image repository need to be packaged into bke
type registry struct {
	ImageAddress string   `yaml:"imageAddress"`
	Architecture []string `yaml:"architecture"`
}

// Repo All defined images are collected into the image repository
type Repo struct {
	Architecture []string   `yaml:"architecture"`
	NeedDownload bool       `yaml:"needDownload"`
	IsKubernetes bool       `yaml:"isKubernetes"`
	SubImages    []SubImage `yaml:"subImages"`
}

type SubImage struct {
	SourceRepo string `yaml:"sourceRepo"`
	TargetRepo string `yaml:"targetRepo"`
	// The following types of mirrors exist，each build needs to get the most recent commit
	// ex: alpine:v4.0-amd64-202502051112
	// so the field will be the url of a mirrored repository, as shown below
	// dockerhub
	// nexus@http://username:password@repository.nexus.com/
	// harbor@http://username:password@repository.harbor.com/
	// registry@http://repository.registry.com/
	ImageTrack string  `yaml:"imageTrack"`
	Images     []Image `yaml:"images"`
}

type Image struct {
	Name        string    `json:"name" yaml:"name"`
	UsedPodInfo []PodInfo `json:"usedPodInfo" yaml:"usedPodInfo"`
	Tag         []string  `json:"tag" yaml:"tag"`
}

type PodInfo struct {
	PodPrefix string `json:"podPrefix" yaml:"podPrefix"`
	NameSpace string `json:"namespace" yaml:"namespace"`
}

// Rpm Define dependent packages, currently only supported yum sources
// Splicing rule as follows the system/systemVersion/systemArchitecture/directory
// example
/*
Centos/7/amd64/docker-ce/*
*/
// This is the directory level and will collect all the files in the directory
type Rpm struct {
	Address            string   `yaml:"address"`
	System             []string `yaml:"system"`
	SystemVersion      []string `yaml:"systemVersion"`
	SystemArchitecture []string `yaml:"systemArchitecture"`
	Directory          []string `yaml:"directory"`
}

// Deb Define dependent packages, currently only supported yum sources
// Splicing rule as follows the system/systemVersion/systemArchitecture/directory
// example
/*
Centos/7/amd64/docker-ce/*
*/
// This is the directory level and will collect all the files in the directory
type Deb struct {
	Address            string   `yaml:"address"`
	System             []string `yaml:"system"`
	SystemVersion      []string `yaml:"systemVersion"`
	SystemArchitecture []string `yaml:"systemArchitecture"`
	Directory          []string `yaml:"directory"`
}

// File Just download the file and collect it in the specified directory
// Download the address/files[0] file, equivalent to `curl http://xxxx/files[0] -o xxx/xx/files/files[0]`
type File struct {
	Address string     `yaml:"address"`
	Files   []FileInfo `yaml:"files"`
}

type FileInfo struct {
	FileName  string `yaml:"fileName"`
	FileAlias string `yaml:"fileAlias"`
}

func (o *Options) Config() {
	cfg := o.buildConfig()
	b, err := yaml.Marshal(cfg)
	if err != nil {
		log.Errorf("Description Failed to parse the configuration file %v", err)
		return
	}

	curPath, err := os.Getwd()
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Failed to get current working directory: %v", err))
	}
	err = os.WriteFile(path.Join(curPath, "build.yaml"), b, utils.DefaultFilePermission)
	if err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Failed to write the configuration file. Procedure %v", err))
		return
	}
	fmt.Println("The sample file is generated in the current directory and is called build.yaml")
}

func (o *Options) buildConfig() BuildConfig {
	return BuildConfig{
		Registry: registry{
			ImageAddress: fmt.Sprintf("registry.bocloud.com/kubernetes/%s", utils.DefaultLocalImageRegistry),
			Architecture: []string{"amd64"},
		},
		Repos: []Repo{
			{
				Architecture: []string{"amd64"},
				NeedDownload: true,
				SubImages: []SubImage{
					{
						SourceRepo: "registry.bocloud.com/kubernetes",
						TargetRepo: "kubernetes",
						Images: []Image{
							{Name: "registry", Tag: []string{"2.8.1"}},
							{Name: "nginx_yum_repo", Tag: []string{"1.23.0-alpine"}},
							{Name: "chartmuseum", Tag: []string{"0.15.0"}},
							{Name: "nfs-server-alpine", Tag: []string{"0.9.0"}},
							{Name: "k3s", Tag: []string{"v1.25.7-k3s1"}},
							{Name: "mirrored-pause", Tag: []string{"3.6"}},
						},
					},
				},
			},
		},
		Rpms: []Rpm{
			{
				Address:            "http://127.0.0.1:40080/",
				System:             []string{"CentOS"},
				SystemVersion:      []string{"7"},
				SystemArchitecture: []string{"amd64"},
				Directory:          []string{"docker-ce"},
			},
		},
		Files: []File{
			{
				Address: "http://127.0.0.1:40080/files/",
				Files: []FileInfo{
					{"bkeadm_linux_amd64", ""},
					{"containerd-1.7.14-linux-amd64.tar.gz", ""},
					{"charts.tar.gz", ""},
					{"nfsshare.tar.gz", ""},
					{"kubectl-v1.23.17-amd64", ""},
					{"cni-plugins-linux-amd64-v1.2.0.tgz", ""},
				},
			},
		},
	}
}
