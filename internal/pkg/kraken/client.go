package kraken

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
