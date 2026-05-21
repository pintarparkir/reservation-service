# reservation-service container.
# Multi-stage so the final image is just the binary + ca-certs.

FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/reservation ./cmd/reservation

FROM alpine:3.19
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /out/reservation /reservation
EXPOSE 8081 9090
ENTRYPOINT ["/reservation"]
