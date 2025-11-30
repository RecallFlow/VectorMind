#!/bin/bash
: <<'COMMENT'

COMMENT
curl -X POST http://localhost:8080/search \
  -H "Content-Type: application/json" \
  -d '{
    "text": "Give me the list of the monsters of the game",
    "max_count": 3,
    "distance_threshold": 0.7
  }'

