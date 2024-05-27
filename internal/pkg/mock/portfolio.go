package mock

import (
	"context"

	"github.com/peetermeos/tabot/internal/app/tabot"
)

type Portfolio struct {
	capital map[string]float64
	fee     float64
	base    string
}

func NewPortfolio(capital float64, base string, fee float64) *Portfolio {
	p := Portfolio{
		capital: map[string]float64{base: capital},
		fee:     fee,
		base:    base,
	}

	return &p
}

func (p *Portfolio) Execute(_ context.Context, input tabot.ExecutionInput) error {
	_, exists := p.capital[input.Base]
	if !exists {
		p.capital[input.Base] = 0
	}

	_, exists = p.capital[input.Symbol]
	if !exists {
		p.capital[input.Symbol] = 0
	}

	if input.Side == "buy" {
		p.capital[input.Base] -= input.Rate * input.Qty
		p.capital[input.Symbol] += input.Qty * (1 - p.fee)
	} else if input.Side == "sell" {
		p.capital[input.Base] += input.Rate * input.Qty * (1 - p.fee)
		p.capital[input.Symbol] -= input.Qty
	}

	return nil
}

func (p *Portfolio) TotalCapital() float64 {
	currentBase, exists := p.capital[p.base]
	if exists {
		return currentBase
	}

	return 0
}
