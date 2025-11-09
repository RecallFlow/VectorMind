#!/bin/bash
: <<'COMMENT'

COMMENT
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Which animals swim?",
    "max_count": 3,
    "distance_threshold": 0.7
  }'

curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Where are the squirrels?",
    "max_count": 3,
    "distance_threshold": 0.7
  }'

curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "text": "What can be found in the pond?",
    "max_count": 3,
    "distance_threshold": 0.7
  }'

