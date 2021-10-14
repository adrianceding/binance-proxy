# binance-proxy
binance proxy
local `candle` ,`orderbook` ,`exchangeinfo`,`24hour`

Only `spot` support `24hour`

`futures`Please use orderbook instead

command
```
      -f string
            futures bind address. (default ":8091")
      -fakekline
            enable fake kline.
      -s string
            spot bind address. (default ":8090")
      -v    print debug log.
```

change freqtrade config
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
```