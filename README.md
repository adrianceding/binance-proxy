<h1 align="center">  Binance Proxy</h1>
<p align="center">
A fast and simple <b>Websocket Proxy</b> for the <b>Binance API</b> written in <b>GoLang</b>. Mimics the behavior of API endpoints to avoid rate limiting imposed on IP's when using REST queries. Intended Usage for multiple instances of applications querying the Binance API at a rate that might lead to banning or blocking, like for example the <a href="https://github.com/freqtrade/freqtrade">Freqtrade Trading Bot</a>, or any other similar application. </p>

<p align="center"><a href="https://github.com/nightshift2k/binance-proxy/releases" target="_blank"><img src="https://img.shields.io/github/v/release/nightshift2k/binance-proxy?style=for-the-badge" alt="latest version" /></a>&nbsp;<img src="https://img.shields.io/github/go-mod/go-version/nightshift2k/binance-proxy?style=for-the-badge" alt="go version" />&nbsp;<img src="https://img.shields.io/tokei/lines/github/nightshift2k/binance-proxy?color=pink&style=for-the-badge" />&nbsp;<a href="https://github.com/nightshift2k/binance-proxy/issues" target="_blank"><img src="https://img.shields.io/github/issues/nightshift2k/binance-proxy?color=purple&style=for-the-badge" alt="github issues" /></a>&nbsp;<img src="https://img.shields.io/github/license/nightshift2k/binance-proxy?color=red&style=for-the-badge" alt="license" /></p>

## ⚡ Quick Start
You can download the pre-compiled binary for the architecture of your choice from the [relaseses page](https://github.com/nightshift2k/binance-proxy/releases) on GitHub.

Unzip the package to a folder of choice, preferably one that's in `$PATH`
```bash
tar -xf binance-proxy_1.2.4_Linux_x86_64.tar.gz -C /usr/local/bin 
```
Starting the proxy:
```bash
binance-proxy
```
That's all you need to know to start! 🎉

### 🐳 Docker-way to quick start

If you don't want to install or compile the binance-proxy to your system, feel free using the prebuild  [Docker images](https://hub.docker.com/r/nightshift2k/binance-proxy) and run it from an isolated container:

```bash
docker run --rm -d nightshift2k/binance-proxy:latest
```
ℹ️ Please pay attention to configuring network access, per default the ports `8090` and `8091` are exposed, if you specify different ports via parameters, you will need to re-configure your docker setup. Please refer to the [docker network documentation](https://docs.docker.com/network/), how to adjust this inside a container.

## ⚒️ Installing from source

First of all, [download](https://golang.org/dl/) and install **Go**. Version `1.17` or higher is required.

Installation is done by using the [`go install`](https://golang.org/cmd/go/#hdr-Compile_and_install_packages_and_dependencies) command and rename installed binary in `$GOPATH/bin`:

```bash
go install github.com/nightshift2k/binance-proxy/cmd/binance-proxy
```

## 📖 Basic Usage
The proxy listens automatically on port **8090** for Spot markets, and port **8091** for Futures markets. Available options for parametrizations are available via `-h`
```
Usage:
  binance-proxy [OPTIONS]

Application Options:
  -v, --verbose                Verbose output (increase with -vv) [$BPX_VERBOSE]
  -p, --port-spot=             Port to which to bind for SPOT markets (default: 8090) [$BPX_PORT_SPOT]
  -t, --port-futures=          Port to which to bind for FUTURES markets (default: 8091) [$BPX_PORT_FUTURES]
  -c, --disable-fake-candles   Disable generation of fake candles (ohlcv) when sockets have not delivered data yet [$BPX_DISABLE_FAKE_CANDLES]
  -s, --disable-spot           Disable proxying spot markets [$BPX_DISABLE_SPOT]
  -f, --disable-futures        Disable proxying futures markets [$BPX_DISABLE_FUTURES]
  -a, --always-show-forwards   Always show requests forwarded via REST even if verbose is disabled [$BPX_ALWAYS_SHOW_FORWARDS]

Help Options:
  -h, --help                   Show this help message
```
### 🪙 Example Usage with Freqtrade
**Freqtrade** needs to be aware, that the **API endpoint** for querying the exchange is not the public endpoint, which is usually `https://api.binance.com` but instead queries are being proxied. To achieve that, the appropriate `config.json` needs to be adjusted in the `{ exchange: { urls: { api: public: "..."} } }` section.

```json
{
    "exchange": {
        "name": "binance",
        "key": "",
        "secret": "",
        "ccxt_config": {
            "enableRateLimit": false,
            "urls": {
                "api": {
                    "public": "http://127.0.0.1:8090/api/v3"
                }
            }
        },
        "ccxt_async_config": {
            "enableRateLimit": false
        }
    }
}
```
This example assumes, that `binance-proxy` is running on the same host as the consuming application, thus `localhost` or `127.0.0.1` is used as the target address. Should `binance-proxy` run in a separate 🐳 **Docker** container, a separate instance or a k8s pod, the target address has to be replaced respectively, and it needs to be ensured that the required ports (8090/8091 per default) are opened for requests.

## ➡️ Supported API endpoints for caching
| Endpoint | Market | Purpose |  Socket Update Interval | Comments |
|----------|--------|---------|----------|---------|
|`/api/v3/klines`<br/>`/fapi/v1/klines`| spot/futures | Kline/candlestick bars for a symbol|~ 2s|Websocket is closed if there is no following request after `2 * interval_time` (for example: A websocket for a symbol on `5m` timeframe is closed after 10 minutes.<br/><br/>Following requests for `klines` can not be delivered from the websocket cache:<br/><li>`limit` parameter is > 1000</ul><li>`startTime` or `endTime` have been specified</ul>|
|`/api/v3/depth`<br/>`/fapi/v1/depth`|spot/futures|Order Book (Depth)|100ms|Websocket is closed if there is no following request after 2 minutes.<br/><br/>The `depth` endpoint serves only a maximum depth of 20.|
|`/api/v3/ticker/24hr`|spot|24hr ticker price change statistics|2s/100ms (see comments)|Websocket is closed if there is no following request after 2 minutes.<br/><br/>For faster updates the values for <li>`lastPrice`</ul><li>`bidPrice`</ul><li>`askPrice`</ul><br>are taken from the `bookTicker` which is updated in an interval of 100ms.|
|`/api/v3/exchangeInfo`<br/>`/fapi/v1/exchangeInfo`| spot/futures| Current exchange trading rules and symbol information|60s (see comments)|`exchangeInfo` is fetched periodically via REST every 60 seconds. It is not a websocket endpoint but just being cached during runtime.|

> 🚨 Every **other** REST query to an endpoint is being **forwarded** 1:1 to the **API** at https://api.binance.com !


## ⚙️ Commands & Options

The following parameters are available to control the behavior of **binance-proxy**:
```bash
binance-proxy [OPTION]
```

| Option | Environment Variable | Description                                              | Type   | Default | Required? |
| ------ | ------------------|-------------------------------------- | ------ | ------- | --------- |
| `-v`   | `$BPX_VERBOSE` | Sets the verbosity to debug level. | `bool` | `false` | No        |
| `-vv`  |`$BPX_VERBOSE`| Sets the verbosity to trace level. | `bool` | `false` | No        |
| `-p`   |`$BPX_PORT_SPOT`| Specifies the listen port for **SPOT** market proxy. | `int` | `8090` | No        |
| `-t`   |`$BPX_PORT_FUTURES`| Specifies the listen port for **FUTURES** market proxy. | `int` | `8091` | No        |
| `-c`   |`$BPX_DISABLE_FAKE_CANDLES`| Disables the generation of fake candles, when not yet recieved through websockets. | `bool` | `false` | No        |
| `-s`   |`$BPX_DISABLE_SPOT`| Disables proxy for **SPOT** markets. | `bool` | `false` | No        |
| `-f`   |`$BPX_DISABLE_FUTURES`| Disables proxy for **FUTURES** markets. | `bool` | `false` | No        |
| `-a`   |`$BPX_ALWAYS_SHOW_FORWARDS`| Always show requests forwarded via REST even if verbose is disabled | `bool` | `false` | No        |

Instead of using command line switches environment variables can be used, there are several ways how those can be implemented. For example `.env` files could be used in combination with `docker-compose`. 

Passing variables to a docker container can also be achieved in different ways, please see the documentation for all available options [here](https://docs.docker.com/compose/environment-variables/).

## 🐞 Bug / Feature Request

If you find a bug (the proxy couldn't handle the query and / or gave undesired results), kindly open an issue [here](https://github.com/nightshift2k/binance-proxy/issues/new) by including a **logfile** and a **meaningful description** of the problem.

If you'd like to request a new function, feel free to do so by opening an issue [here](https://github.com/nightshift2k/binance-proxy/issues/new). 

## 💻 Development
Want to contribute? **Great!🥳**

To fix a bug or enhance an existing module, follow these steps:

- Fork the repo
- Create a new branch (`git checkout -b improve-feature`)
- Make the appropriate changes in the files
- Add changes to reflect the changes made
- Commit your changes (`git commit -am 'Improve feature'`)
- Push to the branch (`git push origin improve-feature`)
- Create a Pull Request

## 🙏 Credits
+ [@adrianceding](https://github.com/adrianceding) for creating the original version, available [here](https://github.com/adrianceding/binance-proxy).

## ⚠️ License

`binance-proxy` is free and open-source software licensed under the [MIT License](https://github.com/nightshift2k/binance-proxy/blob/main/LICENSE). 

By submitting a pull request to this project, you agree to license your contribution under the MIT license to this project.

### 🧬 Third-party library licenses
+ [go-binance](https://github.com/adshao/go-binance/blob/master/LICENSE)
+ [go-flags](https://github.com/jessevdk/go-flags/blob/master/LICENSE)
+ [logrus](https://github.com/sirupsen/logrus/blob/master/LICENSE)
+ [go-time](https://cs.opensource.google/go/x/time/+/master:LICENSE)
+ [go-simplejson](https://github.com/bitly/go-simplejson/blob/master/LICENSE)
+ [websocket](https://github.com/gorilla/websocket/blob/master/LICENSE)
+ [objx](https://github.com/stretchr/objx/blob/master/LICENSE)
+ [testify](https://github.com/stretchr/testify/blob/master/LICENSE)