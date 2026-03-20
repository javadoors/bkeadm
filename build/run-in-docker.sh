#!/bin/bash
###############################################################
# Copyright (c) 2025 Bocloud Technologies Co., Ltd.
# installer is licensed under Mulan PSL v2.
# You can use this software according to the terms and conditions of the Mulan PSL v2.
# You may obtain a copy of Mulan PSL v2 at:
#          http://license.coscl.org.cn/MulanPSL2
# THIS SOFTWARE IS PROVIDED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OF ANY KIND,
# EITHER EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO NON-INFRINGEMENT,
# MERCHANTABILITY OR FIT FOR A PARTICULAR PURPOSE.
# See the Mulan PSL v2 for more details.
###############################################################

set -o errexit
set -o nounset
set -o pipefail

DOCKER_OPTS=${DOCKER_OPTS:-}
BKE_ROOT=$(cd $(dirname "${BASH_SOURCE}")/.. && pwd -P)
FLAGS=$@
IMAGE=registry.cn-hangzhou.aliyuncs.com/bocloud/golang:1.24.5
PKG=bke/bkeadm
ARCH=${ARCH:-}
if [[ -z "$ARCH" ]]; then
  ARCH=$(go env GOARCH)
fi

# create output directory as current user to avoid problem with docker.
mkdir -p "${BKE_ROOT}/bin"

docker run                                            \
  --platform ${ARCH}                                  \
  --tty                                               \
  --rm                                                \
  ${DOCKER_OPTS}                                      \
  -e GOFLAGS="-buildvcs=false"                        \
  -e GOMODCACHE=/go/src/${PKG}/vendor                 \
  -e GOPROXY="https://goproxy.cn,direct"              \
  -v "${BKE_ROOT}:/go/src/${PKG}"                     \
  -w "/go/src/${PKG}"                                 \
  ${IMAGE} /bin/bash -c "${FLAGS}"
