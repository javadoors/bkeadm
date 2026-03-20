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

if [ -n "$DEBUG" ]; then
	set -x
fi

set -o errexit
set -o nounset
set -o pipefail

declare -a mandatory
mandatory=(
  ARCH
  COMMIT_ID
  VERSION
  TIMESTAMP
)

missing=false
for var in "${mandatory[@]}"; do
  if [[ -z "${!var:-}" ]]; then
    echo "Environment variable $var must be set"
    missing=true
  fi
done

export CGO_ENABLED=0
export GOARCH=${ARCH}

if [ "$missing" = true ]; then
  exit 1
fi

go build -tags "remote exclude_graphdriver_btrfs btrfs_noversion exclude_graphdriver_devicemapper containers_image_openpgp" \
-ldflags="-s -w -X main.gitCommitId=${COMMIT_ID} -X main.architecture=`go env GOHOSTOS`/`go env GOHOSTARCH` \
-X main.timestamp=${TIMESTAMP} -X main.ver=${VERSION}" -o bin/bke_${ARCH} .
