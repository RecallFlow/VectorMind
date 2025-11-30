#!/bin/bash
: <<'COMMENT'
Script to test split and store with delimiter endpoint
This script loads the startrek-diseases.txt file, splits it by the "-----" delimiter, and stores all chunks
COMMENT

echo "=== Split document by delimiter and store the Star Trek Medical Database ==="
echo "Loading document from startrek-diseases.txt and splitting by '-----' delimiter"

# Read the text file content and escape it properly for JSON
DOCUMENT_CONTENT=$(cat ./startrek-diseases.txt | jq -Rs .)

# Prepare the JSON payload
JSON_PAYLOAD=$(cat <<EOF
{
  "document": ${DOCUMENT_CONTENT},
  "delimiter": "-----",
  "label": "star-trek-diseases",
  "metadata": "source=federation-medical-database,universe=star-trek"
}
EOF
)

curl -X POST http://localhost:8080/split-and-store-with-delimiter \
  -H "Content-Type: application/json" \
  -d "${JSON_PAYLOAD}"

echo -e "\n"
