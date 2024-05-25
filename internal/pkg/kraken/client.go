package kraken

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/peetermeos/tabot/internal/app/service"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	contentApplicationJSON = "application/json"
	contentURLEncoded      = "application/x-www-form-urlencoded"

	krakenWsURL    = "wss://ws.kraken.com/v2"
	krakenBaseURL  = "https://api.kraken.com"
	krakenAuthPath = "/0/private/GetWebSocketsToken"

	httpTimeout = 10 * time.Second
)

type authResponse struct {
	Error  []interface{} `json:"error"`
	Result struct {
		Token   string `json:"token"`
		Expires int    `json:"expires"` // seconds
	} `json:"result"`
}

type level1Request struct {
	Method string              `json:"method"`
	Params level1RequestParams `json:"params"`
}

type level1RequestParams struct {
	Channel string   `json:"channel"`
	Symbol  []string `json:"symbol"`
}

// level1Response is the L1 exchange rate response from Kraken.
// Sample:
//
//	 {
//			"channel":"ticker",
//			"type":"snapshot",
//			"data":[{
//				"symbol":"BTC/GBP",
//				"bid":53975.7,
//				"bid_qty":0.00282754,
//				"ask":53975.8,
//				"ask_qty":2.79487918,
//				"last":53975.7,
//				"volume":53.42371402,
//				"vwap":53299.8,
//				"low":52499.9,
//				"high":54357.1,
//				"change":1095.7,
//				"change_pct":2.07,
//			}]
//		}
type level1Response struct {
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
	logger      logrus.FieldLogger
	apiKey      string
	apiSecret   string
	token       string
	tokenExpiry time.Time
	conn        *websocket.Conn
}

var ErrAuthFailed = errors.New("authentication failed")

func NewClient(ctx context.Context, logger logrus.FieldLogger, apiKey string, apiSecret string) *Client {
	c := &Client{
		logger: logger.WithFields(logrus.Fields{
			"comp": "kraken-client",
		}),
		apiKey:    apiKey,
		apiSecret: apiSecret,
	}

	err := c.connect(ctx)
	if err != nil {
		c.logger.WithError(err).Error("error connecting")
	}

	return c
}

// Stream returns a channel of ticks from the Kraken websocket.
// Sample response for BTC/GBP:
//
//	ask=53975.8 base=GBP bid=53975.7 instrument=BTC
func (c *Client) Stream(ctx context.Context) <-chan service.Tick {
	tickCh := make(chan service.Tick)

	go func() {
		defer close(tickCh)

		if c.conn == nil {
			err := c.connect(ctx)
			if err != nil {
				c.logger.WithError(err).Error("error connecting")

				return
			}
		}

		for {
			_, payload, err := c.conn.ReadMessage()
			if err != nil {
				c.logger.WithError(err).Error("error reading message from websocket")

				return
			}

			c.logger.WithFields(logrus.Fields{
				"action":  "read_message",
				"payload": string(payload),
			}).Debug("received message")

			var unmarshalled level1Response

			err = json.Unmarshal(payload, &unmarshalled)
			if err != nil {
				c.logger.WithFields(logrus.Fields{
					"action":  "unmarshal_message",
					"payload": string(payload),
				}).WithError(err).Error("error unmarshalling message")

				continue
			}

			if unmarshalled.Channel == "ticker" {
				for _, data := range unmarshalled.Data {
					tickCh <- service.Tick{
						Symbol: data.Symbol,
						Bid:    data.Bid,
						BidQty: data.BidQty,
						Ask:    data.Ask,
						AskQty: data.AskQty,
					}
				}
			}

			select {
			case <-ctx.Done():
				err = c.conn.Close()
				if err != nil {
					c.logger.WithError(err).Error("error closing websocket connection")
				}

				return
			default:
			}
		}
	}()

	return tickCh
}

func (c *Client) Subscribe(symbol string) error {
	if c.conn == nil {
		err := c.connect(context.Background())
		if err != nil {
			return errors.Wrap(err, "error connecting")
		}
	}

	req := level1Request{
		Method: "subscribe",
		Params: level1RequestParams{
			Channel: "ticker",
			Symbol:  []string{symbol},
		},
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return errors.Wrap(err, "error marshalling request")
	}

	err = c.conn.WriteMessage(websocket.TextMessage, reqBody)
	if err != nil {
		return errors.Wrap(err, "error writing message to websocket")
	}

	return nil
}

func (c *Client) Unsubscribe(_ string) error {
	return nil
}

// authenticate sends a request to Kraken to authenticate the client for websocket
// communication. The client's API key and secret are used to sign the request.
func (c *Client) authenticate(ctx context.Context) error {
	nonce := time.Now().UnixMilli()

	reqBody := url.Values{}
	reqBody.Set("nonce", fmt.Sprintf("%d", nonce))

	b64DecodedSecret, err := base64.StdEncoding.DecodeString(c.apiSecret)
	if err != nil {
		return errors.Wrap(err, "error decoding secret")
	}

	signature := getKrakenSignature(krakenAuthPath,
		url.Values{"nonce": {fmt.Sprintf("%d", nonce)}},
		b64DecodedSecret)

	urlStr := krakenBaseURL + krakenAuthPath

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlStr, strings.NewReader(reqBody.Encode()))
	if err != nil {
		return errors.Wrap(err, "error creating request")
	}

	req.Header.Add("Content-Type", contentURLEncoded)
	req.Header.Add("Accept", contentApplicationJSON)
	req.Header.Add("API-Key", c.apiKey)
	req.Header.Add("API-Sign", signature)

	httpClient := http.Client{
		Timeout: httpTimeout,
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "error sending request")
	}

	defer func() { _ = resp.Body.Close() }()

	var unmarshalled authResponse

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "error reading response body")
	}

	err = json.Unmarshal(bodyBytes, &unmarshalled)
	if err != nil {
		return errors.Wrap(err, "error unmarshalling response body")
	}

	if len(unmarshalled.Error) > 0 {
		return errors.Wrapf(ErrAuthFailed, "error: %v", any(unmarshalled.Error))
	}

	c.token = unmarshalled.Result.Token
	c.tokenExpiry = time.Now().
		Add(time.Duration(unmarshalled.Result.Expires) * time.Second)

	return nil
}

func (c *Client) connect(ctx context.Context) error {
	if time.Since(c.tokenExpiry) > 0 {
		err := c.authenticate(ctx)
		if err != nil {
			return errors.Wrap(err, "error authenticating")
		}
	}

	c.logger.WithFields(logrus.Fields{
		"action":   "connect",
		"endpoint": krakenWsURL,
	}).Info("connecting")

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

func getKrakenSignature(urlPath string, values url.Values, secret []byte) string {
	sha := sha256.New()
	sha.Write([]byte(values.Get("nonce") + values.Encode()))
	shaSum := sha.Sum(nil)

	mac := hmac.New(sha512.New, secret)
	mac.Write(append([]byte(urlPath), shaSum...))

	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}
