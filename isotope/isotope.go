package isotope

import (
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/mroth/weightedrand"
	"github.com/wcharczuk/go-chart/v2"
)

// Isotope is a variant of a chemical element.
type Isotope struct {
	// Usually described as "X" in chemistry.
	Symbol string `json:"symbol"`

	// Atomic number is only number of protons. Described as "Z"
	Number int `json:"atomic_number"`

	// Mass is number of protons + neutrons. Described as "A".
	Mass int `json:"mass_number"`
}

// Fragment represents isotope without a symbol.
func Fragment(number, mass int) *Isotope {
	return &Isotope{Number: number, Mass: mass}
}

// Fissiles is slice of fissile isotopes which have very heavy nucleus - so it is "fissionable".
func Fissiles() []*Isotope {
	return []*Isotope{U233(), U235(), P239()}
}

// U235 is Uranium-235 isotope.
func U235() *Isotope {
	return &Isotope{
		Symbol: "U",
		Number: 92,
		Mass:   235,
	}
}

// U235 is Uranium-233 isotope.
func U233() *Isotope {
	return &Isotope{
		Symbol: "U",
		Number: 92,
		Mass:   233,
	}
}

// P239 is Plutonium-239 isotope.
func P239() *Isotope {
	return &Isotope{
		Symbol: "P",
		Number: 94,
		Mass:   239,
	}
}

// Returns random fissionable isotope from isotopes list.
func Random() *Isotope {
	isos := Fissiles()
	rand.Seed(time.Now().UnixNano())
	iso := isos[rand.Intn(len(isos))]
	return iso
}

type Products []*Isotope

// Destabilize destabilizes nucleus of an isotope after neutron absorption.
// It is caused by inducing neutron to the nucleus of an isotope.
// Returns products and neutrons released during fission operation.
func (iso Isotope) Destabilize() (Products, int, error) {
	// increase amu of isotope by one
	iso.induceNeutron()

	// Randomize mass of first fragment based on neutrons released
	neutrons := randomNeutron()
	amu := rand.Intn((iso.Mass-neutrons)-iso.Mass/2) + iso.Mass/2

	// Heavier and lighter fission fragments
	heavier := Fragment((iso.Number*((amu*100)/iso.Mass))/100, amu)
	lighter := Fragment(iso.Number-heavier.Number, iso.Mass-neutrons-amu)

	// Search each fragment isotope equivalent in isotopes
	isos, err := Isotopes()
	if err != nil {
		return nil, 0, err
	}
	for _, iso := range isos {
		if iso.Mass == heavier.Mass && iso.Number == heavier.Number {
			heavier.Symbol = iso.Symbol
		}
		if iso.Mass == lighter.Mass && iso.Number == lighter.Number {
			lighter.Symbol = iso.Symbol
		}
	}
	var prods Products
	// if heavier and lighter fragment has an equivalent, add it to products slice
	if heavier.Symbol != "" && lighter.Symbol != "" {
		prods = append(prods, heavier, lighter)
		return prods, neutrons, nil
	}
	return nil, 0, fmt.Errorf("first or second fragment of a fission reaction does not have equivalent as an isotope")
}

// Name is symbol of an isotope + it's atomic mass number
func (iso *Isotope) Name() string {
	return fmt.Sprintf("%s-%d", iso.Symbol, iso.Mass)
}

// Isotopes returns slice of parsed isotopes from isotopes.json file.
// Parsing occurs only once.
func Isotopes() ([]*Isotope, error) {
	once.Do(func() {
		data, err := file.ReadFile("isotopes.json")
		if err != nil {
			return
		}
		var isos []*Isotope
		json.Unmarshal(data, &isos)
		instance = isos
	})
	return instance, nil
}

// CountSymbols returns map of how many times each chemical element occured.
func (prods Products) CountSymbols() symbols {
	sc := make(symbols)
	for _, prod := range prods {
		sc[prod.Symbol]++
	}
	return sc
}

// CountIsotopes returns map of element symbols and isotopes of this element, and how many times that isotope appears in products
func (prods Products) CountIsotopes() groups {
	ic := make(groups)

	// for each element symbol create map[string]int
	for _, prod := range prods {
		ic[prod.Symbol] = make(map[string]int)
	}
	// populate each element symbol map[string]int with element name and number of occurences
	for _, prod := range prods {
		ic[prod.Symbol][prod.Name()]++
	}

	return ic
}

// CountProbabilities creates a map of element symbol key and avg occurence in percent value
func (prods Products) CountProbabilities() probabilities {
	// symbol : count map
	sc := prods.CountSymbols()

	// sum of every element occurence
	sum := 0
	for _, c := range sc {
		sum += c
	}
	// elements avg map
	probs := make(probabilities)
	for s, c := range sc {
		v := (float64(c) / float64(sum)) * 100 // calculate avg
		probs[s] = v
	}
	return probs
}

// Saves to .json file
func (sc symbols) SaveJson() error {
	data, err := json.MarshalIndent(sc, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile("symbols-count.json", data, 0777)
}

// Saves to .json file
func (ic groups) SaveJson() error {
	data, err := json.MarshalIndent(ic, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile("isotopes-count.json", data, 0777)
}

// Saves to .json file
func (probs probabilities) SaveJson() error {
	data, err := json.MarshalIndent(probs, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile("probs.json", data, 0777)
}

// Saves bar chart of elements to png file
func (sc symbols) SaveChart() {
	var values []chart.Value
	for s, c := range sc {
		values = append(values, chart.Value{Label: s, Value: float64(c)})
	}

	graph := chart.BarChart{
		Title: "Fission products",
		Background: chart.Style{
			Padding: chart.Box{
				Top:   50,
				Right: -15,
			},
		},
		Canvas: chart.Style{
			FontSize: 1,
		},
		YAxis: chart.YAxis{
			Range: &chart.ContinuousRange{
				Min: 0.0,
				Max: 1000,
			},
		},
		Width:    2560,
		Height:   1080,
		BarWidth: 10,
		Bars:     values,
	}
	f, _ := os.Create("products.png")
	defer f.Close()
	graph.Render(chart.PNG, f)
}

// Saves each element symbol map to png file
func (ic groups) SaveChart() {
	for symbol, isotope := range ic {
		var values []chart.Value

		for name, number := range isotope {
			values = append(values, chart.Value{Label: name, Value: float64(number)})

			graph := chart.BarChart{
				Title: symbol,
				Background: chart.Style{
					Padding: chart.Box{
						Top: 50,
					},
				},
				YAxis: chart.YAxis{
					Range: &chart.ContinuousRange{
						Min: 0.0,
						Max: 15000,
					},
				},
				Width:  720,
				Height: 512,
				Bars:   values,
			}

			f, _ := os.Create(fmt.Sprintf("charts/%s.png", symbol))
			defer f.Close()
			graph.Render(chart.PNG, f)
		}
	}
}

// Saves to png file
func (probs probabilities) SaveChart() {
	var values []chart.Value
	for k, v := range probs {
		label := fmt.Sprintf("%s (%.3f)", k, v) + "%"
		values = append(values, chart.Value{Label: label, Value: v})
	}

	pie := chart.DonutChart{
		Title:  "Probability of occurence",
		Width:  3200,
		Height: 1800,
		Values: values,
		Background: chart.Style{
			FontSize:        0.1,
			TextLineSpacing: 1,
		},
		Canvas: chart.Style{
			FontSize:        0.1,
			TextLineSpacing: 1,
		},
	}
	f, _ := os.Create("probs.png")
	defer f.Close()
	pie.Render(chart.PNG, f)
}

type (
	groups        map[string]map[string]int
	symbols       map[string]int
	probabilities map[string]float64
)

func (iso *Isotope) induceNeutron() {
	iso.Mass += 1
}

//go:embed isotopes.json
var file embed.FS

var (
	instance []*Isotope // singleton
	once     sync.Once
)

func randomNeutron() int {
	rand.Seed(time.Now().UnixNano())
	chooser, _ := weightedrand.NewChooser(
		weightedrand.NewChoice(3, 10), // 3 neutrons - 0.1
		weightedrand.NewChoice(2, 30), // 2 neutrons - 0.3
		weightedrand.NewChoice(1, 60), // 1 neutron - 0.6
	)
	n, _ := chooser.Pick().(int)
	return n
}
