#!/usr/bin/env bash

# Copyright 2022 TriggerMesh Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -eu
set -o pipefail

if (( $# > 1 )) || (( $# == 1 )) && [ "$1" == '-h' ]; then
	echo "Updates all TriggerMesh CRDs with common attributes."
	echo
	echo "USAGE:"
	echo "    ${0##*/} [CONFIG_DIR]"

	exit 1
fi

declare config_dir
if (( $# == 0 )); then
	config_dir=config
else
	config_dir="${1%/}"
fi

if [ ! -d "$config_dir" ]; then
	echo "Directory ${config_dir} does not exist."
	exit 1
fi

tool_dir="${0%/*}"

# Creates and populates a Python venv inside the script's directory.
# No-op if it already exists.
function create_venv {
	if [ -d "${tool_dir}"/venv ]; then
		echo 0
		return 0
	elif [ -e "${tool_dir}"/venv ]; then
		echo "venv exists but is not a directory" >&2
		echo 0
		return 1
	fi

	python3 -m venv "${tool_dir}"/venv >/dev/null
	source "${tool_dir}"/venv/bin/activate
	pip3 install -r "${tool_dir}"/requirements.txt >/dev/null
	deactivate

	echo 1
}

# Clears the Python venv inside the script's directory.
function remove_venv {
	rm -rf "${tool_dir}"/venv
}

# Runs the crd-update.py script located inside the script's directory.
function crd_update {
	"${tool_dir}"/crd-update.py
}

declare -i venv_created
venv_created="$(create_venv)"
if ((venv_created)); then
	trap remove_venv EXIT HUP INT QUIT PIPE TERM
fi
source "${tool_dir}"/venv/bin/activate

for crd_file in "$config_dir"/30[0-4]-*.yaml; do
	crd_update <"$crd_file" >"$crd_file".new
	mv "$crd_file".new "$crd_file"
done
