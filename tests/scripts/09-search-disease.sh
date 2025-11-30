#!/bin/bash
: <<'COMMENT'

COMMENT
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "text": "uncontrolled emotion reading",
    "max_count": 3,
    "distance_threshold": 1.5
  }'

