# --- build ---
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /out/oawo-muhasibat .

# --- run ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/oawo-muhasibat .
EXPOSE 8080
ENV GIN_MODE=release
CMD ["./oawo-muhasibat"]
