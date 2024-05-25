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

type ExecutionProvider interface {
	Execute(ctx context.Context, input ExecutionInput) error
	TotalCapital() float64
}

type ExecutionInput struct {
	Symbol string
	Base   string
	Side   string
	Rate   float64
	Qty    float64
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

	// Initialize exchange matrix as identity matrix
	for i := 0; i < dim; i++ {
		exch.Set(i, i, 1)
	}

	dataStream := t.marketData.Stream(ctx)

	// Subscribe to all pairs
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

		// Sample response for BTC/GBP:
		//
		//	ask=53975.8 base=GBP bid=53975.7 instrument=BTC
		//  From here:
		//    1 GBP -> 1/53975.8 BTC (ie 1/ask)
		//    1 BTC -> 53975.7 GBP (ie bid)

		// Update exchange rates:
		// - buy instrument, sell base at this rate, exchange matrix upper triangle
		exch.Set(index(base, t.symbols), index(instrument, t.symbols), 1/tick.Ask)

		// - sell instrument, buy base at this rate, exchange matrix lower triangle
		exch.Set(index(instrument, t.symbols), index(base, t.symbols), tick.Bid)

		// The convention for the exchange rate matrix is:
		// you always go from row to column. Matrix element is the respective
		// exchange rate.

		// Always start at USD
		leg1Idx := index("USD", t.symbols)

		for leg2Idx := 0; leg2Idx < dim && leg2Idx != leg1Idx; leg2Idx++ {
			for leg3Idx := 0; leg3Idx < dim && leg2Idx != leg3Idx && leg3Idx != leg1Idx; leg3Idx++ {
				leg1 := t.symbols[leg1Idx]
				leg2 := t.symbols[leg2Idx]
				leg3 := t.symbols[leg3Idx]

				if leg1 != instrument && leg2 != instrument && leg3 != instrument {
					continue
				}

				// leg1 -> leg2: buy leg2, sell leg1
				// leg2 -> leg3: buy leg3, sell leg2
				leg1Leg2Leg3 := exch.At(leg1Idx, leg2Idx) * exch.At(leg2Idx, leg3Idx)

				// leg3 -> leg1: buy leg1, sell leg3
				leg3Leg1 := exch.At(leg3Idx, leg1Idx)

				if leg1Leg2Leg3 == 0 || leg3Leg1 == 0 {
					continue
				}

				deltaPct := leg1Leg2Leg3 * leg3Leg1 * 100

				if isTradeable(deltaPct) {
					// TODO: Execute trades, make it look nicer

					//// Leg1
					//err := t.trader.Execute(ctx, ExecutionInput{
					//	Symbol: leg2,
					//	Base:   leg1,
					//	Side:   "sell",
					//	Rate:   1 / exch.At(leg2Idx, leg1Idx),
					//})
					//if err != nil {
					//	t.logger.WithFields(logrus.Fields{
					//		"leg1": leg1,
					//	}).WithError(err).Error("failed to execute trade")
					//}
					//// Leg2
					//err = t.trader.Execute(ctx, ExecutionInput{
					//	Symbol: leg3,
					//	Base:   leg2,
					//	Side:   "sell",
					//	Rate:   1 / exch.At(leg3Idx, leg2Idx),
					//})
					//if err != nil {
					//	t.logger.WithFields(logrus.Fields{
					//		"leg2": leg2,
					//	}).WithError(err).Error("failed to execute trade")
					//}
					//
					//// Leg3
					//err = t.trader.Execute(ctx, ExecutionInput{
					//	Symbol: leg3,
					//	Base:   leg1,
					//	Side:   "buy",
					//	Rate:   1 / exch.At(leg3Idx, leg1Idx),
					//})
					//if err != nil {
					//	t.logger.WithFields(logrus.Fields{
					//		"leg3": leg3,
					//	}).WithError(err).Error("failed to execute trade")
					//}

					t.logger.WithFields(logrus.Fields{
						"leg1": leg1,
						"leg2": leg2,
						"leg3": leg3,
						// "leg1Leg2Leg3": fmt.Sprintf("%.4f", capital*leg1Leg2Leg3),
						//"leg3Leg1":     fmt.Sprintf("%.4f", capital*(1/leg3Leg1)),
						"delta_pct": fmt.Sprintf("%.2f", deltaPct-100),
						//	"capital_usd":  fmt.Sprintf("%.4f", t.trader.TotalCapital()),
					}).Infof("calculated rates for %s/%s/%s", leg1, leg2, leg3)
				}
			}
		}
	}
}

func isTradeable(delta float64) bool {
	// TODO: Implement triangular arbitrage opportunity detection

	return delta > 100.2
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
