# Build stage
FROM node:20-alpine3.21 AS builder

WORKDIR /app

# Copy dependency files and install using cache mount
COPY package*.json ./
RUN --mount=type=cache,target=/root/.npm \
    npm ci

# Copy source and build
COPY . .
RUN npm run build

# Run stage
FROM nginx:1.27-alpine

# Copy custom nginx config if it exists
COPY nginx.conf /etc/nginx/conf.d/default.conf

# Copy build artifacts from builder stage
COPY --from=builder /app/dist /usr/share/nginx/html

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
