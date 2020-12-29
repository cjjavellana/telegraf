package coinbase_marketdata

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/parsers"
	"io"
	"log"
	"strconv"
	"sync"
)

type Ticker struct {
	DataType   string  `json:"type"`
	ProductId  string  `json:"product_id"`
	Side       string  `json:"side"`
	Time       string  `json:"time"`
	Price      float64 `json:"price"`
	Open24H    float64 `json:"open_24h"`
	Volume24H  float64 `json:"volume_24h"`
	Low24H     float64 `json:"low_24h"`
	High24H    float64 `json:"high_24h"`
	Volume30D  float64 `json:"volume_30d"`
	BestBid    float64 `json:"best_bid"`
	BestAsk    float64 `json:"best_ask"`
	Size       float64 `json:"last_size"`
	SequenceId int64   `json:"sequence_id"`
	TradeId    int64   `json:"trade_id"`
}

type L2Update struct {
	DataType  string  `json:"type"`
	ProductId string  `json:"product_id"`
	Side      string  `json:"side"`
	Price     float64 `json:"price"`
	Qty       float64 `json:"qty"`
	Time      string  `json:"time"`
}

type WebSocketListener struct {
	ServiceAddress string `toml:"service_address"`
	OnConnectMsg   string `toml:"on_connect_msg"`

	done chan bool

	conn *websocket.Conn
	wg   sync.WaitGroup

	// Mixins
	parsers.Parser
	telegraf.Accumulator
	io.Closer
}

// The telegraf Input Interface Implementation

func (wsl *WebSocketListener) SampleConfig() string {
	return `
## Websocket URL to connect to
service_address = "wss://ws-feed.pro.coinbase.com"
data_format = "json"
json_name_key = "type"
json_time_key = "time"
json_time_format = "2006-01-02T15:04:05.000000Z"
tag_keys = [
	"type", 
	"product_id", 
	"side"
]
json_string_fields = [
	"type", 
	"product_id", 
	"side"
]
json_query = ".changes"
on_connect_msg = '''
{ 
	"type": "subscribe", 
	"product_ids": [ 
		"ETH-USD" 
	], 
	"channels": [ 
		"level2", 
		"heartbeat", 
		{ 
			"name": "ticker", 
			"product_ids": [ 
				"ETH-USD" 
			] 
		} 
	] 
}
'''
`
}

func (wsl *WebSocketListener) Description() string {
	return "Opens a websocket connection to a server and receives updates"
}

func (wsl *WebSocketListener) Gather(_ telegraf.Accumulator) error {
	return nil
}

func (wsl *WebSocketListener) SetParser(parser parsers.Parser) {
	wsl.Parser = parser
}

func (wsl *WebSocketListener) Start(acc telegraf.Accumulator) error {
	wsl.Accumulator = acc

	log.Print("Service Address: ", wsl.ServiceAddress)
	log.Print("Subscription Request: ", wsl.OnConnectMsg)

	c, _, err := websocket.DefaultDialer.Dial(wsl.ServiceAddress, nil)
	if err != nil {
		log.Fatal("dial:", err)
		return err
	}
	wsl.conn = c

	// start the routine for reading incoming data stream
	go wsl.read()

	//
	err = wsl.subscribe()
	if err != nil {
		log.Fatal("subscribe:", err)
		return err
	}

	return nil
}

// takes in a map of l2update data type in the format of
// {
//  "type": "l2update",
//  "product_id": "ETH-USD",
//  "changes": [
//    [
//      "sell",
//      "731.99",
//      "1.24025886"
//    ]
//  ],
//  "time": "2020-12-28T23:54:32.051347Z"
// }
func (wsl *WebSocketListener) parseL2Update(l2UpdateData map[string]interface{}) []L2Update {
	changes := l2UpdateData["changes"].([]interface{})

	var updates []L2Update

	for _, c := range changes {
		change := c.([]interface{})

		side := fmt.Sprintf("%v", change[0])
		price, _ := strconv.ParseFloat(fmt.Sprintf("%v", change[1]), 64)
		qty, _ := strconv.ParseFloat(fmt.Sprintf("%v", change[2]), 64)

		updates = append(updates, L2Update{
			DataType:  fmt.Sprintf("%v", l2UpdateData["type"]),
			ProductId: fmt.Sprintf("%v", l2UpdateData["product_id"]),
			Time:      fmt.Sprintf("%v", l2UpdateData["time"]),
			Side:      side,
			Price:     price,
			Qty:       qty,
		})
	}

	return updates
}

// takes in a map of ticker data type in the format of
// {
//  "type": "ticker",
//  "sequence": 12238444095,
//  "product_id": "ETH-USD",
//  "price": "731.99",
//  "open_24h": "684.11",
//  "volume_24h": "395831.08785795",
//  "low_24h": "680.9",
//  "high_24h": "747",
//  "volume_30d": "6144317.83380943",
//  "best_bid": "731.83",
//  "best_ask": "731.99",
//  "side": "buy",
//  "time": "2020-12-28T23:54:32.051347Z",
//  "trade_id": 71476932,
//  "last_size": "0.24169456"
// }
func (wsl *WebSocketListener) parseTicker(tickerData map[string]interface{}) *Ticker {
	open24H, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["open_24h"]), 64)
	volume24H, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["volume_24h"]), 64)
	low24H, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["low_24h"]), 64)
	high24H, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["high_24h"]), 64)
	volume30D, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["volume_30d"]), 64)
	bestBid, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["best_bid"]), 64)
	bestAsk, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["best_ask"]), 64)
	sequenceId, _ := strconv.ParseInt(fmt.Sprintf("%v", tickerData["sequence"]), 10, 64)
	tradeId, _ := strconv.ParseInt(fmt.Sprintf("%v", tickerData["trade_id"]), 10, 64)
	size, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["last_size"]), 64)
	price, _ := strconv.ParseFloat(fmt.Sprintf("%v", tickerData["price"]), 64)

	return &Ticker{
		DataType:   fmt.Sprintf("%v", tickerData["type"]),
		ProductId:  fmt.Sprintf("%v", tickerData["product_id"]),
		Side:       fmt.Sprintf("%v", tickerData["side"]),
		Time:       fmt.Sprintf("%v", tickerData["time"]),
		Price:      price,
		Open24H:    open24H,
		Volume24H:  volume24H,
		Low24H:     low24H,
		High24H:    high24H,
		Volume30D:  volume30D,
		BestBid:    bestBid,
		BestAsk:    bestAsk,
		Size:       size,
		SequenceId: sequenceId,
		TradeId:    tradeId,
	}
}

func (wsl *WebSocketListener) read() {

	for {
		select {
		case <-wsl.done:
			return

		default:
			_, message, err := wsl.conn.ReadMessage()
			if err != nil {
				log.Println("read: ", err)
				return
			}

			log.Printf("recv: %s\n", message)

			go wsl.addMetric(message)
		}
	}
}

func (wsl *WebSocketListener) addMetric(message []byte) {
	marketData := make(map[string]interface{})
	err := json.Unmarshal(message, &marketData)
	if err != nil {
		wsl.AddError(fmt.Errorf("unable to parse incoming msg: %s", err))
		return
	}

	var data []byte

	if marketData["type"] == "ticker" {
		data, _ = json.Marshal(wsl.parseTicker(marketData))
	} else if marketData["type"] == "l2update" {
		l2Updates := wsl.parseL2Update(marketData)
		for _, updates := range l2Updates {
			data, _ = json.Marshal(updates)
		}
	}

	if data != nil {
		metrics, err := wsl.Parser.Parse(data)
		if err != nil {
			wsl.AddError(fmt.Errorf("unable to parse incoming msg: %s", err))
			return
		}

		for _, m := range metrics {
			wsl.AddMetric(m)
		}
	}
}

func (wsl *WebSocketListener) subscribe() error {
	return wsl.conn.WriteMessage(websocket.TextMessage, []byte(wsl.OnConnectMsg))
}

func (wsl *WebSocketListener) Stop() {
	wsl.done <- true
	if wsl.Closer != nil {
		_ = wsl.Close()
		wsl.Closer = nil
	}
	wsl.wg.Wait()
}

func newSocketListener() *WebSocketListener {
	parser, _ := parsers.NewInfluxParser()

	return &WebSocketListener{
		Parser: parser,
		done:   make(chan bool),
	}
}

func init() {
	inputs.Add("coinbase_marketdata", func() telegraf.Input { return newSocketListener() })
}
