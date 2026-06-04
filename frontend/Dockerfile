# ============================
# 1) Build stage
# ============================
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies
COPY frontend/package*.json ./
RUN npm install

# Copy source
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

# Create public folder (always exists)
RUN mkdir -p public

# Copy public folder only if it exists in builder
RUN if [ -d /app/public ]; then cp -r /app/public/* ./public/; fi

EXPOSE 3000
ENV PORT=3000
ENV NODE_ENV=production

CMD ["node", "server.js"]
