package main

import (
	"physics/isotope"
)

func main() {
	iso := isotope.U235()

	var products isotope.Products
	var neutrons []int

	for i := 0; i < 10000; i++ {
		prods, ns, err := iso.Destabilize()

		if err != nil {
			// do something there...
		}
		products = append(products, prods...)
		neutrons = append(neutrons, ns)
	}

	symbols := products.CountSymbols()
	probs := products.CountProbabilities()
	groups := products.CountIsotopes()

	symbols.SaveJson()
	groups.SaveJson()
	probs.SaveJson()

	symbols.SaveChart()
	probs.SaveChart()
	groups.SaveChart()
}
