#!/bin/bash
: <<'COMMENT'
Script to test similarity search with label filtering
COMMENT

echo "=== Search in 'animals' label ==="
echo "Query: What lives in the forest?"
curl -X POST http://localhost:8080/search_with_label \
  -H "Content-Type: application/json" \
  -d '{
    "text": "What lives in the forest?",
    "label": "animals",
    "max_count": 5
  }'

echo -e "\n\n"

echo "=== Search in 'plants' label ==="
echo "Query: What grows in the forest?"
curl -X POST http://localhost:8080/search_with_label \
  -H "Content-Type: application/json" \
  -d '{
    "text": "What grows in the forest?",
    "label": "plants",
    "max_count": 5
  }'

echo -e "\n\n"

echo "=== Search in 'technology' label ==="
echo "Query: What helps with communication?"
curl -X POST http://localhost:8080/search_with_label \
  -H "Content-Type: application/json" \
  -d '{
    "text": "What helps with communication?",
    "label": "technology",
    "max_count": 5
  }'

echo -e "\n\n"

echo "=== Search in 'animals' label with distance threshold ==="
echo "Query: Swimming creatures"
curl -X POST http://localhost:8080/search_with_label \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Swimming creatures",
    "label": "animals",
    "max_count": 5,
    "distance_threshold": 0.8
  }'

echo -e "\n\n"

echo "=== Search in 'plants' label ==="
echo "Query: Beautiful flowers"
curl -X POST http://localhost:8080/search_with_label \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Beautiful flowers",
    "label": "plants",
    "max_count": 3
  }'

echo -e "\n"
