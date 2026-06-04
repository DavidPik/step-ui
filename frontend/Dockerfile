# ============================
# 1) Build stage
# ============================
FROM node:20-alpine AS builder

WORKDIR /app

# Copy only frontend package files
COPY frontend/package*.json ./
RUN npm install

# Copy frontend source
COPY frontend/ .

# Build Next.js app
RUN npm run build

# ============================
# 2) Runtime stage
# ============================
FROM node:20-alpine AS runner

WORKDIR /app

# Copy standalone build
COPY --from=builder /app/.next/standalone ./

# Copy static assets
COPY --from=builder /app/.next/static ./.next/static

# Copy public folder only if it exists
RUN mkdir -p public
COPY --from=builder /app/public ./public 2>/dev/null || true

EXPOSE 3000
ENV PORT=3000
ENV NODE_ENV=production

CMD ["node", "server.js"]
