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
	"strings"
	"text/tabwriter"

	"gopkg.in/yaml.v3"

	reg "gopkg.openfuyao.cn/bkeadm/pkg/registry"
	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type PreCheckOptions struct {
	root.Options
	File      string `json:"file"`
	OnlyImage bool   `json:"onlyImage"`
}

// loadBuildConfig 加载并解析构建配置文件
func loadBuildConfig(filePath string) (*BuildConfig, error) {
	cfg := &BuildConfig{}
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the file, %s", err.Error())
	}
	if err = yaml.Unmarshal(yamlFile, cfg); err != nil {
		return nil, fmt.Errorf("unable to serialize file, %s", err.Error())
	}
	return cfg, nil
}

// imagePrefix 构建镜像前缀
func imagePrefix(sourceRepo string) string {
	repos := strings.Split(sourceRepo, "/")
	if len(repos) <= 1 {
		return ""
	}
	prefixImage := strings.Join(repos[1:], "/")
	if !strings.HasSuffix(prefixImage, "/") {
		prefixImage += "/"
	}
	return prefixImage
}

// imageTagMap 构建镜像标签映射
func imageTagMap(subImage SubImage, prefixImage string, arch []string) map[string][]string {
	images := map[string][]string{}
	for _, image := range subImage.Images {
		img := prefixImage + image.Name
		images[img] = processTags(image.Tag, arch)
	}
	return images
}

// processTags 处理镜像标签列表
func processTags(tags []string, arch []string) []string {
	var result []string
	for _, tag := range tags {
		result = append(result, expandTag(tag, arch)...)
	}
	return result
}

// expandTag 根据架构展开单个标签
func expandTag(tag string, arch []string) []string {
	if !strings.Contains(tag, cut) {
		return []string{tag}
	}

	tags := make([]string, 0, len(arch))
	for _, ar := range arch {
		tags = append(tags, strings.ReplaceAll(tag, cut, fmt.Sprintf("-%s-", ar)))
	}
	return tags
}

// checkRepoImages 检查配置中的镜像并返回检查结果
func checkRepoImages(cfg *BuildConfig) ([][]string, error) {
	var rows [][]string
	for _, cr := range cfg.Repos {
		for _, subImage := range cr.SubImages {
			repos := strings.Split(subImage.SourceRepo, "/")
			prefixImage := imagePrefix(subImage.SourceRepo)
			images := imageTagMap(subImage, prefixImage, cr.Architecture)

			res, err := reg.ViewRepoImage(repos[0], images)
			if err != nil {
				return nil, fmt.Errorf("failed to check the image %s, %s", subImage.SourceRepo, err.Error())
			}
			for _, v := range res {
				rows = append(rows, v)
			}
		}
	}
	return rows, nil
}

func (bp *PreCheckOptions) PreCheck() {
	cfg, err := loadBuildConfig(bp.File)
	if err != nil {
		log.BKEFormat(log.ERROR, err.Error())
		return
	}

	fmt.Println("Checking the configuration...")
	if !bp.OnlyImage {
		if err = verifyConfigContent(cfg); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Configuration verification fails %s", err.Error()))
			return
		}
	}

	headers := []string{"IMAGE", "TAGS", "ARCHITECTURE", "CREATE_TIME", "SIZE"}
	rows, err := checkRepoImages(cfg)
	if err != nil {
		log.BKEFormat(log.ERROR, err.Error())
		return
	}

	if err = exportTableToFile(bp.File, headers, rows); err != nil {
		return
	}
}

// exportTableToFile 将表格数据导出到文本文件
func exportTableToFile(filePath string, headers []string, rows [][]string) error {
	// 读取文件名称
	fileName := strings.Split(filePath, "/")
	fileName = strings.Split(fileName[len(fileName)-1], ".")
	name := fileName[0]
	// 文件名长度，用于判断是否需要拼接
	const fileLen = 2
	if len(fileName) > fileLen {
		name = strings.Join(fileName[0:len(fileName)-1], ".")
	}

	const tabPadding = 2 // 列之间的最小空格数，用于tabwriter对齐
	// 以表格形式输出到文本中
	w := &strings.Builder{}
	tw := tabwriter.NewWriter(w, 0, 0, tabPadding, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	err := tw.Flush()
	if err != nil {
		fmt.Println("flush tablewriter failed:", err.Error())
		return err
	}
	// 导出文件的权限，rw-r--r--
	const filePerm = 0644
	outputFileName := name + "-config-check.txt"
	err = os.WriteFile(outputFileName, []byte(w.String()), filePerm)
	if err != nil {
		fmt.Println("export txt failed: ", err.Error())
		return err
	}
	fmt.Println(fmt.Sprintf("Check the configuration is successful, check the file %s", outputFileName))
	return nil
}
