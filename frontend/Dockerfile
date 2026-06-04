# ------------------------------------------------------------
# Stage 1 — Build the Next.js application
# ------------------------------------------------------------
FROM node:20-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache libc6-compat

# Copy package files first (better caching)
COPY frontend/package.json frontend/package-lock.json ./

# Install dependencies
RUN npm ci

# Copy the rest of the frontend source
COPY frontend/ .

# Build the Next.js app in standalone mode
RUN npm run build

# ------------------------------------------------------------
# Stage 2 — Production runtime
# ------------------------------------------------------------
FROM node:20-alpine AS runner

WORKDIR /app

ENV NODE_ENV=production

# Copy only the standalone build output
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./public/_next/static
COPY --from=builder /app/public ./public

# Expose Next.js port
EXPOSE 3000

# Start the server
CMD ["node", "server.js"]
