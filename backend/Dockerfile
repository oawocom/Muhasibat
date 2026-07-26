# Build context = repository root (see docker-compose.yml).

# --- frontend build (React + Vite + Tailwind) ---
FROM node:22-alpine AS frontend
WORKDIR /fe
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install
COPY frontend/ ./
# Emit the built SPA to /web (overrides the dev outDir in vite.config.js).
RUN npm run build -- --outDir /web --emptyOutDir

# --- backend build (Go, embeds the built SPA) ---
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY backend/go.mod backend/go.sum* ./
RUN go mod download
COPY backend/ .
COPY --from=frontend /web ./web
RUN CGO_ENABLED=0 go build -o /out/oawo-muhasibat .

# --- runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/oawo-muhasibat .
EXPOSE 8080
ENV GIN_MODE=release
CMD ["./oawo-muhasibat"]
