package main

import (
	"context"
	"os"

	"github.com/peetermeos/tabot/config"
	"github.com/peetermeos/tabot/internal/pkg/kraken"
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

	krakenClient := kraken.NewClient(ctx, tabotLogger, cfg.KrakenKey, cfg.KrakenSecret)

	tabotLogger.Info(krakenClient)
}
