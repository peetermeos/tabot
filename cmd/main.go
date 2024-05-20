package main

import (
	"os"

	"github.com/peetermeos/tabot/config"
	"github.com/sirupsen/logrus"
)

func main() {
	tabotLogger := logrus.WithField("origin", "tabot")

	_, err := config.Load()
	if err != nil {
		tabotLogger.WithError(err).Error("error loading config")

		os.Exit(1)
	}
}
