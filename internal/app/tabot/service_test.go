package tabot

import "testing"

func Test_parsePair(t *testing.T) {
	type args struct {
		pair string
	}

	tests := []struct {
		name  string
		args  args
		want  string
		want1 string
	}{
		{"BTC/USD", args{"BTC/USD"}, "BTC", "USD"},
		{"ETHBTC", args{"ETHBTC"}, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := parsePair(tt.args.pair)

			if got != tt.want {
				t.Errorf("parsePair() got = %v, want %v", got, tt.want)
			}

			if got1 != tt.want1 {
				t.Errorf("parsePair() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_index(t *testing.T) {
	type args struct {
		symbol string
		syms   []string
	}

	tests := []struct {
		name string
		args args
		want int
	}{
		{"BTC", args{"BTC", []string{"BTC", "ETH", "SOL"}}, 0},
		{"ETH", args{"ETH", []string{"BTC", "ETH", "SOL"}}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := index(tt.args.symbol, tt.args.syms); got != tt.want {
				t.Errorf("index() = %v, want %v", got, tt.want)
			}
		})
	}
}
