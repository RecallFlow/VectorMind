#!/bin/bash
: <<'COMMENT'
# Relase script for mcp-rag-server
Usage:
1. Update the version in release.env
2. Run this script: ./release.sh
3. Create a GitHub release with the new tag and description
COMMENT

set -o allexport; source release.env; set +o allexport

echo "Generating release: ${TAG} ${ABOUT}"

find . -name '.DS_Store' -type f -delete

echo "📝 Replacing ${PREVIOUS_DOCKER_TAG} by ${DOCKER_TAG} in files..."

go run ./release/release.go -old="k33g/vectormind:${PREVIOUS_DOCKER_TAG}" -new="k33g/vectormind:${DOCKER_TAG}" -file="tests/with-embeddinggemma/compose.yml"
go run ./release/release.go -old="k33g/vectormind:${PREVIOUS_DOCKER_TAG}" -new="k33g/vectormind:${DOCKER_TAG}" -file="tests/with-mxbai-embed-large/compose.yml"
go run ./release/release.go -old="k33g/vectormind:${PREVIOUS_DOCKER_TAG}" -new="k33g/vectormind:${DOCKER_TAG}" -file="tests/with-nomic/compose.yml"

go run ./release/release.go -old="k33g/vectormind:${PREVIOUS_DOCKER_TAG}" -new="k33g/vectormind:${DOCKER_TAG}" -file="samples/openai-js-sdk/compose.yml"


go run ./release/release.go -old="${PREVIOUS_DOCKER_TAG}" -new="${DOCKER_TAG}" -file="README.md"

git add .
git commit -m "📦 ${ABOUT}"
git push origin main

git tag -a ${TAG} -m "${ABOUT}"
git push origin ${TAG}

