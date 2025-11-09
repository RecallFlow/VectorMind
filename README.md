# VectorMind
> A text RAG (Retrieval Augmented Generation) System based on Redis with REST API endpoints and MCP (Model Context Protocol) server support.

## What is VectorMind?

VectorMind is a lightweight vector database service that provides semantic search capabilities using Redis as the backend storage. It creates embeddings from text content and enables similarity-based search operations.

### Key Features

- **Dual Interface**: Exposes both REST API (port 8080) and MCP server (port 9090) for flexibility
- **Vector Storage**: Uses Redis with HNSW (Hierarchical Navigable Small World) indexing for efficient similarity search
- **Embedding Support**: For example: creates embeddings using the `ai/mxbai-embed-large` model
- **Document Management**: Store documents with optional labels and metadata
- **Similarity Search**: Find similar documents based on text queries with configurable distance thresholds

### Architecture

VectorMind consists of:
- **Redis Server**: Stores embeddings and provides vector search capabilities via RediSearch
- **VectorMind Service**: Go application that handles embedding generation and exposes APIs
- **Embedding Model**: Uses `ai/mxbai-embed-large` model for text embeddings (configurable)

## Getting Started

### Prerequisites

- Docker, Docker Model Runner and Docker Agentic Compose

### Starting VectorMind

1. **Using Docker Compose** (recommended):

**Create** a `compose.yml` file with the following content:
```yaml
services:

  redis-server:
    image: redis:8.2.3-alpine3.22
    environment: 
      - REDIS_ARGS=--save 30 1
    ports:
      - 6379:6379
    volumes:
      - ./data:/data

  vectormind-tests:
    image: k33g/vectormind:0.0.1
    ports:
      - 9090:9090
      - 8080:8080
    environment:
      REDIS_INDEX_NAME: vectormind_index
      REDIS_ADDRESS: redis-server:6379
      REDIS_PASSWORD: ""

      MCP_HTTP_PORT: 9090
      API_REST_PORT: 8080

    models:
      embedding-model:
        endpoint_var: MODEL_RUNNER_BASE_URL
        model_var: EMBEDDING_MODEL

    depends_on:
      redis-server:
        condition: service_started

models:

  embedding-model:
    model: ai/mxbai-embed-large
```

**Run** the following command to start **VectorMind**:
```bash
docker compose up -d
```

This will start:
- Redis server on port `6379`
- VectorMind MCP server on port `9090`
- VectorMind REST API on port `8080`

2. **Environment Variables**:

The compose file automatically configures:
- `REDIS_INDEX_NAME`: vectormind_index
- `REDIS_ADDRESS`: redis-server:6379
- `MCP_HTTP_PORT`: 9090
- `API_REST_PORT`: 8080
- `MODEL_RUNNER_BASE_URL`: Set via models configuration
- `EMBEDDING_MODEL`: ai/mxbai-embed-large

### Verifying the Installation

Check if VectorMind is running:

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "server": "mcp-vectormind-server"
}
```

## How to Use VectorMind

### REST API Usage

#### 1. Create Embeddings

Store text content with optional labels and metadata:

```bash
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
```

Response:
```json
{"id":"doc:b1c36710-9d94-41cb-abfc-aa404b896d1f","content":"Squirrels run in the forest","label":"animals","metadata":"id=animals_1","created_at":"2025-11-09T08:36:01.962629337Z","success":true}
{"id":"doc:fbc259cc-eb8d-425e-a444-d4f5b26400cb","content":"Birds fly in the sky","label":"animals","metadata":"id=animals_2","created_at":"2025-11-09T08:36:02.093359462Z","success":true}
{"id":"doc:0154fc6d-887b-4af2-a5a7-c8b37183554f","content":"Frogs swim in the pond","label":"animals","metadata":"id=animals_3","created_at":"2025-11-09T08:36:02.247079753Z","success":true}
{"id":"doc:3953dfdd-2a92-48de-b61b-0119c9d106fc","content":"Fishes swim in the sea","label":"animals","metadata":"id=animals_4","created_at":"2025-11-09T08:36:02.367855295Z","success":true}
```

#### 2. Search for Similar Documents

Find documents similar to a query text:

```bash
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
```

Response:
```json
{"results":[{"id":"doc:050c7cee-5891-4052-a3c9-40f2bd3abff7","content":"Fishes swim in the sea","distance":0.5175167322158813},{"id":"doc:efe2868d-3330-452c-ac2a-0e835caecdc9","content":"Frogs swim in the pond","distance":0.6700224280357361}],"success":true}
{"results":[{"id":"doc:14e7a8fb-78e5-4fe7-8969-7559b7cd9752","content":"Squirrels run in the forest","distance":0.48874980211257935}],"success":true}
{"results":[{"id":"doc:efe2868d-3330-452c-ac2a-0e835caecdc9","content":"Frogs swim in the pond","distance":0.6417693495750427}],"success":true}
```

**Parameters**:
- `text` (required): The search query
- `max_count` (optional): Maximum number of results (default: 5)
- `distance_threshold` (optional): Maximum distance to filter results (lower = more similar)

