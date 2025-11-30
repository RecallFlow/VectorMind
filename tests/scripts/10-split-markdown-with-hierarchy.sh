#!/bin/bash
: <<'COMMENT'
Script to test split and store markdown with hierarchy endpoint
This script loads the chronicles.md file, splits it by markdown hierarchy, and stores all chunks
COMMENT

echo "=== Split markdown with hierarchy and store the Chronicles of Aethelgard document ==="
echo "Loading document from chronicles.md and splitting with hierarchy preservation"

# Read the markdown file content and escape it properly for JSON
DOCUMENT_CONTENT=$(cat ./chronicles.advanced.md | jq -Rs .)

# Prepare the JSON payload
JSON_PAYLOAD=$(cat <<EOF
{
  "document": ${DOCUMENT_CONTENT},
  "label": "rpg-rules-hierarchy",
  "metadata": "game=chronicles-of-aethelgard,type=rules,split_method=markdown_hierarchy"
}
EOF
)

curl -X POST http://localhost:8080/split-and-store-markdown-with-hierarchy \
  -H "Content-Type: application/json" \
  -d "${JSON_PAYLOAD}"

echo -e "\n"
