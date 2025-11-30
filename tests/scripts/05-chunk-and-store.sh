#!/bin/bash
: <<'COMMENT'
Script to test chunk and store endpoint
This script loads the chronicles.md file, chunks it, and stores all chunks
COMMENT

echo "=== Chunk and store the Chronicles of Aethelgard document ==="
echo "Loading document from chronicles.md with chunk_size=1024, overlap=256"

# Read the markdown file content and escape it properly for JSON
DOCUMENT_CONTENT=$(cat ./chronicles.md | jq -Rs .)

# Prepare the JSON payload
JSON_PAYLOAD=$(cat <<EOF
{
  "document": ${DOCUMENT_CONTENT},
  "label": "rpg-rules",
  "metadata": "game=chronicles-of-aethelgard,type=rules",
  "chunk_size": 1024,
  "overlap": 256
}
EOF
)

curl -X POST http://localhost:8080/chunk-and-store \
  -H "Content-Type: application/json" \
  -d "${JSON_PAYLOAD}"

echo -e "\n"
