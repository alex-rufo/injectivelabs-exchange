# ---------- Stage 1: Build ----------
FROM golang:1.24 AS builder

WORKDIR /app

# Copy go.mod and go.sum to cache dependencies first
COPY go.mod go.sum ./
RUN go mod download
# Copy the rest of the source code
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o server

# ---------- Stage 2: Runtime ----------
FROM alpine:latest AS runtime

WORKDIR /app

# Copy the binary and set permissions in one layer
COPY --from=builder /app/server .
RUN ls -l /app/server && chmod +x /app/server && ls -l /app/server

EXPOSE ${HTTP_PORT}

ENTRYPOINT ./server server --port $HTTP_PORT --currencies $CURRENCIES --interval $INTERVAL --ttl $TTL --subscripition-buffer-size $SUBSCRIPTION_BUFFER_SIZE --coindesk-base-url $COINDESK_BASE_URL --coindesk-timeout $COINDESK_TIMEOUT
