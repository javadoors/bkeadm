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

package docker

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/registry"
	dockerapi "github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

const (
	platformOSIndex      = 0
	platformArchIndex    = 1
	platformVariantIndex = 2
	minPlatformParts     = 2 // OS/Arch
	maxPlatformParts     = 3 // OS/Arch/Variant
)

// DockerClient is the interface for Docker client.
type DockerClient interface {
	GetClient() *dockerapi.Client
	ImageList() ([]ImageRef, error)
	HasImage(image string) bool
	Load(imageFile string) (string, error)
	Save(image, path string) error
	Tag(srcImage, targetImage string) error
	Pull(image ImageRef, options utils.RetryOptions) error
	Push(image ImageRef) error
	Remove(ref ImageRef) error
	EnsureImageExists(image ImageRef, options utils.RetryOptions) error

	Run(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) error
	ContainerStop(containerId string) error
	ContainerRemove(containerId string) error
	ContainerExists(containerName string) (types.ContainerJSON, bool)
	EnsureContainerRun(containerId string) (bool, error)
	CopyFromContainer(containerId, srcPath, dstPath string) error
}

type Client struct {
	Client *dockerapi.Client
	ctx    context.Context
}

type ImageRef struct {
	Image    string `json:"image"`
	Username string `json:"username"`
	Password string `json:"password"`
	Platform string `json:"platform,omitempty"`
}

type ContainerRef struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

const dockerSock = "/var/run/docker.sock"

func NewDockerClient() (DockerClient, error) {
	if !utils.Exists(dockerSock) {
		return nil, errors.New("docker service does not exist. ")
	}

	ctx := context.Background()
	cli, err := dockerapi.NewClientWithOpts(dockerapi.FromEnv, dockerapi.WithAPIVersionNegotiation())
	if err != nil {
		log.Debugf("get container runtime client err:", err)
		return nil, err
	}
	return &Client{
		Client: cli,
		ctx:    ctx,
	}, nil
}

func (c *Client) Close() {
	_ = c.Client.Close()
}

func (c *Client) GetClient() *dockerapi.Client {
	return c.Client
}

func (c *Client) ImageList() ([]ImageRef, error) {
	// docker images
	var localImages []ImageRef

	images, err := c.Client.ImageList(c.ctx, image.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, img := range images {
		for _, name := range img.RepoTags {
			localImages = append(localImages, ImageRef{
				Image:    name,
				Username: "",
				Password: "",
				Platform: "",
			})
		}
	}
	return localImages, nil
}

func (c *Client) HasImage(image string) bool {
	all, err := c.ImageList()
	if err != nil {
		log.Errorf("list image act error: %v", err)
	}
	for _, item := range all {
		if item.Image == image {
			return true
		}
	}
	return false
}
func (c *Client) Tag(srcImage string, targetImage string) error {
	// docker tag xxxx xxxx
	if err := c.Client.ImageTag(c.ctx, srcImage, targetImage); err != nil {
		return err
	}
	return nil
}

func (c *Client) ContainerExists(containerName string) (types.ContainerJSON, bool) {
	containerInfo, _ := c.Client.ContainerInspect(c.ctx, containerName)
	// Check whether the mirror warehouse already exists
	if containerInfo.ContainerJSONBase != nil {
		return containerInfo, true
	}
	return types.ContainerJSON{}, false

}
func (c *Client) ContainerRemove(containerId string) error {
	// docker rm
	containerRmvOpt := container.RemoveOptions{Force: true}
	if err := c.Client.ContainerRemove(c.ctx, containerId, containerRmvOpt); err != nil {
		log.Debugf("remove container %s error: %v", containerId, err)
		return err
	}
	return nil
}

func (c *Client) ContainerStop(containerId string) error {
	// docker stop
	if err := c.Client.ContainerStop(c.ctx, containerId, container.StopOptions{}); err != nil {
		log.Debugf("stop container %s error: %v", containerId, err)
		return err
	}
	return nil
}

func (c *Client) Save(image, path string) error {
	resp, err := c.Client.ImageSave(c.ctx, []string{image})
	if err != nil {
		return nil
	}
	body, err := io.ReadAll(resp)
	if err != nil {
		return err
	}
	err = os.WriteFile(path, body, utils.DefaultFilePermission)
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Load(image string) (string, error) {
	// docker load ./xxxx.tar
	file, err := os.OpenFile(image, os.O_RDONLY, utils.DefaultFilePermission)
	if err != nil {
		return " ", err
	}
	defer file.Close()
	resp, err := c.Client.ImageLoad(c.ctx, file)
	if err != nil {
		return " ", err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	tag := strings.Replace(strings.Split(string(body), ":")[3], "\\n\"}", "", -1)
	imageSource := strings.TrimSpace(strings.Split(string(body), ":")[2] + ":" + tag)
	return imageSource, nil
}

func (c *Client) imagePull(image string, imagePullOptions image.PullOptions,
	retryOptions utils.RetryOptions) (io.ReadCloser, error) {
	var reader io.ReadCloser
	var err error

	for i := 0; i < retryOptions.MaxRetry; i++ {
		reader, err = c.Client.ImagePull(c.ctx, image, imagePullOptions)
		if err == nil {
			return reader, nil
		}
		log.BKEFormat(log.WARN, fmt.Sprintf("Image %s pull failed: %v, retrying (%d/%d)...", image, err, i+1,
			retryOptions.MaxRetry))
		time.Sleep(retryOptions.Delay * time.Second)
	}

	return nil, fmt.Errorf("failed to pull image %s after %d attempts: %w", image, retryOptions.MaxRetry, err)
}

func (c *Client) imageInspectWithRaw(image string, retryOptions utils.RetryOptions) (types.ImageInspect, error) {
	var inspect types.ImageInspect
	var err error

	for i := 0; i < retryOptions.MaxRetry; i++ {
		inspect, _, err = c.Client.ImageInspectWithRaw(c.ctx, image)
		if err == nil {
			return inspect, nil
		}
		log.BKEFormat(log.WARN, fmt.Sprintf("Image %s inspect failed: %v, retrying (%d/%d)...", image, err, i+1,
			retryOptions.MaxRetry))
		time.Sleep(retryOptions.Delay * time.Second)
	}

	return inspect, fmt.Errorf("failed to inspect image %s after %d attempts: %w", image, retryOptions.MaxRetry, err)
}

// Pull pulls the image from the registry.
func (c *Client) Pull(img ImageRef, retryOptions utils.RetryOptions) error {
	imagePullOptions := image.PullOptions{}
	if len(img.Username) != 0 && len(img.Password) != 0 {
		authConfig := registry.AuthConfig{
			Username: img.Username,
			Password: img.Password,
		}
		encodedJSON, err := json.Marshal(authConfig)
		if err != nil {
			return err
		}
		authStr := base64.URLEncoding.EncodeToString(encodedJSON)
		imagePullOptions.RegistryAuth = authStr
	}
	if len(img.Platform) > 0 {
		imagePullOptions.Platform = img.Platform
	}

	reader, err := c.imagePull(img.Image, imagePullOptions, retryOptions)
	if err != nil {
		return err
	}
	out, err := os.Create(filepath.Join(os.TempDir(), "bke-download-image.log"))
	if err != nil {
		log.Warnf("failed to create download log file: %s", err.Error())
	}
	if err := out.Chmod(utils.DefaultFilePermission); err != nil {
		log.Warnf("failed to set download log file permission: %s", err.Error())
	}
	wt := bufio.NewWriter(out)
	defer func() {
		if err := out.Close(); err != nil {
			log.Warnf("failed to close download log file: %s", err.Error())
		}
	}()
	if _, err := io.Copy(wt, reader); err != nil {
		log.Warnf("failed to write to download log: %s", err.Error())
	}
	if err := wt.Flush(); err != nil {
		log.Warnf("failed to flush download log: %s", err.Error())
	}
	if img.Platform != "" {
		inspect, err := c.imageInspectWithRaw(img.Image, retryOptions)
		if err != nil {
			return err
		}
		if !strings.Contains(inspect.Architecture, img.Platform) {
			return errors.New(fmt.Sprintf("Image %s Architecture %s is different from the expected architecture %s",
				img.Image, inspect.Architecture, img.Platform))
		}
	}
	return nil
}

// Push pushes the image to the registry.
func (c *Client) Push(img ImageRef) error {
	authConfig := registry.AuthConfig{
		Username: img.Username,
		Password: img.Password,
	}
	encodedJSON, err := json.Marshal(authConfig)
	if err != nil {
		return err
	}
	authStr := base64.URLEncoding.EncodeToString(encodedJSON)
	imagePushOptions := image.PushOptions{All: true, RegistryAuth: authStr}
	if len(img.Platform) > 0 {
		parts := strings.Split(img.Platform, "/")
		if len(parts) < minPlatformParts || len(parts) > maxPlatformParts {
			return fmt.Errorf("invalid platform: %s (expected 'os/arch' or 'os/arch/variant')", img.Platform)
		}

		platform := specs.Platform{OS: parts[platformOSIndex]}
		if len(parts) > platformArchIndex {
			platform.Architecture = parts[platformArchIndex]
		}
		if len(parts) > platformVariantIndex {
			platform.Variant = parts[platformVariantIndex]
		}
		imagePushOptions.Platform = &platform
	}

	closer, err := c.Client.ImagePush(c.ctx, img.Image, imagePushOptions)
	if err != nil {
		return err
	}
	out, err := os.Create(filepath.Join(os.TempDir(), "bke-push-image.log"))
	if err != nil {
		log.Warnf("failed to create push log file: %s", err.Error())
	}
	if err := out.Chmod(utils.DefaultFilePermission); err != nil {
		log.Warnf("failed to set push log file permission: %s", err.Error())
	}
	wt := bufio.NewWriter(out)
	defer func() {
		if err := out.Close(); err != nil {
			log.Warnf("failed to close push log file: %s", err.Error())
		}
	}()
	if _, err := io.Copy(wt, closer); err != nil {
		log.Warnf("failed to write to push log: %s", err.Error())
	}
	if err := wt.Flush(); err != nil {
		log.Warnf("failed to flush push log: %s", err.Error())
	}
	return nil
}

// Remove removes the image from the local registry.
func (c *Client) Remove(ref ImageRef) error {
	_, err := c.Client.ImageRemove(c.ctx, ref.Image, image.RemoveOptions{Force: true})
	if err != nil {
		return err
	}
	return nil
}

// Run runs the container.
func (c *Client) Run(config *container.Config, hostConfig *container.HostConfig, networkingConfig *network.NetworkingConfig, platform *specs.Platform, containerName string) error {
	resp, err := c.Client.ContainerCreate(c.ctx, config, hostConfig, networkingConfig, platform, containerName)
	if err != nil {
		return err
	}
	if err = c.Client.ContainerStart(c.ctx, resp.ID, container.StartOptions{}); err != nil {
		return err
	}
	log.Debugf("container ID %s", resp.ID)
	return nil
}

// EnsureImageExists ensures the image exists in the local registry.
func (c *Client) EnsureImageExists(image ImageRef, retryOptions utils.RetryOptions) error {
	imageInspect, err := c.imageInspectWithRaw(image.Image, retryOptions)
	if err != nil {
		log.BKEFormat(log.WARN, fmt.Sprintf("Get image %s inspect failed: %v", image.Image, err))
	}
	if imageInspect.ID == "" {
		log.BKEFormat(log.INFO, fmt.Sprintf("Image %s is downloading", image.Image))
		err := c.Pull(image, retryOptions)
		if err != nil {
			return err
		}
	}
	return nil
}

// EnsureContainerRun ensures the container exists and running.
func (c *Client) EnsureContainerRun(containerId string) (bool, error) {
	containerInfo, _ := c.Client.ContainerInspect(c.ctx, containerId)
	// Check whether the mirror warehouse already exists
	if containerInfo.ContainerJSONBase != nil {
		if containerInfo.State.Running {
			return true, nil
		}
		err := c.Client.ContainerStart(c.ctx, containerInfo.ID, container.StartOptions{})
		if err == nil {
			log.BKEFormat(log.INFO, "The image registry service already running")
			return true, nil
		}
		err = c.Client.ContainerRemove(c.ctx, containerInfo.ID, container.RemoveOptions{Force: true})
		if err != nil {
			log.BKEFormat(log.ERROR, "Failed to delete the image registry service")
			return false, err
		}
	}
	return false, nil
}
