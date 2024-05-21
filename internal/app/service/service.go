package service

import (
	"context"

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
}

type BotInput struct {
	Logger     logrus.FieldLogger
	MarketData MarketDataProvider
	Execution  ExecutionProvider
}

func NewTriangleBot(input BotInput) *TriangleBot {
	tabot := &TriangleBot{
		logger:     input.Logger.WithField("comp", "tabot"),
		marketData: input.MarketData,
		trader:     input.Execution,
	}

	return tabot
}

func (t *TriangleBot) Run(ctx context.Context) {
	// TODO: Just a placeholder for now
	//   we will be constructing triangle legs via
	//   matrix multiplication of exchange rates
	zero := mat.NewDense(3, 5, nil)

	t.logger.Info(zero)

	dataStream := t.marketData.Stream(ctx)

	err := t.marketData.Subscribe("BTC/USD")
	if err != nil {
		t.logger.WithError(err).Error("failed to subscribe to BTC/USD")
	}

	err = t.marketData.Subscribe("ETH/USD")
	if err != nil {
		t.logger.WithError(err).Error("failed to subscribe to ETH/USD")
	}

	for tick := range dataStream {
		t.logger.WithField("tick", tick).Info("received tick")
	}
}
