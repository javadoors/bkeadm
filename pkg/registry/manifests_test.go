/******************************************************************
 * Copyright (c) 2026 Huawei Technologies Co., Ltd.
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
	"encoding/json"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
)

func TestCreateMultiArchImage_PreferOCIIndexAndOSLowercase(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	// Stub: first child is OCI manifest, second is docker manifest
	call := 0
	patches.ApplyFunc(getImageSha, func(name string) (string, int, string, error) {
		call++
		if call == 1 {
			return "sha256:child1", 123, DockerV1Schema2MediaType, nil
		}
		return "sha256:child2", 456, DockerV2Schema2MediaType, nil
	})

	var captured string
	patches.ApplyFunc(putManifests, func(manifests string, name string) error {
		captured = manifests
		assert.Equal(t, "example.com/repo/img:latest", name)
		return nil
	})

	err := CreateMultiArchImage([]ImageArch{
		{Name: "example.com/repo/img:latest-amd64", OS: "linux", Architecture: "amd64"},
		{Name: "example.com/repo/img:latest-arm64", OS: "linux", Architecture: "arm64"},
	}, "example.com/repo/img:latest")
	assert.NoError(t, err)
	assert.NotEmpty(t, captured)

	var out map[string]any
	assert.NoError(t, json.Unmarshal([]byte(captured), &out))

	// Top-level should be OCI index if any child is OCI.
	assert.Equal(t, float64(2), out["schemaVersion"])
	assert.Equal(t, DockerV1ListMediaType, out["mediaType"])

	manifests, ok := out["manifests"].([]any)
	assert.True(t, ok)
	assert.Len(t, manifests, 2)

	first := manifests[0].(map[string]any)
	platform := first["platform"].(map[string]any)

	// Ensure field is "os" (lowercase) in the serialized JSON.
	_, hasOSUpper := platform["OS"]
	_, hasOSLower := platform["os"]
	assert.False(t, hasOSUpper)
	assert.True(t, hasOSLower)
	assert.Equal(t, "linux", platform["os"])

	// Child mediaType should be whatever getImageSha returned for that image.
	assert.Equal(t, DockerV1Schema2MediaType, first["mediaType"])
}

func TestCreateMultiArchImage_KeepDockerListWhenAllChildrenDocker(t *testing.T) {
	patches := gomonkey.NewPatches()
	defer patches.Reset()

	patches.ApplyFunc(getImageSha, func(name string) (string, int, string, error) {
		return "sha256:child", 123, DockerV2Schema2MediaType, nil
	})

	var captured string
	patches.ApplyFunc(putManifests, func(manifests string, name string) error {
		captured = manifests
		return nil
	})

	err := CreateMultiArchImage([]ImageArch{
		{Name: "img:latest-amd64", OS: "linux", Architecture: "amd64"},
	}, "img:latest")
	assert.NoError(t, err)

	var out map[string]any
	assert.NoError(t, json.Unmarshal([]byte(captured), &out))
	assert.Equal(t, DockerV2ListMediaType, out["mediaType"])
}
