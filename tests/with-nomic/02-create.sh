#!/bin/bash
: <<'COMMENT'

COMMENT

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Squirrels run in the forest",
        "label": "animals",
        "metadata": "id=animals_1"
    }'

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Birds fly in the sky",
        "label": "animals",
        "metadata": "id=animals_2"
    }'

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Frogs swim in the pond",
        "label": "animals",
        "metadata": "id=animals_3"
    }'

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Fishes swim in the sea",
        "label": "animals",
        "metadata": "id=animals_4"
    }'
