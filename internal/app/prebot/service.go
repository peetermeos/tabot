package prebot

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
)

type MarketDataProvider interface {
	StreamBook(ctx context.Context) <-chan Book
	SubscribeBook(symbol string) error
	UnsubscribeBook(symbol string) error
}

type Book struct {
	Symbol   string
	IsUpdate bool
	Bids     []Level2Book
	Asks     []Level2Book
}

type Level2Book struct {
	Price     float64
	Volume    float64
	Timestamp float64
}

type PressureBot struct {
	logger logrus.FieldLogger
	data   MarketDataProvider
	symbol string

	// TODO: add fields for tracking the order book
	askBook map[float64]float64
	bidBook map[float64]float64

	position float64
	price    float64
}

type BotInput struct {
	Logger     logrus.FieldLogger
	MarketData MarketDataProvider
	Symbol     string
}

func NewPressureBot(input BotInput) *PressureBot {
	return &PressureBot{
		logger:  input.Logger.WithField("comp", "prebot"),
		data:    input.MarketData,
		symbol:  input.Symbol,
		askBook: make(map[float64]float64),
		bidBook: make(map[float64]float64),
	}
}

func (b *PressureBot) Run(ctx context.Context) {
	err := b.data.SubscribeBook(b.symbol)
	if err != nil {
		b.logger.WithError(err).Error("error subscribing to book")

		return
	}

	stream := b.data.StreamBook(ctx)

	for {
		select {
		case book := <-stream:
			b.logger.WithField("book", fmt.Sprintf("%+v", book)).Debug("received book")

			if !book.IsUpdate {
				b.askBook = make(map[float64]float64)
				b.bidBook = make(map[float64]float64)
			}

			for _, bid := range book.Bids {
				if bid.Volume == 0 {
					delete(b.bidBook, bid.Price)

					continue
				}

				b.bidBook[bid.Price] = bid.Volume
			}

			for _, ask := range book.Asks {
				if ask.Volume == 0 {
					delete(b.askBook, ask.Price)

					continue
				}

				b.askBook[ask.Price] = ask.Volume
			}

			totalBid := 0.0
			maxBid := 0.0

			for price, volume := range b.bidBook {
				totalBid += volume
				maxBid = math.Max(maxBid, price)
			}

			totalAsk := 0.0
			minAsk := 9999999999.0

			for price, volume := range b.askBook {
				totalAsk += volume
				minAsk = math.Min(minAsk, price)
			}

			const threshold = 15

			if totalBid-totalAsk > threshold {
				if b.position > 0 {
					continue
				}

				b.logger.WithFields(logrus.Fields{
					"total_bid": fmt.Sprintf("%.4f", totalBid),
					"total_ask": fmt.Sprintf("%.4f", totalAsk),
					"bid":       maxBid,
					"ask":       minAsk,
					"delta":     fmt.Sprintf("%.4f", totalBid-totalAsk),
				}).Info("enter long")

				b.position = 1
				b.price = minAsk
			}

			if totalAsk-totalBid > threshold {
				if b.position < 0 {
					continue
				}

				b.logger.WithFields(logrus.Fields{
					"total_bid": fmt.Sprintf("%.4f", totalBid),
					"total_ask": fmt.Sprintf("%.4f", totalAsk),
					"bid":       maxBid,
					"ask":       minAsk,
					"delta":     fmt.Sprintf("%.4f", totalBid-totalAsk),
				}).Info("enter short")

				b.position = -1
				b.price = maxBid
			}

			if maxBid >= b.price && b.position > 0 {
				b.logger.
					WithFields(logrus.Fields{
						"price": b.price,
						"bid":   maxBid,
					}).Info("exit long")

				b.position = 0
				b.price = 0
			}

			if minAsk <= b.price && b.position < 0 {
				b.logger.
					WithFields(logrus.Fields{
						"price": b.price,
						"ask":   minAsk,
					}).Info("exit short")

				b.position = 0
				b.price = 0
			}

			if minAsk < b.price && b.position > 0 {
				b.logger.
					WithFields(logrus.Fields{
						"price": b.price,
						"bid":   maxBid,
					}).Info("stop loss long")

				b.position = 0
				b.price = 0
			}

			if maxBid > b.price && b.position < 0 {
				b.logger.
					WithFields(logrus.Fields{
						"price": b.price,
						"ask":   minAsk,
					}).Info("stop loss short")

				b.position = 0
				b.price = 0
			}

		case <-ctx.Done():
			b.logger.Info("closing down")

			return
		}
	}
}
