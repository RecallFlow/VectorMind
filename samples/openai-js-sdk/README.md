# Use VectorMind with OpenAI JS SDK sample

**Start the sample**:
```bash
docker model pull hf.co/menlo/jan-nano-gguf:q4_k_m
npm install
# Start VectorMind and Redis
docker compose up -d
node index.js
```

**To restart the sample**:
```bash
docker compose down && docker compose up -d
```

**Stop the sample**:
```bash
docker compose down
```