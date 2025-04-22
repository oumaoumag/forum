# Build stage
FROM golang:1.20-alpine AS builder
RUN apk add --no-cache build-base sqlite-dev

WORKDIR /app
COPY . .
RUN go mod download
WORKDIR /app/cmd
RUN CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o main .

# Final stage
FROM alpine:latest
WORKDIR /app
RUN apk --no-cache add sqlite-libs
COPY --from=builder /app/cmd/main .
COPY --from=builder /app .
RUN chmod +x main
EXPOSE 8080
CMD ["./main"]