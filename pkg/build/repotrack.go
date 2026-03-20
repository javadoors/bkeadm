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
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	url2 "net/url"
	"sort"
	"strings"

	commonutils "gopkg.openfuyao.cn/cluster-api-provider-bke/common/utils"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

const (
	Nexus     = "nexus"
	Harbor    = "harbor"
	DockerHub = "dockerhub"
	Registry  = "registry"

	// urlSplitMinParts 表示 URL 按 "//" 或 ":" 拆分后的最小部分数
	urlSplitMinParts = 2
	// urlSplitThreeParts 表示 URL 按 "@" 拆分后的三部分
	urlSplitThreeParts = 3
	// tagSplitTwoParts 表示 tag 按 "-" 拆分后的两部分
	tagSplitTwoParts = 2
	// tagSplitThreeParts 表示 tag 按 "-" 拆分后的三部分
	tagSplitThreeParts = 3
	// tagThirdElementIndex 表示 tag 拆分后第三个元素的索引
	tagThirdElementIndex = 2
)

type imageInfo struct {
	Name         string   `json:"image_name"`
	Tag          []string `json:"image_tag"`
	Architecture []string `json:"architecture"`
}

// dockerHubTagList Response Tags
type dockerHubTagList struct {
	Count    int    `json:"count"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	Results  []*struct {
		Name     string `json:"name"`
		FullSize int    `json:"full_size"`
		Images   []*struct {
			Size         int    `json:"size"`
			Digest       string `json:"digest"`
			Architecture string `json:"architecture"`
			Os           string `json:"os"`
			Variant      string `json:"variant"`
		} `json:"images"`
		Id                  int    `json:"id"`
		Repository          int    `json:"repository"`
		Creator             int    `json:"creator"`
		LastUpdater         int    `json:"last_updater"`
		LastUpdaterUserName string `json:"last_updater_user_name"`
		V2                  bool   `json:"v2"`
		LastUpdated         string `json:"last_updated"`
	} `json:"results"`
}

type nexusTagList struct {
	Items []*struct {
		Id         string `json:"id"`
		Repository string `json:"repository"`
		Format     string `json:"format"`
		Group      string `json:"group"`
		Name       string `json:"name"`
		Version    string `json:"version"`
		Assets     []*struct {
			DownloadUrl string `json:"downloadUrl"`
			Path        string `json:"path"`
			Id          string `json:"id"`
			Repository  string `json:"repository"`
			Format      string `json:"format"`
			Checksum    struct {
				Sha1   string `json:"sha1"`
				Sha256 string `json:"sha256"`
			} `json:"checksum"`
		} `json:"assets"`
	} `json:"items"`
}

type harborTag struct {
	Id         uint   `json:"id"`
	Digest     string `json:"digest"`
	ExtraAttrs struct {
		Architecture string `json:"architecture"`
		OS           string `json:"os"`
	} `json:"extra_attrs"`
	References []struct {
		Platform struct {
			Architecture string `json:"architecture"`
			OS           string `json:"os"`
		}
	} `json:"references"`
	Tags []struct {
		Name string `json:"name"`
	} `json:"tags"`
}

type registryTag struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

func imageTrack(sourceRepo, imageTrack, imageName, imageTag string, arch []string) (string, error) {
	var err error
	source := ""
	if len(imageTrack) == 0 || strings.Contains(imageTag, cut) {
		source = fmt.Sprintf("%s/%s:%s", sourceRepo, imageName, imageTag)
		source = strings.ReplaceAll(source, "//", "/")
		return source, nil
	}

	repo, url := splitRepo1(imageTrack)
	newUrl, projectName := splitRepo2(url)

	var imageTagList []*imageInfo
	switch repo {
	case DockerHub:
		imageTagList, err = dockerHubTags(imageName)
		if err != nil {
			return source, err
		}
	case Nexus:
		imageTagList, err = nexusTags(newUrl, imageName)
		if err != nil {
			return source, err
		}
	case Harbor:
		if projectName == "" {
			return "", errors.New("Project name cannot be empty ")
		}
		imageTagList, err = harborTags(newUrl, projectName, imageName)
		if err != nil {
			return source, err
		}
	case Registry:
		if projectName == "" {
			return "", errors.New("Project name cannot be empty ")
		}
		imageTagList, err = registryTags(newUrl, projectName, imageName)
		if err != nil {
			return source, err
		}
	default:
		return "", errors.New(fmt.Sprintf("unsupported warehouse type %s", repo))
	}

	latestTag, err := findLatestTag(imageTagList, imageTag, arch)
	if err != nil {
		return source, err
	}
	source = fmt.Sprintf("%s/%s:%s", sourceRepo, imageName, latestTag)
	source = strings.ReplaceAll(source, "//", "/")
	return source, nil
}

func dockerHubTags(imageName string) ([]*imageInfo, error) {
	url := fmt.Sprintf("https://registry.hub.docker.com/v2/repositories/library/%s/tags?page_size=100&&page=1", imageName)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	log.Debugf(url)
	log.Debugf(string(body))

	var tags dockerHubTagList

	if err = json.Unmarshal(body, &tags); err != nil {
		return nil, errors.New("image not found")
	}
	var it []*imageInfo
	for _, t := range tags.Results {
		it1 := imageInfo{Name: imageName}
		it1.Tag = []string{t.Name}
		for _, t1 := range t.Images {
			it1.Architecture = append(it1.Architecture, t1.Architecture)
		}
		it = append(it, &it1)
	}
	return it, nil
}

func nexusTags(url, imageName string) ([]*imageInfo, error) {
	userName, password, url := splitRepo3(url)
	tagUrl := fmt.Sprintf("%s/service/rest/v1/search?docker.imageName=%s", url, imageName)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}

	cookie := ""
	if userName != "" && password != "" {
		loginUrl := fmt.Sprintf("%s/service/rapture/session", url)

		data := make(url2.Values)
		data["username"] = []string{base64.StdEncoding.EncodeToString([]byte(userName))}
		data["password"] = []string{base64.StdEncoding.EncodeToString([]byte(password))}
		resp1, err := http.PostForm(loginUrl, data)
		if err != nil {
			return nil, err
		}
		defer resp1.Body.Close()
		cookie = resp1.Header.Get("Set-Cookie")
	}

	req, err := http.NewRequest("GET", tagUrl, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Cookie", cookie)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	log.Debugf(url)
	log.Debugf(string(body))

	var tags nexusTagList

	if err = json.Unmarshal(body, &tags); err != nil {
		return nil, err
	}
	var it []*imageInfo
	for _, t := range tags.Items {
		it = append(it, &imageInfo{
			Name:         imageName,
			Tag:          []string{t.Version},
			Architecture: []string{},
		})
	}
	return it, nil
}

func harborTags(url, projectName, imageName string) ([]*imageInfo, error) {
	userName, password, url := splitRepo3(url)
	tagUrl := fmt.Sprintf("%s/api/v2.0/projects/%s/repositories/%s/artifacts?page=1&page_size=100&with_tag=true&"+
		"with_label=false&with_scan_overview=false&with_signature=false&with_immutable_status=false",
		url, projectName, imageName)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", tagUrl, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(userName, password)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	log.Debugf(url)
	log.Debugf(string(body))

	var tags []*harborTag

	if err = json.Unmarshal(body, &tags); err != nil {
		return nil, err
	}
	var it []*imageInfo
	for _, tg := range tags {
		it1 := imageInfo{Name: imageName}
		for _, t1 := range tg.Tags {
			it1.Tag = append(it1.Tag, t1.Name)
		}
		for _, t1 := range tg.References {
			it1.Architecture = append(it1.Architecture, t1.Platform.Architecture)
		}
		if len(it1.Architecture) == 0 {
			it1.Architecture = append(it1.Architecture, tg.ExtraAttrs.Architecture)
		}
		if it1.Tag == nil {
			continue
		}
		it = append(it, &it1)
	}
	return it, nil
}

func registryTags(url, projectName, imageName string) ([]*imageInfo, error) {
	userName, password, url := splitRepo3(url)
	tagUrl := fmt.Sprintf("%s/v2/%s/%s/tags/list", url, projectName, imageName)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest("GET", tagUrl, nil)
	if err != nil {
		return nil, err
	}
	if len(userName) > 0 {
		req.SetBasicAuth(userName, password)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	log.Debugf(tagUrl)
	log.Debugf(string(body))

	var img registryTag

	if err = json.Unmarshal(body, &img); err != nil {
		return nil, err
	}
	if len(img.Tags) == 0 {
		return nil, errors.New(fmt.Sprintf("%s has no tags", imageName))
	}

	var it []*imageInfo
	for _, tg := range img.Tags {
		it1 := imageInfo{Name: imageName}
		it1.Tag = append(it1.Tag, tg)
		it = append(it, &it1)
	}
	return it, nil
}

func splitRepo1(compoundAddress string) (string, string) {
	repos := strings.Split(compoundAddress, "@")
	url := ""
	if len(repos) == urlSplitMinParts {
		url = repos[1]
	}
	if len(repos) == urlSplitThreeParts {
		url = repos[1] + "@" + repos[2]
	}
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}
	return repos[0], url
}

func splitRepo2(compoundAddress string) (string, string) {
	url := compoundAddress
	projectName := ""
	u1 := strings.Split(compoundAddress, "//")
	if len(u1) == urlSplitMinParts {
		u2 := strings.Split(u1[1], "/")
		if len(u2) > 1 {
			projectName = strings.Join(u2[1:len(u2)], "/")
			if strings.HasSuffix(projectName, "/") {
				projectName = projectName[0 : len(projectName)-1]
			}
			url = u1[0] + "//" + u2[0]
		}
	}
	return url, projectName
}

func splitRepo3(url string) (string, string, string) {
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}
	if !strings.Contains(url, "@") {
		return "", "", url
	} else {
		a := strings.Split(url, "@")
		if len(a) < urlSplitMinParts {
			return "", "", url
		}
		b := strings.Split(a[0], "//")
		if len(b) < urlSplitMinParts {
			return "", "", url
		}
		c := strings.Split(b[1], ":")
		if len(c) < urlSplitMinParts {
			return "", "", url
		}
		return c[0], c[1], b[0] + "//" + a[1]
	}
}

type tagParseContext struct {
	tagPrefix  string
	arch       []string
	tagMap     map[string]string
	tagList    *[]string
	defaultTag *string
}

// parseTagFormat 解析标签格式并返回处理后的标签信息
func parseTagFormat(tag string, ctx tagParseContext) {
	if !strings.HasPrefix(tag, ctx.tagPrefix) {
		return
	}
	if ctx.tagMap == nil {
		return
	}
	tag1 := strings.Split(tag, "-")
	tag1Len := len(tag1)
	switch tag1Len {
	case 1:
		*ctx.defaultTag = tag
	case tagSplitTwoParts:
		if tag1Len >= tagSplitTwoParts && len(tag1) > 1 {
			ctx.tagMap[tag1[1]] = tag1[0]
			*ctx.tagList = append(*ctx.tagList, tag1[1])
		}
	case tagSplitThreeParts:
		if tag1Len >= tagSplitThreeParts && len(tag1) > tagThirdElementIndex && utils.ContainsString(ctx.arch, tag1[1]) {
			*ctx.tagList = append(*ctx.tagList, tag1[tagThirdElementIndex])
			ctx.tagMap[tag1[tagThirdElementIndex]] = tag1[0] + cut
		}
	default:
		fmt.Println(fmt.Sprintf("unexpected tag format %s", tag))
	}
}

// findLatestTag example busybox
/*
busybox:v2.1
busybox:v2.1-202212242122
busybox:v2.1-amd64-202112242132
busybox:v2.1-arm64-202212242132
busybox:v2.1-arm64-202312242132
*/
func findLatestTag(imageTagList []*imageInfo, tagPrefix string, arch []string) (string, error) {
	if len(imageTagList) == 0 {
		return "", errors.New(fmt.Sprintf("tag %s not found. ", tagPrefix))
	}

	defaultTag := ""
	var tagList []string
	tagMap := map[string]string{}

	ctx := tagParseContext{
		tagPrefix:  tagPrefix,
		arch:       arch,
		tagMap:     tagMap,
		tagList:    &tagList,
		defaultTag: &defaultTag,
	}
	for _, image := range imageTagList {
		if len(image.Architecture) != 0 && !commonutils.SliceContainsSlice(image.Architecture, arch) {
			continue
		}
		for _, tag := range image.Tag {
			parseTagFormat(tag, ctx)
		}
	}

	if len(tagList) == 0 {
		if defaultTag != "" {
			return defaultTag, nil
		}
		return "", errors.New(fmt.Sprintf("%s %s tags %s not found. ", imageTagList[0].Name, strings.Join(arch, ","), tagPrefix))
	}

	sort.Strings(tagList)
	source := tagMap[tagList[len(tagList)-1]] + tagList[len(tagList)-1]
	return source, nil
}
