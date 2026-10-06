# syntax=docker/dockerfile:1

# ---- frontend build ----
FROM node:22-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm ci --ignore-scripts
COPY web/ .
RUN npm run build

# ---- backend build ----
FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download || true
COPY . .
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /out/feedback-api ./cmd/api

# ---- runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /out/feedback-api /app/feedback-api
COPY --from=web-builder /web/dist /app/web/dist
COPY configs/config.example.yaml /app/configs/config.example.yaml
EXPOSE 8080
ENV TZ=Asia/Shanghai
ENV FEEDBACK_ENV=local
ENTRYPOINT ["/app/feedback-api"]