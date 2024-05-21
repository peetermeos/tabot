package service

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/mat"
)

type MarketDataProvider interface {
	Stream(ctx context.Context) <-chan Tick
	Subscribe(symbol string) error
	Unsubscribe(symbol string) error
}

type ExecutionProvider interface{}

type Tick struct {
	Symbol string
	Bid    float64
	BidQty float64
	Ask    float64
	AskQty float64
}

type TriangleBot struct {
	logger     logrus.FieldLogger
	marketData MarketDataProvider
	trader     ExecutionProvider
	symbols    []string
}

type BotInput struct {
	Logger     logrus.FieldLogger
	MarketData MarketDataProvider
	Execution  ExecutionProvider
	Symbols    []string
}

func NewTriangleBot(input BotInput) *TriangleBot {
	tabot := &TriangleBot{
		logger:     input.Logger.WithField("comp", "tabot"),
		marketData: input.MarketData,
		trader:     input.Execution,
		symbols:    input.Symbols,
	}

	return tabot
}

func (t *TriangleBot) Run(ctx context.Context) {
	// TODO: Just a placeholder for now
	//   we will be constructing triangle legs via
	//   matrix multiplication of exchange rates

	dim := len(t.symbols)
	exch := mat.NewDense(dim, dim, nil)

	// Initialize exchange matrix with identity matrix
	for i := 0; i < dim; i++ {
		exch.Set(i, i, 1)
	}

	// TODO: Use Kronecker product to construct triangle legs

	dataStream := t.marketData.Stream(ctx)

	for idx1 := range t.symbols {
		for idx2 := idx1 + 1; idx2 < len(t.symbols); idx2++ {
			ticker := fmt.Sprintf("%s/%s", t.symbols[idx2], t.symbols[idx1])

			err := t.marketData.Subscribe(ticker)
			if err != nil {
				t.logger.WithError(err).Errorf("failed to subscribe to %s", ticker)
			}
		}
	}

	for tick := range dataStream {
		instrument, base := parsePair(tick.Symbol)
		t.logger.
			WithFields(logrus.Fields{
				"instrument": instrument,
				"base":       base,
				"bid":        tick.Bid,
				"ask":        tick.Ask,
			}).Debug("received tick")

		exch.Set(index(base, t.symbols), index(instrument, t.symbols), tick.Bid)

		fmt.Println(exch)
	}
}

func index(symbol string, syms []string) int {
	for i, s := range syms {
		if s == symbol {
			return i
		}
	}

	return -1
}

func parsePair(pair string) (string, string) {
	tickers := strings.Split(pair, "/")
	if len(tickers) != 2 {
		return "", ""
	}

	return tickers[0], tickers[1]
}
