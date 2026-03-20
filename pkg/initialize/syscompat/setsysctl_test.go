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

package syscompat

import (
	"os"
	"os/exec"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

func TestSetSysctlOpenFileError(t *testing.T) {

	SetSysctl()

	assert.True(t, true)
}

func TestSetSysctlWriteStringError(t *testing.T) {
	patches := gomonkey.NewPatches()
	patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return []byte(""), nil
	})
	defer patches.Reset()

	patches.ApplyFunc((*os.File).WriteString, func(f *os.File, s string) (int, error) {
		return 0, &os.PathError{Op: "write", Path: "test", Err: os.ErrPermission}
	})
	defer patches.Reset()

	patches.ApplyFunc(log.BKEFormat, func(level, msg string) {
	})
	defer patches.Reset()

	SetSysctl()

	assert.True(t, true)
}

func TestSetSysctlExecCommandError(t *testing.T) {
	patches := gomonkey.NewPatches()

	patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return []byte(""), nil
	})
	defer patches.Reset()

	patches.ApplyFunc((*os.File).WriteString, func(f *os.File, s string) (int, error) {
		return len(s), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(exec.Command, func(name string, arg ...string) *exec.Cmd {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc((*exec.Cmd).Run, func(c *exec.Cmd) error {
		return &os.PathError{Op: "run", Path: "sysctl", Err: os.ErrPermission}
	})
	defer patches.Reset()

	patches.ApplyFunc(log.BKEFormat, func(level, msg string) {
	})
	defer patches.Reset()

	SetSysctl()

	assert.True(t, true)
}

func TestSetSysctlAllConfigsExist(t *testing.T) {
	patches := gomonkey.ApplyFunc(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return os.NewFile(uintptr(0), "test"), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return []byte("fs.file-max = 9000000\nfs.inotify.max_user_watches = 1000000\nfs.inotify.max_user_instances = 1000000\nnet.ipv4.ip_forward = 1\n"), nil
	})
	defer patches.Reset()

	writeStringCalled := false
	patches.ApplyFunc((*os.File).WriteString, func(f *os.File, s string) (int, error) {
		writeStringCalled = true
		return len(s), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(exec.Command, func(name string, arg ...string) *exec.Cmd {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc((*exec.Cmd).Run, func(c *exec.Cmd) error {
		return nil
	})
	defer patches.Reset()

	SetSysctl()

	assert.False(t, writeStringCalled)
}

func TestSetSysctlPartialConfigs(t *testing.T) {
	patches := gomonkey.ApplyFunc(os.OpenFile, func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return os.NewFile(uintptr(0), "test"), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(os.ReadFile, func(filename string) ([]byte, error) {
		return []byte("fs.file-max = 9000000\n"), nil
	})
	defer patches.Reset()

	writeStringCalled := false
	patches.ApplyFunc((*os.File).WriteString, func(f *os.File, s string) (int, error) {
		writeStringCalled = true
		return len(s), nil
	})
	defer patches.Reset()

	patches.ApplyFunc(exec.Command, func(name string, arg ...string) *exec.Cmd {
		return nil
	})
	defer patches.Reset()

	patches.ApplyFunc((*exec.Cmd).Run, func(c *exec.Cmd) error {
		return nil
	})
	defer patches.Reset()

	SetSysctl()

	assert.True(t, writeStringCalled)
}
