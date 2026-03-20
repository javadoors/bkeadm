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

package registry

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/containers/common/pkg/retry"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/transports/alltransports"
	"github.com/containers/image/v5/types"

	"gopkg.openfuyao.cn/bkeadm/pkg/root"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

type Options struct {
	root.Options
	Args []string `json:"args"`

	// sync
	File          string `json:"file"`
	Source        string `json:"source"`
	Target        string `json:"target"`
	MultiArch     bool   `json:"multi-arch"`
	SrcTLSVerify  bool   `json:"src-tls-verify"`
	DestTLSVerify bool   `json:"dest-tls-verify"`
	SyncRepo      bool   `json:"sync-repo"`

	// transfer
	Arch  string `json:"arch"`
	Image string `json:"image"`

	// view
	Prefix string `json:"prefix"`
	Tags   int    `json:"tags"`
	Export bool   `json:"export"`
}

// getArchList returns the architecture list based on options
func (op *Options) getArchList() []string {
	if op.MultiArch {
		return []string{"amd64", "arm64"}
	}
	if len(op.Arch) > 0 {
		return []string{op.Arch}
	}
	return []string{runtime.GOARCH}
}

// readImageListFromFile reads image names from a file, one per line
func readImageListFromFile(filename string) ([]string, error) {
	fi, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer fi.Close()

	var imageList []string
	buf := bufio.NewScanner(fi)
	for buf.Scan() {
		if text := buf.Text(); len(text) > 0 {
			imageList = append(imageList, text)
		}
	}
	return imageList, nil
}

// ensureTrailingSlash appends a trailing slash if not present
func ensureTrailingSlash(s string) string {
	if !strings.HasSuffix(s, "/") {
		return s + "/"
	}
	return s
}

// removeHTTPSchemePrefix removes http:// or https:// prefix from URL
func removeHTTPSchemePrefix(url string) string {
	for _, prefix := range []string{"https://", "http://"} {
		if strings.HasPrefix(url, prefix) {
			return strings.TrimPrefix(url, prefix)
		}
	}
	return url
}

// setupSyncHTTPClient creates HTTP client for sync operations
func setupSyncHTTPClient(srcRepo string) (*http.Client, string) {
	httpClient := &http.Client{}
	if !strings.HasPrefix(srcRepo, "http") {
		srcRepo = "https://" + srcRepo
		httpClient.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
	}
	return httpClient, srcRepo
}

// fetchImageTags fetches tags for a given image from registry
func fetchImageTags(httpClient *http.Client, srcRepo, img string) ([]string, error) {
	tagsURL := fmt.Sprintf("%s/v2/%s/tags/list", srcRepo, img)
	resp, err := httpClient.Get(tagsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("Failed to close response body: %v", err)
		}
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read tags response: %v", err)
	}

	var tags tagResponse
	if err = json.Unmarshal(body, &tags); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tags: %v", err)
	}

	if tags.Tags == nil || len(tags.Tags) == 0 {
		return nil, fmt.Errorf("no tags found")
	}
	return tags.Tags, nil
}

// buildSyncImageOptions builds Options for syncing a single image tag
func buildSyncImageOptions(baseOp Options, img, tag string) Options {
	syncOp := baseOp
	syncOp.Source = removeHTTPSchemePrefix(baseOp.Source + "/" + img + ":" + tag)
	syncOp.Target = removeHTTPSchemePrefix(baseOp.Target + "/" + img + ":" + tag)
	return syncOp
}

func (op *Options) Sync() {
	if op.SyncRepo {
		newOp := *op
		if err := syncRepo(newOp); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Sync repo %s to %s error %s", op.Source, op.Target, err.Error()))
			return
		}
		log.BKEFormat(log.INFO, fmt.Sprintf("Sync repo %s to %s success", op.Source, op.Target))
		return
	}

	archs := op.getArchList()

	if len(op.File) == 0 {
		if err := syncRepoImage(*op, archs); err != nil {
			log.Error(err.Error())
		}
		return
	}

	imageList, err := readImageListFromFile(op.File)
	if err != nil {
		log.Error(err.Error())
		return
	}

	sourceAddress := ensureTrailingSlash(op.Source)
	targetAddress := ensureTrailingSlash(op.Target)

	for _, image := range imageList {
		newOp := *op
		newOp.Source = sourceAddress + image
		newOp.Target = targetAddress + image
		if err := syncRepoImage(newOp, archs); err != nil {
			log.Error(err.Error())
			return
		}
	}
}

func syncRepo(op Options) error {
	httpClient, srcRepo := setupSyncHTTPClient(op.Source)
	url := fmt.Sprintf("%s/v2/_catalog?n=10000", srcRepo)

	resp, err := httpClient.Get(url)
	if err != nil {
		log.Fatalf("Failed to get repo: %v", err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read body: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		log.Warnf("Failed to close response body: %v", err)
	}

	var repos repo
	err = json.Unmarshal(body, &repos)
	if err != nil {
		log.Fatalf("Failed to unmarshal body: %v", err)
	}
	if repos.Repositories == nil || len(repos.Repositories) == 0 {
		log.Fatalf("Failed to get repo: %v", err)
	}

	failedImages := map[string][]string{}
	// syncRepo 用于整库同步，固定同步双架构
	arch := []string{"amd64", "arm64"}

	for _, img := range repos.Repositories {
		srcImg := srcRepo + "/" + img
		log.Infof("Handle image %s", srcImg)

		tags, err := fetchImageTags(httpClient, srcRepo, img)
		if err != nil {
			log.Warnf("%s get tags failed: %s", img, err.Error())
			failedImages[srcImg] = []string{err.Error()}
			continue
		}

		for _, tag := range utils.ReverseArray(tags) {
			syncImageOp := buildSyncImageOptions(op, img, tag)
			log.Infof("Sync image %s to %s", syncImageOp.Source, syncImageOp.Target)

			if err = syncRepoImage(syncImageOp, arch); err != nil {
				log.BKEFormat(log.ERROR, fmt.Sprintf("Sync image %s:%s error: %s", img, tag, err.Error()))
				failedImages[syncImageOp.Source] = []string{err.Error()}
				continue
			}
		}
	}

	if len(failedImages) > 0 {
		for key, value := range failedImages {
			log.BKEFormat(log.WARN, fmt.Sprintf("Failed to sync image %s %v", key, value))
		}
	}

	return nil
}

func syncRepoImage(newOp Options, arch []string) error {
	if len(arch) == 1 {
		return syncSingleArchRepoImage(newOp, arch[0])
	}
	return syncMultiArchRepoImage(newOp, arch)
}

// syncSingleArchRepoImage 同步单架构镜像
func syncSingleArchRepoImage(newOp Options, ar string) error {
	imageAddress := newOp.Source
	targetAddress := newOp.Target
	op := Options{
		MultiArch:     false,
		SrcTLSVerify:  newOp.SrcTLSVerify,
		DestTLSVerify: newOp.DestTLSVerify,
		Arch:          ar,
		Source:        imageAddress,
		Target:        targetAddress,
	}

	log.BKEFormat(log.NIL, fmt.Sprintf("Sync image %s to %s", imageAddress, targetAddress))
	if err := CopyRegistry(op); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Sync image %s to %s error %s", imageAddress, targetAddress, err.Error()))
		return trySyncWithArchSuffix(op, imageAddress, targetAddress, ar)
	}
	return nil
}

// trySyncWithArchSuffix 尝试使用架构后缀同步镜像
func trySyncWithArchSuffix(op Options, imageAddress, targetAddress, ar string) error {
	imageAddress = imageAddress + "-" + ar
	log.BKEFormat(log.NIL, fmt.Sprintf("Sync image %s to %s", imageAddress, targetAddress))
	op.Source = imageAddress
	if err := CopyRegistry(op); err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Sync image %s to %s error %s", imageAddress, targetAddress, err.Error()))
		return err
	}
	return nil
}

// syncMultiArchRepoImage 同步多架构镜像
func syncMultiArchRepoImage(newOp Options, arch []string) error {
	imageAddress := newOp.Source
	targetAddress := newOp.Target
	op := Options{
		MultiArch:     newOp.MultiArch,
		SrcTLSVerify:  newOp.SrcTLSVerify,
		DestTLSVerify: newOp.DestTLSVerify,
		Source:        imageAddress,
		Target:        targetAddress,
	}

	if err := CopyRegistry(op); err == nil {
		return nil
	} else {
		log.BKEFormat(log.WARN, fmt.Sprintf("Sync image %s to %s error %s by regitry",
			imageAddress, targetAddress, err.Error()))
	}

	return syncArchImagesAndCreateManifest(op, imageAddress, targetAddress, arch)
}

// syncArchImagesAndCreateManifest 同步各架构镜像并创建 manifest
func syncArchImagesAndCreateManifest(op Options, imageAddress, targetAddress string, arch []string) error {
	img := make([]ImageArch, 0, len(arch))
	op.MultiArch = false

	for _, ar := range arch {
		op.Arch = ar
		op.Source = imageAddress + "-" + ar
		op.Target = targetAddress + "-" + ar
		if err := CopyRegistry(op); err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("Sync image %s to %s error %s , arch %s",
				imageAddress, targetAddress, ar, err.Error()))
			return err
		}
		img = append(img, ImageArch{
			Name:         op.Target,
			OS:           "linux",
			Architecture: ar,
		})
	}

	if err := CreateMultiArchImage(img, targetAddress); err != nil {
		log.BKEFormat(log.ERROR, fmt.Sprintf("Tche creation of multiple schema images manifests fails %s %s",
			targetAddress, err.Error()))
		return err
	}
	return nil
}

// CopyRegistry copies an image from a source to a destination.
func CopyRegistry(op Options) error {
	imageAddress, targetAddress := normalizeImageAddresses(op.Source, op.Target)
	srcRef, destRef, err := parseImageReferences(imageAddress, targetAddress, op.Source)
	if err != nil {
		return err
	}

	policyContext, err := createPolicyContext()
	if err != nil {
		return err
	}

	sourceCtx, destinationCtx, err := createSystemContexts(op)
	if err != nil {
		return err
	}

	return executeCopyWithRetry(copyParams{
		srcRef:         srcRef,
		destRef:        destRef,
		policyContext:  policyContext,
		sourceCtx:      sourceCtx,
		destinationCtx: destinationCtx,
		imageAddress:   imageAddress,
		targetAddress:  targetAddress,
		multiArch:      op.MultiArch,
	})
}

// normalizeImageAddresses 为没有 transport 前缀的引用添加 docker://
func normalizeImageAddresses(source, target string) (string, string) {
	imageAddress := source
	targetAddress := target
	if !hasTransportPrefix(imageAddress) {
		imageAddress = fmt.Sprintf("docker://%s", imageAddress)
	}
	if !hasTransportPrefix(targetAddress) {
		targetAddress = fmt.Sprintf("docker://%s", targetAddress)
	}
	return imageAddress, targetAddress
}

// parseImageReferences 解析源和目标镜像引用
func parseImageReferences(imageAddress, targetAddress, originalSource string) (types.ImageReference,
	types.ImageReference, error) {
	srcRef, err := alltransports.ParseImageName(imageAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid source name %s: %v", originalSource, err)
	}
	destRef, err := alltransports.ParseImageName(targetAddress)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid destination name %s: %v", targetAddress, err)
	}
	return srcRef, destRef, nil
}

// createPolicyContext 创建签名策略上下文
func createPolicyContext() (*signature.PolicyContext, error) {
	policy := &signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}}
	policyContext, err := signature.NewPolicyContext(policy)
	if err != nil {
		return nil, fmt.Errorf("error loading trust policy: %v", err)
	}
	return policyContext, nil
}

// createSystemContexts 创建源和目标系统上下文
func createSystemContexts(op Options) (*types.SystemContext, *types.SystemContext, error) {
	sourceCtx, err := newSystemContext()
	if err != nil {
		return nil, nil, err
	}
	if !op.SrcTLSVerify {
		sourceCtx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(true)
		sourceCtx.DockerDaemonInsecureSkipTLSVerify = true
	}

	destinationCtx, err := newSystemContext()
	if err != nil {
		return nil, nil, err
	}
	if !op.DestTLSVerify {
		destinationCtx.DockerInsecureSkipTLSVerify = types.NewOptionalBool(true)
		destinationCtx.DockerDaemonInsecureSkipTLSVerify = true
	}

	if len(op.Arch) > 0 {
		sourceCtx.ArchitectureChoice = op.Arch
		destinationCtx.ArchitectureChoice = op.Arch
	}
	return sourceCtx, destinationCtx, nil
}

// copyParams 镜像复制参数
type copyParams struct {
	srcRef         types.ImageReference
	destRef        types.ImageReference
	policyContext  *signature.PolicyContext
	sourceCtx      *types.SystemContext
	destinationCtx *types.SystemContext
	imageAddress   string
	targetAddress  string
	multiArch      bool
}

// executeCopyWithRetry 执行带重试的镜像复制操作
func executeCopyWithRetry(params copyParams) error {
	ctx := context.Background()
	imageListSelection := copy.CopySystemImage
	if params.multiArch {
		imageListSelection = copy.CopyAllImages
	}

	return retry.IfNecessary(ctx, func() error {
		_, err := copy.Image(ctx, params.policyContext, params.destRef, params.srcRef, &copy.Options{
			RemoveSignatures:   false,
			ReportWriter:       os.Stdout,
			SourceCtx:          params.sourceCtx,
			DestinationCtx:     params.destinationCtx,
			ImageListSelection: imageListSelection,
			PreserveDigests:    false,
		})
		if err != nil {
			log.BKEFormat(log.ERROR, fmt.Sprintf("exec sync image %s to %s error %s", params.imageAddress,
				params.targetAddress, err.Error()))
			return err
		}
		return nil
	}, &retry.Options{
		MaxRetry: 9,
		Delay:    1,
	})
}

// hasTransportPrefix checks if the image reference already has a transport prefix
func hasTransportPrefix(ref string) bool {
	transports := []string{
		"docker://",
		"oci:",
		"dir:",
		"docker-archive:",
		"oci-archive:",
		"docker-daemon:",
		"containers-storage:",
	}
	for _, transport := range transports {
		if strings.HasPrefix(ref, transport) {
			return true
		}
	}
	return false
}

// newSystemContext returns a *types.SystemContext corresponding to opts.
// It is guaranteed to return a fresh instance, so it is safe to make additional updates to it.
func newSystemContext() (*types.SystemContext, error) {
	ctx := &types.SystemContext{
		RegistriesDirPath:        "",
		ArchitectureChoice:       "",
		OSChoice:                 "",
		VariantChoice:            "",
		SystemRegistriesConfPath: "",
		BigFilesTemporaryDir:     "",
		DockerRegistryUserAgent:  "bke/v1.0.0",
	}

	ctx.DockerCertPath = ""
	ctx.OCISharedBlobDirPath = ""
	ctx.AuthFilePath = os.Getenv("REGISTRY_AUTH_FILE")
	ctx.DockerDaemonHost = ""
	ctx.DockerDaemonCertPath = ""
	return ctx, nil
}
