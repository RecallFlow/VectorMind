#!/bin/bash
: <<'COMMENT'
Script to create embeddings with different labels
COMMENT

echo "Creating animals embeddings..."

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Squirrels run in the forest",
        "label": "animals",
        "metadata": "type=mammal"
    }'

echo -e "\n"

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Birds fly in the sky",
        "label": "animals",
        "metadata": "type=bird"
    }'

echo -e "\n"

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Frogs swim in the pond",
        "label": "animals",
        "metadata": "type=amphibian"
    }'

echo -e "\n"

echo "Creating plants embeddings..."

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Oak trees grow in the forest",
        "label": "plants",
        "metadata": "type=tree"
    }'

echo -e "\n"

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Roses bloom in the garden",
        "label": "plants",
        "metadata": "type=flower"
    }'

echo -e "\n"

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Ferns thrive in the shade",
        "label": "plants",
        "metadata": "type=fern"
    }'

echo -e "\n"

echo "Creating technology embeddings..."

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Smartphones connect people worldwide",
        "label": "technology",
        "metadata": "category=mobile"
    }'

echo -e "\n"

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Computers process large amounts of data",
        "label": "technology",
        "metadata": "category=computing"
    }'

echo -e "\n"

curl -X POST http://localhost:8080/embeddings \
    -H "Content-Type: application/json" \
    -d '{
        "content": "Robots automate manufacturing processes",
        "label": "technology",
        "metadata": "category=robotics"
    }'

echo -e "\n"
echo "All embeddings created!"
