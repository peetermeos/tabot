package mock

import (
	"context"
	"github.com/peetermeos/tabot/internal/app/service"
)

type Portfolio struct {
	capital float64
	fee     float64
}

func NewPortfolio(capital float64, fee float64) *Portfolio {
	p := Portfolio{
		capital: capital,
		fee:     fee,
	}

	return &p
}

func (p Portfolio) Execute(_ context.Context, _ service.ExecutionInput) error {
	//TODO implement me
	panic("implement me")
}
