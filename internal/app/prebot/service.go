package prebot

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"math"
	"sort"
)

const bookLength = 10

const fee = 0.0025

type MarketDataProvider interface {
	StreamBook(ctx context.Context) <-chan Book
	SubscribeBook(symbol string) error
	UnsubscribeBook(symbol string) error
}

type ExecutionProvider interface {
	Execute(ctx context.Context, input ExecutionInput) error
	TotalCapital() float64
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

type ExecutionInput struct {
	Symbol string
	Base   string
	Side   string
	Rate   float64
	Qty    float64
}

type PressureBot struct {
	logger logrus.FieldLogger
	data   MarketDataProvider
	symbol string

	askBook []bookItem
	bidBook []bookItem

	position float64
	price    float64
}

type bookItem struct {
	price  float64
	volume float64
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
		askBook: make([]bookItem, 0),
		bidBook: make([]bookItem, 0),
	}
}

func (b *PressureBot) Run(ctx context.Context) {
	err := b.data.SubscribeBook(b.symbol)
	if err != nil {
		b.logger.WithError(err).Error("error subscribing to book")

		return
	}

	stream := b.data.StreamBook(ctx)

	pnl := 0.0

	for {
		select {
		case book := <-stream:
			b.logger.WithField("book", fmt.Sprintf("%+v", book)).Debug("received book")

			if !book.IsUpdate {
				b.askBook = make([]bookItem, 0)
				b.bidBook = make([]bookItem, 0)
			}

			for _, bid := range book.Bids {
				if bid.Volume == 0 {
					b.bidBook = removeLevel(b.bidBook, bid.Price)

					continue
				}

				b.bidBook = setVolume(b.bidBook, bid.Price, bid.Volume)
			}

			for _, ask := range book.Asks {
				if ask.Volume == 0 {
					b.askBook = removeLevel(b.askBook, ask.Price)

					continue
				}

				b.askBook = setVolume(b.askBook, ask.Price, ask.Volume)
			}

			if len(b.askBook) > bookLength {
				// Clean up book, retain lowest 10 elements
				b.askBook = b.askBook[:bookLength]
			}

			if len(b.bidBook) > bookLength {
				// Clean up book, retain top 10 elements
				b.bidBook = b.bidBook[(len(b.bidBook) - bookLength):]
			}

			totalBid := 0.0
			maxBid := 0.0

			for _, item := range b.bidBook {
				totalBid += item.volume
				maxBid = math.Max(maxBid, item.price)
			}

			totalAsk := 0.0
			minAsk := 9999999999.0

			for _, item := range b.askBook {
				totalAsk += item.volume
				minAsk = math.Min(minAsk, item.price)
			}

			const threshold = 30

			const tradeSize = 1000.0

			if totalBid-totalAsk > threshold {
				if b.position != 0 {
					continue
				}

				pnl -= tradeSize * fee

				b.logger.WithFields(logrus.Fields{
					"total_bid": fmt.Sprintf("%.4f", totalBid),
					"total_ask": fmt.Sprintf("%.4f", totalAsk),
					"bid":       maxBid,
					"ask":       minAsk,
					"delta":     fmt.Sprintf("%.4f", totalBid-totalAsk),
					"pnl":       pnl,
				}).Info("enter long")

				b.position = tradeSize / minAsk
				b.price = minAsk
			}

			if totalAsk-totalBid > threshold {
				if b.position != 0 {
					continue
				}

				pnl -= tradeSize * fee

				b.logger.WithFields(logrus.Fields{
					"total_bid": fmt.Sprintf("%.4f", totalBid),
					"total_ask": fmt.Sprintf("%.4f", totalAsk),
					"bid":       maxBid,
					"ask":       minAsk,
					"delta":     fmt.Sprintf("%.4f", totalBid-totalAsk),
					"pnl":       pnl,
				}).Info("enter short")

				b.position = -tradeSize / maxBid
				b.price = maxBid
			}

			var target = 0.008 * b.price

			if maxBid >= b.price+target && b.position > 0 {
				pnl += b.position * (maxBid - b.price)
				pnl -= tradeSize * fee

				b.logger.
					WithFields(logrus.Fields{
						"price": b.price,
						"bid":   maxBid,
						"pnl":   pnl,
					}).Info("exit long")

				b.position = 0
				b.price = 0
			}

			if minAsk <= b.price-target && b.position < 0 {
				pnl += b.position * (b.price - minAsk)
				pnl -= tradeSize * fee

				b.logger.
					WithFields(logrus.Fields{
						"price": b.price,
						"ask":   minAsk,
						"pnl":   pnl,
					}).Info("exit short")

				b.position = 0
				b.price = 0
			}

			if minAsk < b.price && b.position > 0 {
				pnl += b.position * (maxBid - b.price)
				pnl -= tradeSize * fee

				b.logger.
					WithFields(logrus.Fields{
						"price": b.price,
						"bid":   maxBid,
						"pnl":   pnl,
					}).Info("stop loss long")

				b.position = 0
				b.price = 0
			}

			if maxBid > b.price && b.position < 0 {
				pnl += b.position * (b.price - minAsk)
				pnl -= tradeSize * fee

				b.logger.
					WithFields(logrus.Fields{
						"price": b.price,
						"ask":   minAsk,
						"pnl":   pnl,
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

func setVolume(book []bookItem, price, volume float64) []bookItem {
	for i, item := range book {
		if item.price == price {
			book[i].volume = volume

			return book
		}
	}

	book = append(book, bookItem{
		price:  price,
		volume: volume,
	})

	// sort the book by price
	sort.Slice(book, func(i, j int) bool {
		return book[i].price < book[j].price
	})

	return book
}

func removeLevel(book []bookItem, price float64) []bookItem {
	for i, item := range book {
		if item.price == price {
			book = append(book[:i], book[i+1:]...)

			break
		}
	}

	return book
}
