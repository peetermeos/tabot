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

		//fa := mat.Formatted(exch, mat.Prefix("    "), mat.Squeeze())
		//
		//// and then print with and without zero value elements.
		//fmt.Printf("rates:\nA = % .4f\n\n", fa)

		usdBtcEth := exch.At(index("BTC", t.symbols), index("USD", t.symbols)) *
			exch.At(index("ETH", t.symbols), index("BTC", t.symbols))
		ethUsd := exch.At(index("USD", t.symbols), index("ETH", t.symbols))

		usdBtcSol := exch.At(index("BTC", t.symbols), index("USD", t.symbols)) *
			exch.At(index("SOL", t.symbols), index("BTC", t.symbols))
		solUsd := exch.At(index("USD", t.symbols), index("SOL", t.symbols))

		usdEthSol := exch.At(index("ETH", t.symbols), index("USD", t.symbols)) *
			exch.At(index("SOL", t.symbols), index("ETH", t.symbols))

		usdLtcEth := exch.At(index("LTC", t.symbols), index("USD", t.symbols)) *
			exch.At(index("ETH", t.symbols), index("LTC", t.symbols))

		t.logger.WithFields(logrus.Fields{
			"usdBtcEth": fmt.Sprintf("%.4f", 1/usdBtcEth),
			"ethUsd":    fmt.Sprintf("%.4f", ethUsd),
			"delta":     fmt.Sprintf("%.4f", ethUsd-1/usdBtcEth),
		}).Info("calculated rates for USD/BTC/ETH")

		t.logger.WithFields(logrus.Fields{
			"usdBtcSol": fmt.Sprintf("%.4f", 1/usdBtcSol),
			"solUsd":    fmt.Sprintf("%.4f", solUsd),
			"delta":     fmt.Sprintf("%.4f", solUsd-1/usdBtcSol),
		}).Info("calculated rates for USD/BTC/SOL")

		t.logger.WithFields(logrus.Fields{
			"usdEthSol": fmt.Sprintf("%.4f", 1/usdEthSol),
			"solUsd":    fmt.Sprintf("%.4f", solUsd),
			"delta":     fmt.Sprintf("%.4f", solUsd-1/usdEthSol),
		}).Info("calculated rates for USD/ETH/SOL")

		t.logger.WithFields(logrus.Fields{
			"usdLtcEth": fmt.Sprintf("%.4f", 1/usdLtcEth),
			"ethUsd":    fmt.Sprintf("%.4f", ethUsd),
			"delta":     fmt.Sprintf("%.4f", ethUsd-1/usdLtcEth),
		}).Info("calculated rates for USD/LTC/ETH")
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
