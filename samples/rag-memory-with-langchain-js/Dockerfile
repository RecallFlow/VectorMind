# Stage 1: Build stage
FROM node:22.19.0-trixie

WORKDIR /app
COPY package*.json ./
RUN npm install
COPY *.js .

# Create non-root user
RUN groupadd --gid 1001 nodejs && \
    useradd --uid 1001 --gid nodejs --shell /bin/bash --create-home bob-loves-js

# Change ownership of the app directory
RUN chown -R bob-loves-js:nodejs /app

# Switch to non-root user
USER bob-loves-js

