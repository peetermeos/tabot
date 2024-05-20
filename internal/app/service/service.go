package service

import (
	"fmt"
	"gonum.org/v1/gonum/mat"
)

type Tick struct {
	Symbol string
	Bid    float64
	BidQty float64
	Ask    float64
	AskQty float64
}

type TriangleBot struct {
}

func (t *TriangleBot) Run() {
	// TODO: Just a placeholder for now
	//   we will be constructing triangle legs via
	//   matrix multiplication of exchange rates
	zero := mat.NewDense(3, 5, nil)
	fmt.Println(zero)
}
