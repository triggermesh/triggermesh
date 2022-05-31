#!/usr/bin/env bash

RELEASE=${1:-${GIT_TAG}}
RELEASE=${RELEASE:-${CIRCLE_TAG}}

if [ -z "${RELEASE}" ]; then
  echo "Usage:"
  echo "release-notes.sh VERSION"
  exit 1
fi

if ! git rev-list ${RELEASE} >/dev/null 2>&1; then
	echo "${RELEASE} does not exist"
	exit
fi

KREPO="triggermesh"
BASE_URL="https://github.com/triggermesh/${KREPO}/releases/download/${RELEASE}"
PREV_RELEASE=${PREV_RELEASE:-$(git describe --tags --abbrev=0 ${RELEASE}^ 2>/dev/null)}
PREV_RELEASE=${PREV_RELEASE:-$(git rev-list --max-parents=0 ${RELEASE}^ 2>/dev/null)}
NOTABLE_CHANGES=$(git cat-file -p ${RELEASE} | sed '/-----BEGIN PGP SIGNATURE-----/,//d' | tail -n +6)
CHANGELOG=$(git log --no-merges --pretty=format:'- [%h] %s (%aN)' ${PREV_RELEASE}..${RELEASE})
if [ $? -ne 0 ]; then
  echo "Error creating changelog"
  exit 1
fi

IMAGE_REPO=${IMAGE_REPO:-"gcr.io/triggermesh"}
PLATFORMS=$(sed -n -e "s/^\(TARGETS[[:space:]]*?=[[:space:]]*\)\(.*\)$/\2/p" Makefile)
RELEASE_ASSETS_TABLE=$(
  echo -n "| component | container |"; echo
  echo -n "| -- | -- |"; echo
  for cmd in cmd/*; do
    echo -n "| ${cmd##*/} | [${IMAGE_REPO}/${cmd##*/}:${RELEASE}](https://${IMAGE_REPO}/${cmd##*/}:${RELEASE})"
    echo -n " |"; echo
  done
  echo
)

cat <<EOF
${NOTABLE_CHANGES}

## Installation

Install all TriggerMesh custom resources:


\`\`\`console
kubectl apply -f https://github.com/triggermesh/triggermesh/releases/download/${RELEASE}/triggermesh-crds.yaml
\`\`\`

Install the TriggerMesh open source platform components:

\`\`\`console
kubectl apply -f https://github.com/triggermesh/triggermesh/releases/download/${RELEASE}/triggermesh.yaml
\`\`\`

${RELEASE_ASSETS_TABLE}

## Changelog

${CHANGELOG}
EOF
