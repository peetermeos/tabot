package kraken

import (
	"bytes"
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
	"time"

	"github.com/gorilla/websocket"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	contentApplicationJSON = "application/json"

	krakenWsURL   = "wss://ws.kraken.com/v2"
	krakenAuthURL = "https://api.kraken.com/0/private/GetWebSocketsToken"

	httpTimeout = 10 * time.Second
)

type authRequest struct {
	Nonce int64 `json:"nonce"`
}

type authResponse struct {
	Error  []interface{} `json:"error"`
	Result struct {
		Token   string `json:"token"`
		Expires int    `json:"expires"` // seconds
	} `json:"result"`
}

type level1Request struct {
	Method string `json:"method"`
	Params struct {
		Channel string   `json:"channel"`
		Symbol  []string `json:"symbol"`
	} `json:"params"`
}

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

func (c *Client) authenticate(ctx context.Context) error {
	nonce := time.Now().Unix()

	reqBody := authRequest{
		Nonce: nonce,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrap(err, "error marshaling request body")
	}

	b64DecodedSecret, err := base64.StdEncoding.DecodeString(c.apiSecret)
	if err != nil {
		return errors.Wrap(err, "error decoding secret")
	}

	signature := getKrakenSignature(krakenAuthURL,
		url.Values{"nonce": {fmt.Sprintf("%d", nonce)}},
		b64DecodedSecret)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, krakenAuthURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return errors.Wrap(err, "error creating request")
	}

	req.Header.Add("Content-Type", contentApplicationJSON)
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

	bodyBytes, err = io.ReadAll(resp.Body)
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
