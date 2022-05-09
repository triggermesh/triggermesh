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

pushd "$tool_dir"
python3 -m venv venv
source venv/bin/activate
pip3 install -r requirements.txt
popd

function crd_update {
	"${tool_dir}"/crd-update.py
}

for crd_file in "$config_dir"/30[0-4]-*.yaml; do
	crd_update <"$crd_file" >"$crd_file".new
	mv "$crd_file".new "$crd_file"
done

deactivate
