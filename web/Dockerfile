FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /snmpendlog-web .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /snmpendlog-web .
COPY templates/ ./templates/
COPY static/ ./static/

EXPOSE 8080
CMD ["./snmpendlog-web"]
