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

if (( $# > 1 )) || (( $# == 1 )) && [ "$1" == '-h' ]; then
	echo "Verifies that all TriggerMesh components have a properly set up kodata directory."
	echo
	echo "USAGE:"
	echo "    ${0##*/} [CMD_DIR]"

	exit 1
fi

declare cmd_dir
if (( $# == 0 )); then
	cmd_dir=cmd
else
	cmd_dir="${1%/}"
fi

if [ ! -d "$cmd_dir" ]; then
	echo "Directory ${cmd_dir} does not exist."
	exit 1
fi

declare -a expect_symlinks
expect_symlinks=(
	LICENSES
	HEAD
	refs
)

declare -A kodata_errors
kodata_errors=()

for cmd in "$cmd_dir"/*/; do
	for filename in "${expect_symlinks[@]}"; do
		filepath="${cmd}kodata/${filename}"

		if [ ! -L "$filepath" ]; then
			kodata_errors["$filepath"]="does not exist or is not a symlink"
			continue
		elif [ ! -e "$filepath" ]; then
			kodata_errors["$filepath"]="broken symlink"
			continue
		fi
	done
done

num_errors="${#kodata_errors[@]}"
if (( num_errors > 0 )); then
	echo "Found ${num_errors} errors in kodata directories."
	echo

	for filepath in "${!kodata_errors[@]}"; do
		echo "${filepath}:"
		echo "    ${kodata_errors[$filepath]}"
	done

	exit 1
fi
