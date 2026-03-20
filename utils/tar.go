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

package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	goutils "go-frpc/utils"
	"io"
	"os"
	"path/filepath"

	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

// Tar 将 src 目录打包成 tar.gz 压缩包，dst 为压缩包的保存路径
func Tar(src, dst string, absolute bool) error {
	fw, err := createTarFile(dst)
	if err != nil {
		return err
	}
	defer fw.Close()

	gw, tw, err := createCompressedWriter(fw)
	if err != nil {
		return err
	}
	defer gw.Close()
	defer tw.Close()

	return packDirectory(src, absolute, tw)
}

func createTarFile(dst string) (*os.File, error) {
	fw, err := os.Create(dst)
	if err != nil {
		return nil, fmt.Errorf("failed to create tar file %s: %w", dst, err)
	}

	if err := fw.Chmod(DefaultFilePermission); err != nil {
		log.Warnf("failed to set file permission for %s: %s", dst, err.Error())
	}

	return fw, nil
}

func createCompressedWriter(fw *os.File) (*gzip.Writer, *tar.Writer, error) {
	gw := gzip.NewWriter(fw)
	tw := tar.NewWriter(gw)
	return gw, tw, nil
}

func packDirectory(src string, absolute bool, tw *tar.Writer) error {
	return walkWithHandler(src,
		func(fileName string, fi os.FileInfo) error {
			return packFile(src, fileName, fi, absolute, tw)
		},
		defaultErrorHandler,
	)
}

func walkWithHandler(root string, fn func(string, os.FileInfo) error, errFn func(error) error) error {
	if errFn == nil {
		errFn = func(err error) error { return err }
	}

	return filepath.Walk(root, func(fileName string, fi os.FileInfo, err error) error {
		if err != nil {
			return errFn(err)
		}
		return fn(fileName, fi)
	})
}

func defaultErrorHandler(err error) error {
	return err
}

func packFile(src, fileName string, fi os.FileInfo, absolute bool, tw *tar.Writer) error {
	hdr, err := createTarHeader(src, fileName, fi, absolute)
	if err != nil {
		return err
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return fmt.Errorf("failed to write tar header for %s: %w", fileName, err)
	}

	if fi.Mode().IsRegular() {
		if err := writeFileContent(fileName, tw); err != nil {
			return err
		}
	}

	log.Debugf("Packaged %s successfully\n", fileName)
	return nil
}

func createTarHeader(src, fileName string, fi os.FileInfo, absolute bool) (*tar.Header, error) {
	hdr, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return nil, fmt.Errorf("failed to create tar header for %s: %w", fileName, err)
	}

	if absolute {
		hdr.Name = filepath.ToSlash(fileName)
	} else {
		relPath, err := filepath.Rel(src, fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to get relative path for %s: %w", fileName, err)
		}
		hdr.Name = filepath.ToSlash(relPath)
	}

	return hdr, nil
}

func writeFileContent(fileName string, tw *tar.Writer) error {
	fr, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", fileName, err)
	}
	defer fr.Close()

	if _, err := io.Copy(tw, fr); err != nil {
		return fmt.Errorf("failed to copy file %s to tar: %w", fileName, err)
	}

	return nil
}

// UnTar extracts a tar archive to the specified destination directory
func UnTar(src, dst string) error {
	return goutils.UnTar(src, dst, false)
}
