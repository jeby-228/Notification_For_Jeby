# Dockerfile for Member API
# Build with: docker build -t member-api .

FROM golang:1.24-alpine AS builder
RUN apk --no-cache add ca-certificates
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o member_api ./main.go

FROM alpine:3.21.6
RUN apk --no-cache add ca-certificates && \
    update-ca-certificates
WORKDIR /app

COPY --from=builder /src/member_api ./member_api

ARG PORT=8080
EXPOSE ${PORT}

RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

ENTRYPOINT ["./member_api"]
