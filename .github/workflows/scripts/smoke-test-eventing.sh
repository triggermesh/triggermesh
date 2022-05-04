#!/usr/bin/env bash

# The Knative Eventing webhook occasionally responds with "connection refused"
# although its Pod(s) reports "Ready", making Knative Eventing unusable.
# This script performs a smoke test of the Eventing installation by attempting
# a create/delete cycle of a basic Channel object. In case of failure, it
# deletes the Eventing webhook Pod(s) and retries.

set -eu
set -o pipefail

function create_channel {
	local err

	err="$(kubectl create -f - 2>&1 >/dev/null <<-EOM
	apiVersion: messaging.knative.dev/v1
	kind: Channel
	metadata:
	  name: smoke-test
	EOM
	)"

	echo "$err"
}

function delete_channel {
	kubectl delete channels.messaging.knative.dev/smoke-test >/dev/null
}

declare -i succeeded=0
declare last_err

declare -i max_attempts=3
declare -i was_retried

# outer loop:
#  - try to create Channel multiple times
#  - delete webhook and retry on failure (3 attempts)
for attempt in $(seq 1 "$max_attempts"); do
	was_retried=0

	# inner loop: create Channel with retries (6 attempts)
	for _ in $(seq 1 6); do
		last_err="$(create_channel)"
		if [ -z "${last_err:-}" ]; then
			succeeded=1
			break
		fi

		was_retried=1
		echo -n '.' >&2
		sleep 5
	done

	if ((was_retried)); then
		# flush stderr, important in non-interactive environments (CI)
		echo >&2
	fi

	if ((succeeded)); then
		delete_channel
		break
	fi

	if ((attempt < max_attempts)); then
		kubectl -n knative-eventing delete pods -l 'app.kubernetes.io/component=eventing-webhook'
	fi
done

if [ -n "$last_err" ]; then
	echo "Smoke test Channel creation failed: ${last_err}" >&2
fi
