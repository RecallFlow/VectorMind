# VectorMind Tests

This document describes the test suite for VectorMind and how to run the tests.

## Test Structure

The test suite is divided into two categories:

### Unit Tests

Unit tests do not require any external dependencies and can run independently. They test:

- `TestCreateRedisClient` - Verifies Redis client creation with various configurations
- `TestCloseRedisClient` - Verifies Redis client closure
- `TestConvertEmbeddingToFloat32` - Tests conversion from float64 to float32 arrays
- `TestConvertOpenAIEmbeddingResponseToFloat32` - Tests conversion from OpenAI embedding response
- `TestFloatsToBytes` - Verifies float32 to byte array conversion
- `TestFloatsToBytesRoundTrip` - Verifies conversion consistency and correctness
- `TestHealthCheckHandler` - Tests the HTTP health check endpoint
- `TestSimilaritySearchHandler_RequestValidation` - Tests request validation for similarity search (HTTP method, JSON parsing, required fields)
- `TestSimilaritySearchRequest_DistanceThresholdField` - Tests JSON serialization/deserialization of the optional `distance_threshold` parameter
- `TestSplitAndStoreMarkdownWithHierarchyHandler_RequestValidation` - Tests request validation for split markdown with hierarchy endpoint
- `TestSplitAndStoreMarkdownWithHierarchyRequest_JSONMarshaling` - Tests JSON marshaling of split markdown with hierarchy requests
- `TestSplitAndStoreMarkdownWithHierarchyResponse_JSONMarshaling` - Tests JSON marshaling of split markdown with hierarchy responses

#### Splitter Package Tests

The `splitter` package includes comprehensive unit tests for markdown processing:

- `TestParseMarkdownHierarchy` - Tests markdown parsing with hierarchical structure (6 test cases covering different hierarchy levels and edge cases)
- `TestBuildHierarchy` - Tests hierarchy string generation from markdown structure
- `TestChunkWithMarkdownHierarchy` - Tests chunk generation with TITLE, HIERARCHY, and CONTENT metadata (4 test cases)
- `TestChunkWithMarkdownHierarchy_Format` - Verifies the exact format of generated chunks
- `TestMarkdownChunkStruct` - Tests the MarkdownChunk data structure

### Integration Tests

Integration tests require a running Redis instance and test:

- `TestIndexExists_Integration` - Checks if a Redis vector index exists
- `TestCreateEmbeddingIndex_Integration` - Creates a vector index in Redis
- `TestDropIndex_Integration` - Drops a vector index from Redis
- `TestStoreEmbedding_Integration` - Stores embeddings in Redis
- `TestSimilaritySearch_Integration` - Performs similarity search on stored embeddings

## Running Tests

### Run All Tests

```bash
go test
```

### Run Unit Tests Only

```bash
go test -short
```

This skips all integration tests that require external dependencies.

### Run Integration Tests Only

```bash
go test -run Integration
```

### Run with Verbose Output

```bash
go test -v
```

### Run Specific Test

```bash
go test -run TestHealthCheckHandler
```

### Run Tests for Specific Package

```bash
# Run all tests in the splitter package
go test -v ./splitter/

# Run specific test in splitter package
go test -v -run TestChunkWithMarkdownHierarchy ./splitter/
```

### Run All Tests in All Packages

```bash
# Short mode (unit tests only)
go test -short ./...

# All tests including integration tests
go test ./...
```

## Prerequisites for Integration Tests

Integration tests require:

1. **Redis Stack** running locally on `localhost:6379`
   - Redis Stack includes RediSearch module for vector operations

2. Start Redis Stack with Docker:
   ```bash
   docker run -d --name redis-stack -p 6379:6379 redis/redis-stack:latest
   ```

## Test Coverage

To generate a test coverage report:

```bash
go test -cover
```

For detailed coverage report:

```bash
go test -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Continuous Integration

When running in CI/CD environments:

- Use `-short` flag to run only unit tests if Redis is not available
- Ensure Redis Stack is running before executing integration tests
- Set appropriate timeouts for long-running tests

Example CI command:
```bash
# Unit tests only
go test -short -v

# Full test suite (requires Redis)
go test -v
```
