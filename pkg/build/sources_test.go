/*
 *
 * Copyright (c) 2025 Bocloud Technologies Co., Ltd.
 * installer is licensed under Mulan PSL v2.
 * You can use this software according to the terms and conditions of the Mulan PSL v2.
 * You may obtain a copy of Mulan PSL v2 at:
 *          <http://license.coscl.org.cn/MulanPSL2>
 * THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
 * EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT
 * MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
 * See the Mulan PSL v2 for more details.
 *
 */

package build

import (
	"fmt"
	"os"
	"testing"

	"github.com/agiledragon/gomonkey/v2"
	"github.com/stretchr/testify/assert"

	"gopkg.openfuyao.cn/bkeadm/pkg/executor/exec"
	"gopkg.openfuyao.cn/bkeadm/pkg/global"
	"gopkg.openfuyao.cn/bkeadm/utils"
	"gopkg.openfuyao.cn/bkeadm/utils/log"
)

// TestDownloadUrlContent tests the downloadUrlContent function
// which downloads content from URL to local storage
func TestDownloadUrlContent(t *testing.T) {
	tests := []struct {
		name         string
		files        []File
		storagePath  string
		mockDownload func(string, string) error
		expectError  bool
	}{
		{
			name: "successful download",
			files: []File{
				{
					Address: "http://example.com",
					Files: []FileInfo{
						{FileName: "file1.tar.gz"},
					},
				},
			},
			storagePath: "/tmp/storage",
			mockDownload: func(url, target string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "download error",
			files: []File{
				{
					Address: "http://example.com",
					Files: []FileInfo{
						{FileName: "file1.tar.gz"},
					},
				},
			},
			storagePath: "/tmp/storage",
			mockDownload: func(url, target string) error {
				return fmt.Errorf("download error")
			},
			expectError: true,
		},
		{
			name:        "empty files",
			files:       []File{},
			storagePath: "/tmp/storage",
			mockDownload: func(url, target string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "address without trailing slash",
			files: []File{
				{
					Address: "http://example.com",
					Files: []FileInfo{
						{FileName: "file1.tar.gz"},
					},
				},
			},
			storagePath: "/tmp/storage",
			mockDownload: func(url, target string) error {
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.DownloadSignalFile, tt.mockDownload)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := downloadUrlContent(tt.files, tt.storagePath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDownloadFile tests the downloadFile function
// which orchestrates downloading files, charts, and patches
func TestDownloadFile(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	tests := []struct {
		name           string
		cfg            *BuildConfig
		mockBuildFiles func([]File, string, <-chan struct{}) error
		mockDownload   func([]File, string) error
		expectError    bool
	}{
		{
			name: "successful download",
			cfg: &BuildConfig{
				Files:   []File{{Address: "http://example.com", Files: []FileInfo{{FileName: "file1.tar.gz"}}}},
				Charts:  []File{{Address: "http://example.com/charts", Files: []FileInfo{{FileName: "charts.tar.gz"}}}},
				Patches: []File{{Address: "http://example.com/patches", Files: []FileInfo{{FileName: "patch1.tar.gz"}}}},
			},
			mockBuildFiles: func(files []File, storagePath string, stopChan <-chan struct{}) error {
				return nil
			},
			mockDownload: func(files []File, storagePath string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "files build error",
			cfg: &BuildConfig{
				Files: []File{{Address: "http://example.com", Files: []FileInfo{{FileName: "file1.tar.gz"}}}},
			},
			mockBuildFiles: func(files []File, storagePath string, stopChan <-chan struct{}) error {
				return fmt.Errorf("files build error")
			},
			mockDownload: func(files []File, storagePath string) error {
				return nil
			},
			expectError: true,
		},
		{
			name: "charts build error",
			cfg: &BuildConfig{
				Files:  []File{{Address: "http://example.com", Files: []FileInfo{{FileName: "file1.tar.gz"}}}},
				Charts: []File{{Address: "http://example.com/charts", Files: []FileInfo{{FileName: "charts.tar.gz"}}}},
			},
			mockBuildFiles: func(files []File, storagePath string, stopChan <-chan struct{}) error {
				if storagePath == tmpPackagesCharts {
					return fmt.Errorf("charts build error")
				}
				return nil
			},
			mockDownload: func(files []File, storagePath string) error {
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(buildFiles, tt.mockBuildFiles)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(downloadUrlContent, tt.mockDownload)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := downloadFile(tt.cfg, stopChan)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildRpms tests the buildRpms function
// which builds RPM packages
func TestBuildRpms(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	tests := []struct {
		name           string
		cfg            *BuildConfig
		mockDownload   func(*BuildConfig, <-chan struct{}) error
		mockFileAdapt  func() error
		mockBuildChart func() error
		mockBuildRpm   func() error
		mockSyncPkg    func(string, []string, []string, []string, []string) error
		mockTarGz      func(string, string) error
		expectError    bool
	}{
		{
			name: "successful build",
			cfg: &BuildConfig{
				Rpms: []Rpm{
					{
						Address:            "http://example.com",
						System:             []string{"CentOS"},
						SystemVersion:      []string{"7"},
						SystemArchitecture: []string{"amd64"},
						Directory:          []string{"docker-ce"},
					},
				},
			},
			mockDownload:   func(cfg *BuildConfig, stopChan <-chan struct{}) error { return nil },
			mockFileAdapt:  func() error { return nil },
			mockBuildChart: func() error { return nil },
			mockBuildRpm:   func() error { return nil },
			mockSyncPkg: func(url string, systems, versions, architectures, directory []string) error {
				return nil
			},
			mockTarGz:   func(prefix, target string) error { return nil },
			expectError: false,
		},
		{
			name: "download error",
			cfg: &BuildConfig{
				Rpms: []Rpm{{Address: "http://example.com", System: []string{"CentOS"}, SystemVersion: []string{"7"}, SystemArchitecture: []string{"amd64"}, Directory: []string{"docker-ce"}}},
			},
			mockDownload: func(cfg *BuildConfig, stopChan <-chan struct{}) error {
				return fmt.Errorf("download error")
			},
			mockFileAdapt:  func() error { return nil },
			mockBuildChart: func() error { return nil },
			mockBuildRpm:   func() error { return nil },
			mockSyncPkg: func(url string, systems, versions, architectures, directory []string) error {
				return nil
			},
			mockTarGz:   func(prefix, target string) error { return nil },
			expectError: true,
		},
		{
			name: "sync package error",
			cfg: &BuildConfig{
				Rpms: []Rpm{{Address: "http://example.com", System: []string{"CentOS"}, SystemVersion: []string{"7"}, SystemArchitecture: []string{"amd64"}, Directory: []string{"docker-ce"}}},
			},
			mockDownload:   func(cfg *BuildConfig, stopChan <-chan struct{}) error { return nil },
			mockFileAdapt:  func() error { return nil },
			mockBuildChart: func() error { return nil },
			mockBuildRpm:   func() error { return nil },
			mockSyncPkg: func(url string, systems, versions, architectures, directory []string) error {
				return fmt.Errorf("sync error")
			},
			mockTarGz:   func(prefix, target string) error { return nil },
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(downloadFile, tt.mockDownload)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(fileVersionAdaptation, tt.mockFileAdapt)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(buildFileChart, tt.mockBuildChart)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(buildFileRpm, tt.mockBuildRpm)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(syncPackage, tt.mockSyncPkg)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.TarGZ, tt.mockTarGz)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := buildRpms(tt.cfg, stopChan)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildFiles tests the buildFiles function
// which builds files from URL
func TestBuildFiles(t *testing.T) {
	stopChan := make(chan struct{})
	defer close(stopChan)

	tests := []struct {
		name         string
		files        []File
		storagePath  string
		mockDownload func(string, string) error
		expectError  bool
	}{
		{
			name: "successful build",
			files: []File{
				{
					Address: "http://example.com",
					Files:   []FileInfo{{FileName: "file1.tar.gz"}},
				},
			},
			storagePath: "/tmp/storage",
			mockDownload: func(url, target string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "download error",
			files: []File{
				{
					Address: "http://example.com",
					Files:   []FileInfo{{FileName: "file1.tar.gz"}},
				},
			},
			storagePath: "/tmp/storage",
			mockDownload: func(url, target string) error {
				return fmt.Errorf("download error")
			},
			expectError: true,
		},
		{
			name:         "external stop",
			files:        []File{{Address: "http://example.com", Files: []FileInfo{{FileName: "file1.tar.gz"}}}},
			storagePath:  "/tmp/storage",
			mockDownload: func(url, target string) error { return nil },
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			localStopChan := make(chan struct{})
			if tt.name == "external stop" {
				close(localStopChan)
			}

			patches := gomonkey.ApplyFunc(utils.DownloadFile, tt.mockDownload)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := buildFiles(tt.files, tt.storagePath, localStopChan)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestSyncPackage tests the syncPackage function
// which syncs packages across systems and versions
func TestSyncPackage(t *testing.T) {
	tests := []struct {
		name              string
		url               string
		systems           []string
		versions          []string
		architectures     []string
		directory         []string
		mockProcessSystem func(string, string, string, []string, []string) error
		expectError       bool
	}{
		{
			name:              "successful sync",
			url:               "http://example.com/",
			systems:           []string{"CentOS"},
			versions:          []string{"7"},
			architectures:     []string{"amd64"},
			directory:         []string{"docker-ce"},
			mockProcessSystem: func(url, system, version string, archs, dirs []string) error { return nil },
			expectError:       false,
		},
		{
			name:          "process system error",
			url:           "http://example.com/",
			systems:       []string{"CentOS"},
			versions:      []string{"7"},
			architectures: []string{"amd64"},
			directory:     []string{"docker-ce"},
			mockProcessSystem: func(url, system, version string, archs, dirs []string) error {
				return fmt.Errorf("process system error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(processSystemVersion, tt.mockProcessSystem)
			defer patches.Reset()

			err := syncPackage(tt.url, tt.systems, tt.versions, tt.architectures, tt.directory)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProcessSystemVersion tests the processSystemVersion function
// which processes a specific system version
func TestProcessSystemVersion(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		system          string
		version         string
		architectures   []string
		directory       []string
		mockProcessArch func(string, string, string, string, []string) error
		expectError     bool
	}{
		{
			name:          "successful process",
			url:           "http://example.com/",
			system:        "CentOS",
			version:       "7",
			architectures: []string{"amd64"},
			directory:     []string{"docker-ce"},
			mockProcessArch: func(url, system, version, arch string, dirs []string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "process architecture error",
			url:           "http://example.com/",
			system:        "CentOS",
			version:       "7",
			architectures: []string{"amd64"},
			directory:     []string{"docker-ce"},
			mockProcessArch: func(url, system, version, arch string, dirs []string) error {
				return fmt.Errorf("process arch error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(processArchitecture, tt.mockProcessArch)
			defer patches.Reset()

			err := processSystemVersion(tt.url, tt.system, tt.version, tt.architectures, tt.directory)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProcessArchitecture tests the processArchitecture function
// which processes a specific architecture
func TestProcessArchitecture(t *testing.T) {
	tests := []struct {
		name                string
		url                 string
		system              string
		version             string
		arch                string
		directories         []string
		mockDownloadPackage func(string, string, string, string, string) error
		mockExecuteCommand  func(string, ...string) (string, error)
		expectError         bool
	}{
		{
			name:                "successful process",
			url:                 "http://example.com/",
			system:              "CentOS",
			version:             "7",
			arch:                "amd64",
			directories:         []string{"docker-ce"},
			mockDownloadPackage: func(url, system, version, arch, dir string) error { return nil },
			mockExecuteCommand: func(command string, args ...string) (string, error) {
				return "", nil
			},
			expectError: false,
		},
		{
			name:        "download package error",
			url:         "http://example.com/",
			system:      "CentOS",
			version:     "7",
			arch:        "amd64",
			directories: []string{"docker-ce"},
			mockDownloadPackage: func(url, system, version, arch, dir string) error {
				return fmt.Errorf("download error")
			},
			mockExecuteCommand: func(command string, args ...string) (string, error) {
				return "", nil
			},
			expectError: true,
		},
		{
			name:                "execute command error",
			url:                 "http://example.com/",
			system:              "CentOS",
			version:             "7",
			arch:                "amd64",
			directories:         []string{"docker-ce"},
			mockDownloadPackage: func(url, system, version, arch, dir string) error { return nil },
			mockExecuteCommand: func(command string, args ...string) (string, error) {
				return "", fmt.Errorf("command error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(downloadPackageDirectory, tt.mockDownloadPackage)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc((*exec.CommandExecutor).ExecuteCommandWithOutput, func(c *exec.CommandExecutor, command string, args ...string) (string, error) {
				return tt.mockExecuteCommand(command, args...)
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := processArchitecture(tt.url, tt.system, tt.version, tt.arch, tt.directories)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestDownloadPackageDirectory tests the downloadPackageDirectory function
// which downloads a package directory
func TestDownloadPackageDirectory(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		system          string
		version         string
		arch            string
		directory       string
		mockMkdirAll    func(string, os.FileMode) error
		mockDownloadAll func(string, string) error
		expectError     bool
	}{
		{
			name:            "successful download",
			url:             "http://example.com/",
			system:          "CentOS",
			version:         "7",
			arch:            "amd64",
			directory:       "docker-ce",
			mockMkdirAll:    func(path string, mode os.FileMode) error { return nil },
			mockDownloadAll: func(url, dir string) error { return nil },
			expectError:     false,
		},
		{
			name:            "mkdir error",
			url:             "http://example.com/",
			system:          "CentOS",
			version:         "7",
			arch:            "amd64",
			directory:       "docker-ce",
			mockMkdirAll:    func(path string, mode os.FileMode) error { return fmt.Errorf("mkdir error") },
			mockDownloadAll: func(url, dir string) error { return nil },
			expectError:     true,
		},
		{
			name:            "download all error",
			url:             "http://example.com/",
			system:          "CentOS",
			version:         "7",
			arch:            "amd64",
			directory:       "docker-ce",
			mockMkdirAll:    func(path string, mode os.FileMode) error { return nil },
			mockDownloadAll: func(url, dir string) error { return fmt.Errorf("download error") },
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.MkdirAll, tt.mockMkdirAll)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.DownloadAllFiles, tt.mockDownloadAll)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := downloadPackageDirectory(tt.url, tt.system, tt.version, tt.arch, tt.directory)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestFindBkeBinaries tests the findBkeBinaries function
// which finds BKE binary files
func TestFindBkeBinaries(t *testing.T) {
	tests := []struct {
		name        string
		mockReadDir func(string) ([]os.DirEntry, error)
		expectError bool
		expectCount int
	}{
		{
			name: "find bke binary",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "bke"},
				}, nil
			},
			expectError: false,
			expectCount: 1,
		},
		{
			name: "find bkeadm binary",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "bkeadm_linux_amd64"},
				}, nil
			},
			expectError: false,
			expectCount: 1,
		},
		{
			name: "find multiple binaries",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "bke"},
					&mockDirEntry{name: "bke_amd64"},
				}, nil
			},
			expectError: false,
			expectCount: 2,
		},
		{
			name: "read dir error",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, fmt.Errorf("read dir error")
			},
			expectError: true,
			expectCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			result, err := findBkeBinaries()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectCount, len(result))
			}
		})
	}
}

// TestInstallSingleBkeBinary tests the installSingleBkeBinary function
// which installs a single BKE binary
func TestInstallSingleBkeBinary(t *testing.T) {
	tests := []struct {
		name         string
		bkeName      string
		mockCopyFile func(string, string) error
		mockChmod    func(string, os.FileMode) error
		mockExecute  func(string, string, ...string) (string, error)
		expectError  bool
	}{
		{
			name:    "successful install",
			bkeName: "bke",
			mockCopyFile: func(src, dst string) error {
				return nil
			},
			mockChmod: func(name string, mode os.FileMode) error {
				return nil
			},
			mockExecute: func(command, shell string, args ...string) (string, error) {
				return "v1.0.0", nil
			},
			expectError: false,
		},
		{
			name:    "copy file error",
			bkeName: "bke",
			mockCopyFile: func(src, dst string) error {
				return fmt.Errorf("copy error")
			},
			mockChmod: func(name string, mode os.FileMode) error {
				return nil
			},
			mockExecute: func(command, shell string, args ...string) (string, error) {
				return "", nil
			},
			expectError: true,
		},
		{
			name:    "chmod error",
			bkeName: "bke",
			mockCopyFile: func(src, dst string) error {
				return nil
			},
			mockChmod: func(name string, mode os.FileMode) error {
				return fmt.Errorf("chmod error")
			},
			mockExecute: func(command, shell string, args ...string) (string, error) {
				return "", nil
			},
			expectError: true,
		},
		{
			name:    "execute command error",
			bkeName: "bke",
			mockCopyFile: func(src, dst string) error {
				return nil
			},
			mockChmod: func(name string, mode os.FileMode) error {
				return nil
			},
			mockExecute: func(command, shell string, args ...string) (string, error) {
				return "", fmt.Errorf("execute error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.CopyFile, tt.mockCopyFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Chmod, tt.mockChmod)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc((*exec.CommandExecutor).ExecuteCommandWithOutput, func(c *exec.CommandExecutor, command string, args ...string) (string, error) {
				return tt.mockExecute(command, "", args...)
			})
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			version, err := installSingleBkeBinary(tt.bkeName)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, version)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, version)
			}
		})
	}
}

// TestBuildBkeBinary tests the buildBkeBinary function
// which builds the BKE binary
func TestBuildBkeBinary(t *testing.T) {
	tests := []struct {
		name                string
		mockFindBinaries    func() ([]string, error)
		mockInstallSingle   func(string) (string, error)
		mockInstallMultiple func([]string) (string, error)
		expectError         bool
	}{
		{
			name: "single binary found",
			mockFindBinaries: func() ([]string, error) {
				return []string{"bke"}, nil
			},
			mockInstallSingle: func(name string) (string, error) {
				return "v1.0.0", nil
			},
			mockInstallMultiple: func(list []string) (string, error) {
				return "", nil
			},
			expectError: false,
		},
		{
			name: "multiple binaries found",
			mockFindBinaries: func() ([]string, error) {
				return []string{"bke", "bke_amd64"}, nil
			},
			mockInstallSingle: func(name string) (string, error) {
				return "", nil
			},
			mockInstallMultiple: func(list []string) (string, error) {
				return "v1.0.0", nil
			},
			expectError: false,
		},
		{
			name: "find binaries error",
			mockFindBinaries: func() ([]string, error) {
				return nil, fmt.Errorf("find error")
			},
			mockInstallSingle: func(name string) (string, error) {
				return "", nil
			},
			mockInstallMultiple: func(list []string) (string, error) {
				return "", nil
			},
			expectError: true,
		},
		{
			name: "no binaries found",
			mockFindBinaries: func() ([]string, error) {
				return []string{}, nil
			},
			mockInstallSingle: func(name string) (string, error) {
				return "", nil
			},
			mockInstallMultiple: func(list []string) (string, error) {
				return "", nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(findBkeBinaries, tt.mockFindBinaries)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(installSingleBkeBinary, tt.mockInstallSingle)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(installMultipleBkeBinaries, tt.mockInstallMultiple)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			version, err := buildBkeBinary()

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, version)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, version)
			}
		})
	}
}

// TestMoveFilesFromSubfolder tests the moveFilesFromSubfolder function
// which moves files from subfolder to parent folder
func TestMoveFilesFromSubfolder(t *testing.T) {
	tests := []struct {
		name          string
		parentPath    string
		subfolderPath string
		mockReadDir   func(string) ([]os.DirEntry, error)
		mockStat      func(string) (os.FileInfo, error)
		mockCopyFile  func(string, string) error
		mockRemove    func(string) error
		expectError   bool
	}{
		{
			name:          "successful move",
			parentPath:    "/tmp/parent",
			subfolderPath: "/tmp/parent/sub",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "file1.txt"},
				}, nil
			},
			mockStat: func(path string) (os.FileInfo, error) {
				return nil, os.ErrNotExist
			},
			mockCopyFile: func(src, dst string) error {
				return nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "file exists skip",
			parentPath:    "/tmp/parent",
			subfolderPath: "/tmp/parent/sub",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "file1.txt"},
				}, nil
			},
			mockStat: func(path string) (os.FileInfo, error) {
				return nil, nil
			},
			mockCopyFile: func(src, dst string) error {
				return nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:          "read dir error",
			parentPath:    "/tmp/parent",
			subfolderPath: "/tmp/parent/sub",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, fmt.Errorf("read error")
			},
			mockStat: func(path string) (os.FileInfo, error) {
				return nil, nil
			},
			mockCopyFile: func(src, dst string) error {
				return nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Stat, tt.mockStat)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(utils.CopyFile, tt.mockCopyFile)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Remove, tt.mockRemove)
			defer patches.Reset()

			err := moveFilesFromSubfolder(tt.parentPath, tt.subfolderPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestRemoveDir tests the removeDir function
// which removes a directory recursively
func TestRemoveDir(t *testing.T) {
	tests := []struct {
		name        string
		dirPath     string
		mockReadDir func(string) ([]os.DirEntry, error)
		mockRemove  func(string) error
		expectError bool
	}{
		{
			name:    "successful remove",
			dirPath: "/tmp/test",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{}, nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:    "contains file",
			dirPath: "/tmp/test",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "file.txt"},
				}, nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:    "contains directory",
			dirPath: "/tmp/test",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				if path == "/tmp/test" {
					return []os.DirEntry{
						&mockDirEntry{name: "subdir", isDir: true},
					}, nil
				}
				return []os.DirEntry{}, nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:    "read dir error",
			dirPath: "/tmp/test",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, fmt.Errorf("read error")
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Remove, tt.mockRemove)
			defer patches.Reset()

			err := removeDir(tt.dirPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestProcessFolder tests the processFolder function
// which processes a folder by moving files from subfolders
func TestProcessFolder(t *testing.T) {
	tests := []struct {
		name          string
		rootPath      string
		mockReadDir   func(string) ([]os.DirEntry, error)
		mockMoveFiles func(string, string) error
		mockRemoveDir func(string) error
		expectError   bool
	}{
		{
			name:     "successful process",
			rootPath: "/tmp/root",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "subdir", isDir: true},
				}, nil
			},
			mockMoveFiles: func(parent, sub string) error {
				return nil
			},
			mockRemoveDir: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:     "move files error",
			rootPath: "/tmp/root",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "subdir", isDir: true},
				}, nil
			},
			mockMoveFiles: func(parent, sub string) error {
				return fmt.Errorf("move error")
			},
			mockRemoveDir: func(path string) error {
				return nil
			},
			expectError: true,
		},
		{
			name:     "read dir error",
			rootPath: "/tmp/root",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return nil, fmt.Errorf("read error")
			},
			mockMoveFiles: func(parent, sub string) error {
				return nil
			},
			mockRemoveDir: func(path string) error {
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(moveFilesFromSubfolder, tt.mockMoveFiles)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(removeDir, tt.mockRemoveDir)
			defer patches.Reset()

			err := processFolder(tt.rootPath)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildFileChart tests the buildFileChart function
// which builds chart files
func TestBuildFileChart(t *testing.T) {
	tests := []struct {
		name              string
		mockReadDirCharts func(string) ([]os.DirEntry, error)
		mockReadDirFiles  func(string) ([]os.DirEntry, error)
		mockUnTarGz       func(string, string) error
		mockRemove        func(string) error
		mockRePackage     func(string, string, string) error
		expectError       bool
	}{
		{
			name: "successful build",
			mockReadDirCharts: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "chart1"},
				}, nil
			},
			mockReadDirFiles: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: utils.ChartFile},
				}, nil
			},
			mockUnTarGz: func(src, dst string) error {
				return nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			mockRePackage: func(srcDir, subDir, target string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "empty charts directory",
			mockReadDirCharts: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{}, nil
			},
			mockReadDirFiles: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{}, nil
			},
			mockUnTarGz: func(src, dst string) error {
				return nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			mockRePackage: func(srcDir, subDir, target string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "repackage error",
			mockReadDirCharts: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "chart1"},
				}, nil
			},
			mockReadDirFiles: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: utils.ChartFile},
				}, nil
			},
			mockUnTarGz: func(src, dst string) error {
				return nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			mockRePackage: func(srcDir, subDir, target string) error {
				return fmt.Errorf("repackage error")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDirCharts)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDirFiles)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.UnTarGZ, tt.mockUnTarGz)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Remove, tt.mockRemove)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(rePackageChart, tt.mockRePackage)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := buildFileChart()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestBuildFileRpm tests the buildFileRpm function
// which builds RPM files
func TestBuildFileRpm(t *testing.T) {
	tests := []struct {
		name        string
		mockReadDir func(string) ([]os.DirEntry, error)
		mockUnTarGz func(string, string) error
		mockRemove  func(string) error
		expectError bool
	}{
		{
			name: "RPM file found and processed",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: utils.RPMDataFile},
				}, nil
			},
			mockUnTarGz: func(src, dst string) error {
				return nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "RPM file not found",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: "other.tar.gz"},
				}, nil
			},
			mockUnTarGz: func(src, dst string) error {
				return nil
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: false,
		},
		{
			name: "untar error",
			mockReadDir: func(path string) ([]os.DirEntry, error) {
				return []os.DirEntry{
					&mockDirEntry{name: utils.RPMDataFile},
				}, nil
			},
			mockUnTarGz: func(src, dst string) error {
				return fmt.Errorf("untar error")
			},
			mockRemove: func(path string) error {
				return nil
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(os.ReadDir, tt.mockReadDir)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.UnTarGZ, tt.mockUnTarGz)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(os.Remove, tt.mockRemove)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			err := buildFileRpm()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestInstallMultipleBkeBinaries tests the installMultipleBkeBinaries function
// which installs multiple BKE binaries
func TestInstallMultipleBkeBinaries(t *testing.T) {
	tests := []struct {
		name          string
		bkeBinaryList []string
		mockCopyFile  func(string, string) error
		mockChmod     func(string, os.FileMode) error
		mockExecute   func(string, string, ...string) (string, error)
		expectError   bool
		expectVersion string
	}{
		{
			name:          "copy file error",
			bkeBinaryList: []string{"bke"},
			mockCopyFile: func(src, dst string) error {
				return fmt.Errorf("copy error")
			},
			mockChmod: func(name string, mode os.FileMode) error {
				return nil
			},
			mockExecute: func(command, shell string, args ...string) (string, error) {
				return "", nil
			},
			expectError:   true,
			expectVersion: "",
		},
		{
			name:          "chmod error",
			bkeBinaryList: []string{"bke"},
			mockCopyFile: func(src, dst string) error {
				return nil
			},
			mockChmod: func(name string, mode os.FileMode) error {
				return fmt.Errorf("chmod error")
			},
			mockExecute: func(command, shell string, args ...string) (string, error) {
				return "", nil
			},
			expectError:   true,
			expectVersion: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(utils.CopyFile, tt.mockCopyFile)
			defer patches.Reset()

			patches.ApplyFunc(os.Chmod, tt.mockChmod)
			defer patches.Reset()

			patches.ApplyFunc((*exec.CommandExecutor).ExecuteCommandWithOutput, func(c *exec.CommandExecutor, command string, args ...string) (string, error) {
				return tt.mockExecute(command, "", args...)
			})
			defer patches.Reset()

			patches.ApplyFunc(log.BKEFormat, func(level, msg string) {})
			defer patches.Reset()

			patches.ApplyGlobalVar(&tmpPackagesFiles, "/tmp/packages/files")
			defer patches.Reset()

			patches.ApplyGlobalVar(&usrBin, "/usr/bin")
			defer patches.Reset()

			version, err := installMultipleBkeBinaries(tt.bkeBinaryList)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, version)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectVersion, version)
			}
		})
	}
}

// TestRePackageChart tests the rePackageChart function
// which repackages a chart directory into a tar.gz file
func TestRePackageChart(t *testing.T) {
	tests := []struct {
		name              string
		srcDir            string
		subDir            string
		target            string
		mockProcessFolder func(string) error
		mockTarGZWithDir  func(string, string, string) error
		expectError       bool
	}{
		{
			name:   "successful repackage",
			srcDir: "/tmp/charts",
			subDir: "charts",
			target: "/tmp/charts.tar.gz",
			mockProcessFolder: func(path string) error {
				return nil
			},
			mockTarGZWithDir: func(srcDir, subDir, target string) error {
				return nil
			},
			expectError: false,
		},
		{
			name:   "process folder error",
			srcDir: "/tmp/charts",
			subDir: "charts",
			target: "/tmp/charts.tar.gz",
			mockProcessFolder: func(path string) error {
				return fmt.Errorf("process folder error")
			},
			mockTarGZWithDir: func(srcDir, subDir, target string) error {
				return nil
			},
			expectError: true,
		},
		{
			name:   "tar gzip error",
			srcDir: "/tmp/charts",
			subDir: "charts",
			target: "/tmp/charts.tar.gz",
			mockProcessFolder: func(path string) error {
				return nil
			},
			mockTarGZWithDir: func(srcDir, subDir, target string) error {
				return fmt.Errorf("tar gzip error")
			},
			expectError: true,
		},
		{
			name:   "empty subdir repackage",
			srcDir: "/tmp/charts",
			subDir: "",
			target: "/tmp/charts.tar.gz",
			mockProcessFolder: func(path string) error {
				return nil
			},
			mockTarGZWithDir: func(srcDir, subDir, target string) error {
				return nil
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			patches := gomonkey.ApplyFunc(processFolder, tt.mockProcessFolder)
			defer patches.Reset()

			patches = gomonkey.ApplyFunc(global.TarGZWithDir, tt.mockTarGZWithDir)
			defer patches.Reset()

			err := rePackageChart(tt.srcDir, tt.subDir, tt.target)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
