# ------------------------------------------------------------
# Stage 1 — Build the Next.js application
# ------------------------------------------------------------
FROM node:20-alpine AS builder

WORKDIR /app

RUN apk add --no-cache libc6-compat

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci

COPY frontend/ .
RUN npm run build

# ------------------------------------------------------------
# Stage 2 — Production runtime
# ------------------------------------------------------------
FROM node:20-alpine AS runner

WORKDIR /app
ENV NODE_ENV=production

# Copy standalone server
COPY --from=builder /app/.next/standalone ./

# Copy static assets
COPY --from=builder /app/.next/static ./public/_next/static

# No public/ copy — Next.js standalone does not need it

EXPOSE 3000
CMD ["node", "server.js"]
