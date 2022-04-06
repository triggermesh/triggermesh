#!/usr/bin/env bash
#
# This script is for generating API reference docs for Knative components.


# Copyright 2018 Knative authors
#
# Based on the Knative same file at https://github.com/knative/docs/
#
# Copyright 2022 TriggerMesh Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

SCRIPTDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
[[ -n "${DEBUG:-}" ]] && set -x

OUTPUT_DIR="${OUTPUT_DIR:-$SCRIPTDIR/../output}"
TEMPLATE_DIR="${TEMPLATE_DIR:-$SCRIPTDIR/../hack/api-docs-template}"

REFDOCS_PKG="github.com/ahmetb/gen-crd-api-reference-docs"
REFDOCS_REPO="https://${REFDOCS_PKG}.git"
REFDOCS_VER="v0.3.0"

TRIGGERMESH_REPO="github.com/triggermesh/triggermesh"
TRIGGERMESH_COMMIT="${TRIGGERMESH_COMMIT:-main}"
TRIGGERMESH_OUTPUT_FILE_PREFIX=""

cleanup_refdocs_root=
cleanup_repo_clone_root=

trap cleanup EXIT

log() {
    echo "$@" >&2
}

fail() {
    log "error: $*"
    exit 1
}

install_go_bin() {
    local pkg
    pkg="$1"
    go install "$pkg"
    # will be downloaded to "$(go env GOPATH)/bin/$(basename $pkg)"
}

repo_tarball_url() {
    local repo commit
    repo="$1"
    commit="$2"
    echo "https://$repo/archive/$commit.tar.gz"
}

clone_at_commit() {
    local repo commit dest
    repo="$1"
    commit="$2"
    dest="$3"
    mkdir -p "${dest}"
    git clone "${repo}" "${dest}"
    git --git-dir="${dest}/.git" --work-tree="${dest}" checkout --detach --quiet "${commit}"
}

gen_refdocs() {
    local refdocs_bin gopath out_file repo_root api_dir
    refdocs_bin="$1"
    gopath="$2"
    template_dir="$3"
    out_file="$4"
    repo_root="$5"
    api_dir="$6"

    (
        cd "${repo_root}"
        env GOPATH="${gopath}" "${refdocs_bin}" \
            -out-file "${out_file}" \
            -api-dir "${api_dir}" \
            -template-dir "${template_dir}" \
            -config "${SCRIPTDIR}/reference-docs-gen-config.json"
    )
}

cleanup() {
    if [ -d "${cleanup_refdocs_root}" ]; then
        echo "Cleaning up tmp directory: ${cleanup_refdocs_root}"
        rm -rf -- "${cleanup_refdocs_root}"
    fi

    chmod -R 755 "${cleanup_repo_clone_root}"

    if [ -d "${cleanup_repo_clone_root}" ]; then
        echo "Cleaning up tmp directory: ${cleanup_repo_clone_root}"
        rm -rf -- "${cleanup_repo_clone_root}"
    fi
}

# The 'extglob' flag is used by the Bash parser. Functions are parsed ahead of execution, therefore the flag must be set
# before the code containing extended globs is parsed.
# See also https://stackoverflow.com/a/49283991
shopt -s extglob

main() {
    if [[ -n "${GOPATH:-}" ]]; then
        fail "GOPATH should not be set."
    fi
    if ! command -v "go" 1>/dev/null ; then
        fail "\"go\" is not in PATH"
    fi
    if ! command -v "git" 1>/dev/null ; then
        fail "\"git\" is not in PATH"
    fi

    # install and place the refdocs tool
    local refdocs_bin refdocs_bin_expected refdocs_dir template_dir
    refdocs_dir="$(mktemp -d)"
    cleanup_refdocs_root="${refdocs_dir}"
    # clone repo for ./templates
    git clone --quiet --depth=1 "${REFDOCS_REPO}" "${refdocs_dir}"

    template_dir="${TEMPLATE_DIR}"

    # install bin
    install_go_bin "${REFDOCS_PKG}@${REFDOCS_VER}"

    # move bin to final location
    refdocs_bin="${refdocs_dir}/refdocs"
    refdocs_bin_expected="$(go env GOPATH)/bin/$(basename ${REFDOCS_PKG})"
    mv "${refdocs_bin_expected}" "${refdocs_bin}"
    [[ ! -f "${refdocs_bin}" ]] && fail "refdocs failed to install"

    local clone_root out_dir
    clone_root="$(mktemp -d)"
    cleanup_repo_clone_root="${clone_root}"
    out_dir="$(mktemp -d)"

    local triggermesh_root
    triggermesh_root="${clone_root}/src/${TRIGGERMESH_REPO}"
    clone_at_commit "https://${TRIGGERMESH_REPO}.git" "${TRIGGERMESH_COMMIT}" \
        "${triggermesh_root}"

    mkdir -p "$OUTPUT_DIR/"

    array=(${triggermesh_root}/pkg/apis/!(common)/)
    for dir in "${array[@]}"
    do
        local group output
        group="$(basename $dir)"
        out_file="${TRIGGERMESH_OUTPUT_FILE_PREFIX}${group}.md"

        gen_refdocs "${refdocs_bin}" "${clone_root}" "${template_dir}" \
            "${out_dir}/${out_file}" "${triggermesh_root}" "./pkg/apis/${group}"

        cp "${out_dir}/${out_file}" "$OUTPUT_DIR/${out_file}"
    done

    log "SUCCESS: Generated docs written to $OUTPUT_DIR/."
}

main "$@"
