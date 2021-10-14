# build stage
FROM golang:1.17 as builder
WORKDIR /app
COPY . .
RUN go mod download
RUN go mod vendor
RUN go mod tidy
RUN CGO_ENABLED=0 go build -o binance-proxy ./cmd/binance-proxy/main.go

# target stage
FROM alpine
COPY --from=builder /app/binance-proxy /go/bin/binance-proxy
EXPOSE 8090
EXPOSE 8091
ENTRYPOINT ["/go/bin/binance-proxy"]