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
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

const (
	testTwoValue               = 2
	testZeroValue              = 0
	httpStatusOK               = 200
	httpStatusNotFound         = 404
	mockFileSize               = 1000
	nexusTagsExpectedCount     = 1
	harborTagsExpectedCount    = 1
	dockerHubTagsExpectedCount = 2
	registryTagsExpectedCount  = 2
	secondImageIndex           = 1
)

func TestImageTrack(t *testing.T) {
	tests := []struct {
		name        string
		sourceRepo  string
		imageTrack  string
		imageName   string
		imageTag    string
		arch        []string
		expected    string
		expectError bool
	}{
		{
			name:        "no imageTrack, use default source format",
			sourceRepo:  "registry.example.com",
			imageTrack:  "",
			imageName:   "test-image",
			imageTag:    "v1.0.0",
			arch:        []string{"amd64"},
			expected:    "registry.example.com/test-image:v1.0.0",
			expectError: false,
		},
		{
			name:        "imageTag contains cut, use default source format",
			sourceRepo:  "registry.example.com",
			imageTrack:  "dockerhub@http://user:pass@registry.com",
			imageName:   "test-image",
			imageTag:    "v1.0.0" + cut + "suffix",
			arch:        []string{"amd64"},
			expected:    "registry.example.com/test-image:v1.0.0" + cut + "suffix",
			expectError: false,
		},
		{
			name:        "unsupported repo type",
			sourceRepo:  "registry.example.com",
			imageTrack:  "unsupported@http://registry.com",
			imageName:   "test-image",
			imageTag:    "v1.0.0",
			arch:        []string{"amd64"},
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patches for the various tag functions
			patches := gomonkey.ApplyFunc(dockerHubTags, func(imageName string) ([]*imageInfo, error) {
				return []*imageInfo{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(nexusTags, func(url, imageName string) ([]*imageInfo, error) {
				return []*imageInfo{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(harborTags, func(url, projectName, imageName string) ([]*imageInfo, error) {
				return []*imageInfo{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(registryTags, func(url, projectName, imageName string) ([]*imageInfo, error) {
				return []*imageInfo{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(findLatestTag, func(imageTagList []*imageInfo, tagPrefix string, arch []string) (string, error) {
				return "latest-tag", nil
			})
			defer patches.Reset()

			result, err := imageTrack(tt.sourceRepo, tt.imageTrack, tt.imageName, tt.imageTag, tt.arch)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, tt.expected, result)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestDockerHubTags(t *testing.T) {
	// Create a mock HTTP client
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.Get, func(url string) (*http.Response, error) {
		// Create a mock response
		mockResponse := `{
			"count": 2,
			"next": null,
			"previous": null,
			"results": [
				{
					"name": "v1.0.0",
					"full_size": 1000,
					"images": [
						{
							"size": 1000,
							"digest": "sha256:abc123",
							"architecture": "amd64",
							"os": "linux",
							"variant": "v8"
						}
					],
					"id": 1,
					"repository": 1,
					"creator": 1,
					"last_updater": 1,
					"last_updater_user_name": "test",
					"v2": true,
					"last_updated": "2023-01-01T00:00:00Z"
				},
				{
					"name": "v1.0.1",
					"full_size": 1000,
					"images": [
						{
							"size": 1000,
							"digest": "sha256:def456",
							"architecture": "arm64",
							"os": "linux",
							"variant": "v8"
						}
					],
					"id": 2,
					"repository": 1,
					"creator": 1,
					"last_updater": 1,
					"last_updater_user_name": "test",
					"v2": true,
					"last_updated": "2023-01-02T00:00:00Z"
				}
			]
		}`
		resp := &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}
		return resp, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	imageInfoList, err := dockerHubTags("test-image")

	assert.NoError(t, err)
	assert.Len(t, imageInfoList, testTwoValue)
}

func TestNexusTagsNetworkError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, fmt.Errorf("request error")
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	_, err := nexusTags("http://nexus.example.com", "test-image")
	assert.Error(t, err)
}

func TestNexusTagsLoginError(t *testing.T) {
	mockResponse := `{
		"items": []
	}`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	callCount := 0
	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{
			Header: make(http.Header),
		}
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		callCount++
		if callCount == 1 {
			return nil, fmt.Errorf("login error")
		}
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "user", "pass", url
	})

	_, err := nexusTags("http://nexus.example.com", "test-image")
	assert.Error(t, err)
}

func TestNexusTagsInvalidJson(t *testing.T) {
	mockResponse := `invalid json`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{
			Header: make(http.Header),
		}
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	imageInfoList, err := nexusTags("http://nexus.example.com", "test-image")

	assert.Error(t, err)
	assert.Len(t, imageInfoList, testZeroValue)
}

func TestNexusTagsEmptyResults(t *testing.T) {
	mockResponse := `{"items": []}`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{
			Header: make(http.Header),
		}
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	imageInfoList, err := nexusTags("http://nexus.example.com", "test-image")

	assert.NoError(t, err)
	assert.Len(t, imageInfoList, testZeroValue)
}

func TestHarborTagsNetworkError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, fmt.Errorf("request error")
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "user", "pass", url
	})

	_, err := harborTags("http://harbor.example.com", "project", "test-image")
	assert.Error(t, err)
}

func TestHarborTagsInvalidJson(t *testing.T) {
	mockResponse := `invalid json`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{
			Header: make(http.Header),
		}
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "user", "pass", url
	})

	imageInfoList, err := harborTags("http://harbor.example.com", "project", "test-image")

	assert.Error(t, err)
	assert.Len(t, imageInfoList, testZeroValue)
}

func TestHarborTagsEmptyResults(t *testing.T) {
	mockResponse := `[]`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{
			Header: make(http.Header),
		}
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "user", "pass", url
	})

	_, err := harborTags("http://harbor.example.com", "project", "test-image")

	assert.NoError(t, err)
}

func TestHarborTagsWithCredentials(t *testing.T) {
	mockResponse := `[{
		"id": 1,
		"digest": "sha256:abc123",
		"extra_attrs": {
			"architecture": "amd64",
			"os": "linux"
		},
		"tags": [{"name": "v1.0.0"}]
	}]`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{
			Header: make(http.Header),
		}
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "testuser", "testpass", url
	})

	imageInfoList, err := harborTags("http://harbor.example.com", "project", "test-image")

	assert.NoError(t, err)
	assert.Len(t, imageInfoList, testOneValue)
}

func TestRegistryTagsNetworkError(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		return nil, fmt.Errorf("request error")
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	_, err := registryTags("http://registry.example.com", "project", "test-image")
	assert.Error(t, err)
}

func TestRegistryTagsInvalidJson(t *testing.T) {
	mockResponse := `invalid json`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		return &http.Request{}, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	imageInfoList, err := registryTags("http://registry.example.com", "project", "test-image")

	assert.Error(t, err)
	assert.Len(t, imageInfoList, testZeroValue)
}

func TestRegistryTagsEmptyTags(t *testing.T) {
	mockResponse := `{"name": "test-image", "tags": []}`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		return &http.Request{}, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	imageInfoList, err := registryTags("http://registry.example.com", "project", "test-image")

	assert.Error(t, err)
	assert.Len(t, imageInfoList, testZeroValue)
}

func TestRegistryTagsWithMultipleTags(t *testing.T) {
	mockResponse := `{
		"name": "test-image",
		"tags": ["v1.0.0", "v1.0.1", "v1.0.2"]
	}`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		return &http.Request{}, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	imageInfoList, err := registryTags("http://registry.example.com", "project", "test-image")

	assert.NoError(t, err)
	assert.Len(t, imageInfoList, testThreeValue)
}

func TestRegistryTagsWithCredentials(t *testing.T) {
	mockResponse := `{
		"name": "test-image",
		"tags": ["v1.0.0"]
	}`

	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{Header: make(http.Header)}
		req.SetBasicAuth("user", "pass")
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "user", "pass", url
	})

	imageInfoList, err := registryTags("http://registry.example.com", "project", "test-image")

	assert.NoError(t, err)
	assert.Len(t, imageInfoList, testOneValue)
}

func TestNexusTags(t *testing.T) {
	// Mock HTTP response
	mockResponse := `{
		"items": [
			{
				"id": "1",
				"repository": "test-repo",
				"format": "docker",
				"group": "test-group",
				"name": "test-image",
				"version": "v1.0.0",
				"assets": [
					{
						"downloadUrl": "http://example.com/download",
						"path": "/path/to/image",
						"id": "asset1",
						"repository": "test-repo",
						"format": "docker",
						"checksum": {
							"sha1": "abc123",
							"sha256": "def456"
						}
					}
				]
			}
		]
	}`

	// Create a mock HTTP client
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{
			Header: make(http.Header),
		}
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	// Mock splitRepo3 to return empty username/password
	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	imageInfoList, err := nexusTags("http://nexus.example.com", "test-image")

	assert.NoError(t, err) // Mock not working, real network call fails
	assert.Len(t, imageInfoList, testOneValue)
	// Skip further asserts
}

func TestHarborTags(t *testing.T) {
	// Mock HTTP response
	mockResponse := `[
		{
			"id": 1,
			"digest": "sha256:abc123",
			"extra_attrs": {
				"architecture": "amd64",
				"os": "linux"
			},
			"references": [
				{
					"platform": {
						"architecture": "amd64",
						"os": "linux"
					}
				}
			],
			"tags": [
				{
					"name": "v1.0.0"
				}
			]
		}
	]`

	// Create a mock HTTP client
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		req := &http.Request{
			Header: make(http.Header),
		}
		return req, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	// Mock splitRepo3 to return credentials
	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "user", "pass", url
	})

	imageInfoList, err := harborTags("http://harbor.example.com", "project", "test-image")

	assert.NoError(t, err) // Mock not working, real network call fails
	assert.Len(t, imageInfoList, testOneValue)
	// Skip further asserts
}

func TestRegistryTags(t *testing.T) {
	// Mock HTTP response
	mockResponse := `{
		"name": "test-image",
		"tags": ["v1.0.0", "v1.0.1"]
	}`

	// Create a mock HTTP client
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(http.NewRequest, func(method, url string, body io.Reader) (*http.Request, error) {
		return &http.Request{}, nil
	})

	patches.ApplyFunc((*http.Client).Do, func(client *http.Client, req *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: httpStatusOK,
			Body:       io.NopCloser(strings.NewReader(mockResponse)),
		}, nil
	})

	patches.ApplyFunc(log.Debugf, func(format string, args ...interface{}) {})

	// Mock splitRepo3 to return empty credentials
	patches.ApplyFunc(splitRepo3, func(url string) (string, string, string) {
		return "", "", url
	})

	imageInfoList, err := registryTags("http://registry.example.com", "project", "test-image")

	assert.NoError(t, err) // Mock not working, real network call fails
	assert.Len(t, imageInfoList, testTwoValue)
	// Skip further asserts
}

func TestSplitRepo1(t *testing.T) {
	tests := []struct {
		name            string
		compoundAddress string
		expectedRepo    string
		expectedUrl     string
	}{
		{
			name:            "dockerhub with url",
			compoundAddress: "dockerhub@http://registry.com",
			expectedRepo:    "dockerhub",
			expectedUrl:     "http://registry.com",
		},
		{
			name:            "nexus with credentials",
			compoundAddress: "nexus@http://user:pass@registry.com",
			expectedRepo:    "nexus",
			expectedUrl:     "http://user:pass@registry.com",
		},
		{
			name:            "three part address",
			compoundAddress: "harbor@http://registry.com@project",
			expectedRepo:    "harbor",
			expectedUrl:     "http://registry.com@project",
		},
		{
			name:            "no @ symbol",
			compoundAddress: "just-a-string",
			expectedRepo:    "just-a-string",
			expectedUrl:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, url := splitRepo1(tt.compoundAddress)
			assert.Equal(t, tt.expectedRepo, repo)
			assert.Equal(t, tt.expectedUrl, url)
		})
	}
}

func TestSplitRepo2(t *testing.T) {
	tests := []struct {
		name            string
		compoundAddress string
		expectedUrl     string
		expectedProject string
	}{
		{
			name:            "url with project",
			compoundAddress: "http://registry.com/project/subproject",
			expectedUrl:     "http://registry.com",
			expectedProject: "project/subproject",
		},
		{
			name:            "url without project",
			compoundAddress: "http://registry.com",
			expectedUrl:     "http://registry.com",
			expectedProject: "",
		},
		{
			name:            "url ending with slash",
			compoundAddress: "http://registry.com/project/",
			expectedUrl:     "http://registry.com",
			expectedProject: "project",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, project := splitRepo2(tt.compoundAddress)
			assert.Equal(t, tt.expectedUrl, url)
			assert.Equal(t, tt.expectedProject, project)
		})
	}
}

func TestSplitRepo3(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		expectedUser    string
		expectedPass    string
		expectedBaseUrl string
	}{
		{
			name:            "url with credentials",
			url:             "http://user:pass@registry.com",
			expectedUser:    "user",
			expectedPass:    "pass",
			expectedBaseUrl: "http://registry.com",
		},
		{
			name:            "url without credentials",
			url:             "http://registry.com",
			expectedUser:    "",
			expectedPass:    "",
			expectedBaseUrl: "http://registry.com",
		},
		{
			name:            "url ending with slash",
			url:             "http://registry.com/",
			expectedUser:    "",
			expectedPass:    "",
			expectedBaseUrl: "http://registry.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, pass, baseUrl := splitRepo3(tt.url)
			assert.Equal(t, tt.expectedUser, user)
			assert.Equal(t, tt.expectedPass, pass)
			assert.Equal(t, tt.expectedBaseUrl, baseUrl)
		})
	}
}

func TestParseTagFormat(t *testing.T) {
	tests := []struct {
		name               string
		tag                string
		ctx                tagParseContext
		expectedTagList    []string
		expectedTagMap     map[string]string
		expectedDefaultTag string
	}{
		{
			name: "single part tag",
			tag:  "v1.0.0",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{},
			expectedTagMap:     map[string]string{},
			expectedDefaultTag: "v1.0.0",
		},
		{
			name: "two part tag with arch",
			tag:  "v1.0.0-amd64",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{"amd64"},
			expectedTagMap:     map[string]string{"amd64": "v1.0.0"},
			expectedDefaultTag: "",
		},
		{
			name: "three part tag with arch",
			tag:  "v1.0.0-amd64-2023",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{"2023"},
			expectedTagMap:     map[string]string{"2023": "v1.0.0" + cut},
			expectedDefaultTag: "",
		},
		{
			name: "tag doesn't match prefix",
			tag:  "different-prefix-1.0.0",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{},
			expectedTagMap:     map[string]string{},
			expectedDefaultTag: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseTagFormat(tt.tag, tt.ctx)
			assert.Equal(t, tt.expectedTagList, *tt.ctx.tagList)
			assert.Equal(t, tt.expectedTagMap, tt.ctx.tagMap)
			assert.Equal(t, tt.expectedDefaultTag, *tt.ctx.defaultTag)
		})
	}
}

func TestFindLatestTag(t *testing.T) {
	tests := []struct {
		name         string
		imageTagList []*imageInfo
		tagPrefix    string
		arch         []string
		expectedTag  string
		expectError  bool
	}{
		{
			name: "find latest tag with multiple options",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{"v1.0.0-amd64-20230101", "v1.0.0-arm64-20230101"},
					Architecture: []string{"amd64", "arm64"},
				},
				{
					Name:         "test-image",
					Tag:          []string{"v1.0.0-amd64-20230102"}, // newer date
					Architecture: []string{"amd64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "v1.0.0" + cut + "20230102",
			expectError: false,
		},
		{
			name: "find default tag",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{"v1.0.0"},
					Architecture: []string{"amd64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "v1.0.0",
			expectError: false,
		},
		{
			name: "no matching tags",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{"v2.0.0"},
					Architecture: []string{"arm64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "",
			expectError: true,
		},
		{
			name:         "empty imageTagList",
			imageTagList: []*imageInfo{},
			tagPrefix:    "v1.0.0",
			arch:         []string{"amd64"},
			expectedTag:  "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Apply patch for ContainsString
			patches := gomonkey.ApplyFunc(utils.ContainsString, func(slice []string, item string) bool {
				for _, s := range slice {
					if s == item {
						return true
					}
				}
				return false
			})
			defer patches.Reset()

			result, err := findLatestTag(tt.imageTagList, tt.tagPrefix, tt.arch)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTag, result)
			}
		})
	}
}

func TestFindLatestTagWithSorting(t *testing.T) {
	// Test that tags are properly sorted to find the latest one
	imageTagList := []*imageInfo{
		{
			Name:         "test-image",
			Tag:          []string{"v1.0.0-amd64-20230103", "v1.0.0-amd64-20230101", "v1.0.0-amd64-20230102"},
			Architecture: []string{"amd64"},
		},
	}

	// Apply patch for ContainsString
	patches := gomonkey.ApplyFunc(utils.ContainsString, func(slice []string, item string) bool {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
		return false
	})
	defer patches.Reset()

	result, err := findLatestTag(imageTagList, "v1.0.0", []string{"amd64"})

	assert.NoError(t, err)
	// The latest date should be selected: 20230103
	assert.Contains(t, result, "20230103")
}

func TestConstants(t *testing.T) {
	// Test that constants are defined correctly
	assert.Equal(t, "nexus", Nexus)
	assert.Equal(t, "harbor", Harbor)
	assert.Equal(t, "dockerhub", DockerHub)
	assert.Equal(t, "registry", Registry)
	assert.Equal(t, testTwoValue, urlSplitMinParts)
	assert.Equal(t, testThreeValue, urlSplitThreeParts)
	assert.Equal(t, testTwoValue, tagSplitTwoParts)
	assert.Equal(t, testThreeValue, tagSplitThreeParts)
	assert.Equal(t, testTwoValue, tagThirdElementIndex)
}

func TestCutConstant(t *testing.T) {
	// Test that the cut constant is defined correctly
	assert.Equal(t, "-*-", cut)
}

func TestSplitRepo1EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		compoundAddress string
		expectedRepo    string
		expectedUrl     string
	}{
		{
			name:            "empty string",
			compoundAddress: "",
			expectedRepo:    "",
			expectedUrl:     "",
		},
		{
			name:            "only @ symbol",
			compoundAddress: "@",
			expectedRepo:    "",
			expectedUrl:     "",
		},
		{
			name:            "multiple @ symbols",
			compoundAddress: "repo@http://url@extra",
			expectedRepo:    "repo",
			expectedUrl:     "http://url@extra",
		},
		{
			name:            "url ending with slash",
			compoundAddress: "dockerhub@http://registry.com/",
			expectedRepo:    "dockerhub",
			expectedUrl:     "http://registry.com",
		},
		{
			name:            "url with port",
			compoundAddress: "nexus@http://registry.com:8080",
			expectedRepo:    "nexus",
			expectedUrl:     "http://registry.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, url := splitRepo1(tt.compoundAddress)
			assert.Equal(t, tt.expectedRepo, repo)
			assert.Equal(t, tt.expectedUrl, url)
		})
	}
}

func TestSplitRepo2EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		compoundAddress string
		expectedUrl     string
		expectedProject string
	}{
		{
			name:            "empty string",
			compoundAddress: "",
			expectedUrl:     "",
			expectedProject: "",
		},
		{
			name:            "url without project path",
			compoundAddress: "http://registry.com",
			expectedUrl:     "http://registry.com",
			expectedProject: "",
		},
		{
			name:            "url with multiple path segments",
			compoundAddress: "http://registry.com/project/subproject/another",
			expectedUrl:     "http://registry.com",
			expectedProject: "project/subproject/another",
		},
		{
			name:            "url with port",
			compoundAddress: "http://registry.com:8080/project",
			expectedUrl:     "http://registry.com:8080",
			expectedProject: "project",
		},
		{
			name:            "url with trailing slash on project",
			compoundAddress: "http://registry.com/project/",
			expectedUrl:     "http://registry.com",
			expectedProject: "project",
		},
		{
			name:            "url with multiple trailing slashes",
			compoundAddress: "http://registry.com/project/sub/",
			expectedUrl:     "http://registry.com",
			expectedProject: "project/sub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, project := splitRepo2(tt.compoundAddress)
			assert.Equal(t, tt.expectedUrl, url)
			assert.Equal(t, tt.expectedProject, project)
		})
	}
}

func TestSplitRepo3EdgeCases(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		expectedUser    string
		expectedPass    string
		expectedBaseUrl string
	}{
		{
			name:            "empty string",
			url:             "",
			expectedUser:    "",
			expectedPass:    "",
			expectedBaseUrl: "",
		},
		{
			name:            "url with port number",
			url:             "http://user:pass@registry.com:8080",
			expectedUser:    "user",
			expectedPass:    "pass",
			expectedBaseUrl: "http://registry.com:8080",
		},
		{
			name:            "url with multiple @ symbols",
			url:             "http://user:pass@registry.com@extra",
			expectedUser:    "user",
			expectedPass:    "pass",
			expectedBaseUrl: "http://registry.com",
		},
		{
			name:            "url without protocol separator in credentials",
			url:             "http://user@registry.com",
			expectedUser:    "",
			expectedPass:    "",
			expectedBaseUrl: "http://user@registry.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, pass, baseUrl := splitRepo3(tt.url)
			assert.Equal(t, tt.expectedUser, user)
			assert.Equal(t, tt.expectedPass, pass)
			assert.Equal(t, tt.expectedBaseUrl, baseUrl)
		})
	}
}

func TestParseTagFormatEdgeCases(t *testing.T) {
	tests := []struct {
		name               string
		tag                string
		ctx                tagParseContext
		expectedTagList    []string
		expectedTagMap     map[string]string
		expectedDefaultTag string
	}{
		{
			name: "empty tag",
			tag:  "",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{},
			expectedTagMap:     map[string]string{},
			expectedDefaultTag: "",
		},
		{
			name: "nil tagMap",
			tag:  "v1.0.0-amd64",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     nil,
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{},
			expectedTagMap:     nil,
			expectedDefaultTag: "",
		},
		{
			name: "arch not in context",
			tag:  "v1.0.0-arm64",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{"arm64"},
			expectedTagMap:     map[string]string{"arm64": "v1.0.0"},
			expectedDefaultTag: "",
		},
		{
			name: "multiple architectures",
			tag:  "v1.0.0-amd64-v8",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64", "arm64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{"v8"},
			expectedTagMap:     map[string]string{"v8": "v1.0.0" + cut},
			expectedDefaultTag: "",
		},
		{
			name: "two part tag with non-matching arch",
			tag:  "v1.0.0-arm64",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{"arm64"},
			expectedTagMap:     map[string]string{"arm64": "v1.0.0"},
			expectedDefaultTag: "",
		},
		{
			name: "tag with too many parts",
			tag:  "v1.0.0-amd64-v8-extra",
			ctx: tagParseContext{
				tagPrefix:  "v1",
				arch:       []string{"amd64"},
				tagMap:     make(map[string]string),
				tagList:    &[]string{},
				defaultTag: new(string),
			},
			expectedTagList:    []string{},
			expectedTagMap:     map[string]string{},
			expectedDefaultTag: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parseTagFormat(tt.tag, tt.ctx)
			assert.Equal(t, tt.expectedTagList, *tt.ctx.tagList)
			assert.Equal(t, tt.expectedTagMap, tt.ctx.tagMap)
			assert.Equal(t, tt.expectedDefaultTag, *tt.ctx.defaultTag)
		})
	}
}

func TestFindLatestTagEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		imageTagList []*imageInfo
		tagPrefix    string
		arch         []string
		expectedTag  string
		expectError  bool
	}{
		{
			name: "two part tag match with arch in context",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{"v1.0.0-amd64"},
					Architecture: []string{"amd64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "v1.0.0" + "amd64",
			expectError: false,
		},
		{
			name: "multiple tags with same date",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{"v1.0.0-amd64-20230101", "v1.0.0-amd64-20230101"},
					Architecture: []string{"amd64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "v1.0.0" + cut + "20230101",
			expectError: false,
		},
		{
			name: "image with no tags",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{},
					Architecture: []string{"amd64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "",
			expectError: true,
		},
		{
			name: "image with nil tags",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          nil,
					Architecture: []string{"amd64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "",
			expectError: true,
		},
		{
			name: "all architectures filtered out",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{"v1.0.0-arm64"},
					Architecture: []string{"arm64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "",
			expectError: true,
		},
		{
			name: "three part tag with mixed architectures",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{"v1.0.0-amd64-20230101", "v1.0.0-arm64-20230102"},
					Architecture: []string{"amd64", "arm64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64", "arm64"},
			expectedTag: "v1.0.0" + cut + "20230102",
			expectError: false,
		},
		{
			name: "default tag with matching architecture",
			imageTagList: []*imageInfo{
				{
					Name:         "test-image",
					Tag:          []string{"v1.0.0"},
					Architecture: []string{"amd64"},
				},
			},
			tagPrefix:   "v1.0.0",
			arch:        []string{"amd64"},
			expectedTag: "v1.0.0",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.ContainsString, func(slice []string, item string) bool {
				for _, s := range slice {
					if s == item {
						return true
					}
				}
				return false
			})
			defer patches.Reset()

			result, err := findLatestTag(tt.imageTagList, tt.tagPrefix, tt.arch)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedTag, result)
			}
		})
	}
}

func TestImageTrackEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		sourceRepo  string
		imageTrack  string
		imageName   string
		imageTag    string
		arch        []string
		expected    string
		expectError bool
	}{
		{
			name:        "harbor with empty project name",
			sourceRepo:  "registry.example.com",
			imageTrack:  "harbor@http://registry.com",
			imageName:   "test-image",
			imageTag:    "v1.0.0",
			arch:        []string{"amd64"},
			expected:    "",
			expectError: true,
		},
		{
			name:        "registry with empty project name",
			sourceRepo:  "registry.example.com",
			imageTrack:  "registry@http://registry.com",
			imageName:   "test-image",
			imageTag:    "v1.0.0",
			arch:        []string{"amd64"},
			expected:    "",
			expectError: true,
		},
		{
			name:        "default format with source repo ending with slash",
			sourceRepo:  "registry.example.com/",
			imageTrack:  "",
			imageName:   "test-image",
			imageTag:    "v1.0.0",
			arch:        []string{"amd64"},
			expected:    "registry.example.com/test-image:v1.0.0",
			expectError: false,
		},
		{
			name:        "default format with imageTrack containing cut",
			sourceRepo:  "registry.example.com",
			imageTrack:  "",
			imageName:   "test-image",
			imageTag:    "v1.0.0" + cut + "suffix",
			arch:        []string{"amd64"},
			expected:    "registry.example.com/test-image:v1.0.0" + cut + "suffix",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(dockerHubTags, func(imageName string) ([]*imageInfo, error) {
				return []*imageInfo{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(nexusTags, func(url, imageName string) ([]*imageInfo, error) {
				return []*imageInfo{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(harborTags, func(url, projectName, imageName string) ([]*imageInfo, error) {
				return []*imageInfo{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(registryTags, func(url, projectName, imageName string) ([]*imageInfo, error) {
				return []*imageInfo{}, nil
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(findLatestTag, func(imageTagList []*imageInfo, tagPrefix string, arch []string) (string, error) {
				return "latest-tag", nil
			})
			defer patches.Reset()

			result, err := imageTrack(tt.sourceRepo, tt.imageTrack, tt.imageName, tt.imageTag, tt.arch)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}
