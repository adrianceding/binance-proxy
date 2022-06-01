
# binance-proxy

Developed based on [https://github.com/adshao/go-binance](https://github.com/adshao/go-binance). He is `the shoulders of giants`.Thanks

local `klines` ,`depth` ,`exchangeInfo`,`24hour`

| Data | Spot Interval | Future Interval |
|--|--|--|
|klines|2000ms(or 1000ms?) depend on binance limit| 250ms |
|depth|100ms| 100ms |
|exchangeInfo|60s| 60s |
|24hr.BidPrice/AskPrice| realtime | realtime |
|24hr.AnotherField|1000ms | 500ms |

**Caution**
1. Because the structure of the `24hr` data of future is different from that of spot, and the delay of websocket is very large (spot: 1000ms, future: 500ms). Therefore, the `BidPrice/AskPrice` field of `24hr` actually uses the data of `bookTicker` , it is real-time
2. Due to the depth of binance weboskcet only provides 20-depth push.So proxy maintains a 1000 depth orderbook locally
3. The case where kline automatically converts to a reverse proxy,See the table below for details. They are **OR** relationship
   1. The `startTime` parameter is set and is earlier than the first date cached locally
   2. The `endTime` parameter is set
   3. The `limit` parameter is less than or equal to 0 or greater than 1000
   4. The `symbol` or `interval` parameter is not set
4. `kline` needs to rely on api data for initialization. And due to the limit and wait the first websocket push data, there may be a certain delay. About a few seconds.

**Command**
```
      -f string
            futures bind address. (default ":8091")
      -s string
            spot bind address. (default ":8090")
      -v    print debug log.
```

**Freqtrade config**
```
"exchange": {
        "name": "binance",
        "key": "",
        "secret": "",
        "ccxt_config": {
            "enableRateLimit": false,
            "urls": {
                "api": {
                    "public": "http://127.0.01:8090/api/v3", # spot add this 
                    "fapiPublic": "http://127.0.01:8091/fapi/v1" # futures add this
                }
            }
        },
        "ccxt_async_config": {
            "enableRateLimit": false,
        },
}
````