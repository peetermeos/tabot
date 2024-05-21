package kraken

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestClient_authenticate(t *testing.T) {
	krakenKey, _ := os.LookupEnv("KRAKEN_API_KEY")
	krakenSecret, _ := os.LookupEnv("KRAKEN_API_SECRET")

	if krakenKey == "" || krakenSecret == "" {
		t.Skip("KRAKEN_API_KEY or KRAKEN_API_SECRET not set")
	}

	type fields struct {
		apiKey    string
		apiSecret string
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{"Want token", fields{krakenKey, krakenSecret}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Client{
				logger:    logrus.New(),
				apiKey:    tt.fields.apiKey,
				apiSecret: tt.fields.apiSecret,
			}

			if err := c.authenticate(context.Background()); (err != nil) != tt.wantErr {
				t.Errorf("authenticate() error = %v, wantErr %v", err, tt.wantErr)
			}

			t.Log(c.token)
		})
	}
}
