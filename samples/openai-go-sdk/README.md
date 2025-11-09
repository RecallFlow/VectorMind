# Use VectorMind with OpenAI Golang SDK sample

**Start the sample**:
```bash
docker model pull hf.co/menlo/jan-nano-gguf:q4_k_m
# Start VectorMind and Redis
docker compose up -d
go run main.go
```

**To restart the sample**:
```bash
docker compose down && docker compose up -d
```

**Stop the sample**:
```bash
docker compose down
```