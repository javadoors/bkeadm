/*
 * Copyright (c) 2025 Bocloud Technologies Co., Ltd.
 * installer is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          <http://license.coscl.org.cn/MulanPSL2>
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 */

package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	testOneValue = 1
)

func TestSyncImageSingleArch(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		target      string
		arch        string
		expectPanic bool
	}{
		{
			name:        "pull image error causes panic without mock",
			source:      "source.example.com/image:v1.0",
			target:      "target.example.com/image:v1.0",
			arch:        "amd64",
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.Panics(t, func() {
					syncImage(tt.source, tt.target, []string{tt.arch})
				})
			}
		})
	}
}

func TestSyncImageMultiArch(t *testing.T) {
	tests := []struct {
		name        string
		source      string
		target      string
		arch        []string
		expectPanic bool
	}{
		{
			name:        "multi arch sync causes panic without mock",
			source:      "source.example.com/image:v1.0",
			target:      "target.example.com/image:v1.0",
			arch:        []string{"amd64", "arm64"},
			expectPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				assert.Panics(t, func() {
					syncImage(tt.source, tt.target, tt.arch)
				})
			}
		})
	}
}

func TestNeedRemoveImageVariable(t *testing.T) {
	needRemoveImage = []string{}
	assert.NotPanics(t, func() {
		needRemoveImage = append(needRemoveImage, "test-image:v1.0")
	})
	assert.Len(t, needRemoveImage, testOneValue)
}
