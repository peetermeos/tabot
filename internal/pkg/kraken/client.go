package kraken

import (
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	krakenWsURL = "wss://ws.kraken.com/v2"
)

type Level1Request struct {
	Method string `json:"method"`
	Params struct {
		Channel string   `json:"channel"`
		Symbol  []string `json:"symbol"`
	} `json:"params"`
}

type Level1Response struct {
	Channel string `json:"channel"`
	Type    string `json:"type"`
	Data    []struct {
		Symbol    string  `json:"symbol"`
		Bid       float64 `json:"bid"`
		BidQty    float64 `json:"bid_qty"`
		Ask       float64 `json:"ask"`
		AskQty    float64 `json:"ask_qty"`
		Last      float64 `json:"last"`
		Volume    float64 `json:"volume"`
		Vwap      float64 `json:"vwap"`
		Low       float64 `json:"low"`
		High      float64 `json:"high"`
		Change    float64 `json:"change"`
		ChangePct float64 `json:"change_pct"`
	} `json:"data"`
}

type Client struct {
	logger logrus.FieldLogger
	conn   *websocket.Conn
}

func NewClient(logger logrus.FieldLogger) *Client {
	c := &Client{
		logger: logger.WithFields(logrus.Fields{
			"comp": "kraken-client",
		}),
	}

	return c
}

func (c *Client) connect() error {
	c.logger.WithFields(logrus.Fields{
		"action":   "connect",
		"endpoint": krakenWsURL,
	}).
		Info("connecting")

	h := http.Header{}

	//nolint:bodyclose
	conn, _, err := websocket.DefaultDialer.Dial(krakenWsURL, h)
	if err != nil {
		return errors.Wrap(err, "error connecting to websocket")
	}

	c.conn = conn

	c.conn.SetPongHandler(func(msg string) error {
		c.logger.WithFields(logrus.Fields{"action": "pong", "msg": msg}).
			Debug("received pong")

		return nil
	})

	c.logger.WithFields(logrus.Fields{"action": "connect"}).
		Info("success")

	return nil
}
