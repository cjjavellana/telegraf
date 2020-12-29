# Coinbase Market Data Websocket Listener
Opens a websocket connection to coinbase and receive realtime market data.

## Plugin Parameters

`service_address` - The websocket address of coinbase's matching engine

`on_connect_msg` - The subscription message to be sent to coinbase upon successful connection. 
See [this](https://docs.pro.coinbase.com/?r=1#subscribe) for more details and on how to customize it.

## Getting Started
1. Install Telegraf
   ```bash
   $ go install telegraf 
    ```
   
2. Clone this repository
    ```bash
   $ cd $GOPATH/src/telegraf/plugins/inputs
   $ git clone <this repository>
    ```
   
3. Edit all/all.go
    ```bash
    $ vim all/all.go
    ```

   Add
   ```go 
    _ "github.com/influxdata/telegraf/plugins/inputs/coinbase_marketdata"
   ```
4. Compile
    ```bash
    $ cd $GOPATH/src/telegraf
    $ make telegraf
    $ ./telegraf -sample-config -input-filter coinbase_marketdata -output-filter influxdb -debug > telegraf.conf.test
    ```
5. Start InfluxDB & Grafana
    ```bash
    $ cd $GOPATH/src/telegraf
    $ docker-compose up -d influxdb grafana
    ```
6. Start Telegraf
    ```shell
    $ cd $GOPATH/src/telegraf
    $ ./telegraf -config telegraf.conf.test -debug
    ```

## Sample Responses

Ticker
```json
{
  "type": "ticker",
  "sequence": 12238444095,
  "product_id": "ETH-USD",
  "price": "731.99",
  "open_24h": "684.11",
  "volume_24h": "395831.08785795",
  "low_24h": "680.9",
  "high_24h": "747",
  "volume_30d": "6144317.83380943",
  "best_bid": "731.83",
  "best_ask": "731.99",
  "side": "buy",
  "time": "2020-12-28T23:54:32.051347Z",
  "trade_id": 71476932,
  "last_size": "0.24169456"
}
```

l2update
```json
{
  "type": "l2update",
  "product_id": "ETH-USD",
  "changes": [
    [
      "sell",
      "731.99",
      "1.24025886"
    ]
  ],
  "time": "2020-12-28T23:54:32.051347Z"
}
```
