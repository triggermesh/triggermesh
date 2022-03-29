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
	echo "Verifies that all TriggerMesh CRDs have valid annotations."
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

declare -a required_commands
required_commands=(
	yq
	jq
)

declare -a missing_commands
missing_commands=()

for cmd in "${required_commands[@]}"; do
	if ! command -v "$cmd" >/dev/null; then
		missing_commands+=("$cmd")
	fi
done

if (( ${#missing_commands[@]} > 0 )); then
	echo "Missing required commands: ${missing_commands[*]}"
	exit 1
fi

declare -a json_annotations
json_annotations=(
	registry.knative.dev/eventTypes
	registry.triggermesh.io/acceptedEventTypes
)

declare -A annotation_errors
annotation_errors=()

for crd_file in "$config_dir"/30[0-4]-*.yaml; do
	for annotation in "${json_annotations[@]}"; do
		if ! yq eval ".metadata.annotations[\"${annotation}\"]" "$crd_file" | jq >/dev/null; then
			annotation_errors["$crd_file"]="Value of \"${annotation}\" isn't valid JSON"
		fi
	done
done

num_errors="${#annotation_errors[@]}"
if (( num_errors > 0 )); then
	echo "Found ${num_errors} errors in CRD annotations."
	echo

	for crd_file in "${!annotation_errors[@]}"; do
		echo "$crd_file:"
		echo "    ${annotation_errors[${crd_file}]}"
	done

	exit 1
fi
