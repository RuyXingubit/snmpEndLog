FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /nms-web .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata && \
    addgroup -S -g 10001 nms && \
    adduser -S -D -u 10001 -G nms nms
WORKDIR /app
COPY --from=builder /nms-web .
COPY templates/ ./templates/
COPY static/ ./static/
RUN chown -R nms:nms /app

USER nms
EXPOSE 8080
CMD ["./nms-web"]
