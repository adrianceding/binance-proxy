FROM alpine
COPY binance-proxy /usr/local/bin/binance-proxy
ENTRYPOINT ["/usr/local/bin/binance-proxy"]
