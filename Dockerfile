# @Ref: docs/sps/plans/20260730_infra_plan.md | @Date: 2026-07-30
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o pdeploy ./cmd/pdeploy

FROM alpine:latest
RUN apk add --no-cache git rsync openssh-client ca-certificates tzdata

WORKDIR /app
COPY --from=builder /app/pdeploy .

RUN mkdir -p /app/data /app/workspace && \
    chmod 755 /app/data /app/workspace

ENV PORT=8080
ENV DB_PATH=/app/data/pdeploy.db
ENV WORKSPACE_DIR=/app/workspace

EXPOSE 8080

ENTRYPOINT ["/app/pdeploy"]
