package main

import (
	"context"
	"os"
	"strings"

	_ "github.com/breml/rootcerts"
	"github.com/peetermeos/tabot/config"
	"github.com/peetermeos/tabot/internal/app/service"
	"github.com/peetermeos/tabot/internal/pkg/kraken"
	"github.com/peetermeos/tabot/internal/pkg/mock"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	tabotLogger := logrus.WithField("origin", "tabot")

	cfg, err := config.Load()
	if err != nil {
		tabotLogger.WithError(err).Error("error loading config")

		os.Exit(1)
	}

	logLevel, _ := logrus.ParseLevel(cfg.LogLevel)
	logrus.SetLevel(logLevel)

	krakenClient := kraken.NewClient(ctx, tabotLogger, cfg.KrakenKey, cfg.KrakenSecret)
	mockPortfolio := mock.NewPortfolio(10000, "USD", 0.0025)

	botInput := service.BotInput{
		Logger:     tabotLogger,
		MarketData: krakenClient,
		Execution:  mockPortfolio,
		Symbols:    strings.Split(cfg.Symbols, ","),
	}

	tabot := service.NewTriangleBot(botInput)

	tabot.Run(ctx)
}
