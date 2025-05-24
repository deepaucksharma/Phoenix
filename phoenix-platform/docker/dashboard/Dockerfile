# Build stage
FROM node:18-alpine AS builder

WORKDIR /app

# Copy package files
COPY dashboard/package*.json ./
RUN npm ci --only=production

# Copy source code
COPY dashboard/ .

# Build the application
RUN npm run build

# Runtime stage
FROM nginx:alpine

# Copy custom nginx config
COPY docker/dashboard/nginx.conf /etc/nginx/nginx.conf

# Copy built app from builder
COPY --from=builder /app/dist /usr/share/nginx/html

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]