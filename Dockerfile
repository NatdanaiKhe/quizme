# Stage 1: Build Frontend SPA
FROM node:22-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Backend Monolith (embeds frontend/dist)
FROM golang:alpine AS backend-builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/quizme ./cmd/server/main.go

# Stage 3: Minimal Runtime Container
FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend-builder /app/bin/quizme /app/quizme

EXPOSE 8080
ENTRYPOINT ["/app/quizme"]
