package main

import (
	"context"
	"github.com/peetermeos/tabot/internal/app/prebot"
	"github.com/peetermeos/tabot/internal/pkg/kraken"
	"os"

	_ "github.com/breml/rootcerts"
	"github.com/peetermeos/tabot/config"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	logger := logrus.WithField("origin", "prebot")

	cfg, err := config.Load()
	if err != nil {
		logger.WithError(err).Error("error loading config")

		os.Exit(1)
	}

	logLevel, _ := logrus.ParseLevel(cfg.LogLevel)
	logrus.SetLevel(logLevel)

	krakenClient := kraken.NewClient(ctx, logger, cfg.KrakenKey, cfg.KrakenSecret)

	botInput := prebot.BotInput{
		Logger:     logger,
		MarketData: krakenClient,
		Symbol:     cfg.Symbol,
	}

	app := prebot.NewPressureBot(botInput)

	app.Run(ctx)
}
