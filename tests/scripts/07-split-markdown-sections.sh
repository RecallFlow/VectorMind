#!/bin/bash
: <<'COMMENT'
Script to test split and store markdown sections endpoint
This script loads the chronicles.md file, splits it by markdown sections, and stores all sections
COMMENT

echo "=== Split markdown by sections and store the Chronicles of Aethelgard document ==="
echo "Loading document from chronicles.md and splitting by headers"

# Read the markdown file content and escape it properly for JSON
DOCUMENT_CONTENT=$(cat ./chronicles.md | jq -Rs .)

# Prepare the JSON payload
JSON_PAYLOAD=$(cat <<EOF
{
  "document": ${DOCUMENT_CONTENT},
  "label": "rpg-rules-sections",
  "metadata": "game=chronicles-of-aethelgard,type=rules,split_method=markdown_sections"
}
EOF
)

curl -X POST http://localhost:8080/split-and-store-markdown-sections \
  -H "Content-Type: application/json" \
  -d "${JSON_PAYLOAD}"

echo -e "\n"
