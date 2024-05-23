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

type ExecutionInput struct {
	Symbol string
	Base   string
	Side   string
	Qty    float64
}

type ExecutionProvider interface {
	Execute(ctx context.Context, input ExecutionInput) error
}

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
	dim := len(t.symbols)
	exch := mat.NewDense(dim, dim, nil)

	// Initialize exchange matrix with identity matrix
	for i := 0; i < dim; i++ {
		exch.Set(i, i, 1)
	}

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
		exch.Set(index(instrument, t.symbols), index(base, t.symbols), 1/tick.Ask)

		// TODO: Implement triangular arbitrage opportunity detection
		// TODO: Execute trades

		const capital = 1000

		// Always start at USD
		leg1Idx := index("USD", t.symbols)

		for leg2Idx := 0; leg2Idx < dim; leg2Idx++ {
			for leg3Idx := 0; leg3Idx < dim; leg3Idx++ {
				if leg2Idx == leg3Idx || leg2Idx == leg1Idx || leg3Idx == leg1Idx {
					continue
				}

				leg1 := t.symbols[leg1Idx]
				leg2 := t.symbols[leg2Idx]
				leg3 := t.symbols[leg3Idx]

				if leg1 != instrument && leg2 != instrument && leg3 != instrument {
					continue
				}

				leg1Leg2Leg3 := exch.At(leg2Idx, leg1Idx) * exch.At(leg3Idx, leg2Idx)
				leg3Leg1 := exch.At(leg1Idx, leg3Idx)

				if leg1Leg2Leg3 == 0 || leg3Leg1 == 0 {
					continue
				}

				t.logger.WithFields(logrus.Fields{
					"leg1":         leg1,
					"leg2":         leg2,
					"leg3":         leg3,
					"leg1Leg2Leg3": fmt.Sprintf("%.4f", capital*leg1Leg2Leg3),
					"leg3Leg1":     fmt.Sprintf("%.4f", capital*(1/leg3Leg1)),
					"delta usd":    fmt.Sprintf("%.4f", capital*(1/leg3Leg1-leg1Leg2Leg3)*leg3Leg1),
				}).Infof("calculated rates for %s/%s/%s", leg1, leg2, leg3)
			}
		}
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
