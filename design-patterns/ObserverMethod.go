// Behavioral Patterns
// Observer: Use when multiple components need to react to a single event.
package main

type Observer interface {
	Update(symbol string, price float64)
}

type Subject interface {
	Attach(observer Observer)
	Detach(observer Observer)
	NotifyObservers()
}

type Stock struct {
	observers []Observer
	symbol string
	price float64
}

func NewStock(symbol string) *Stock {
	return &Stock{
		observers : []Observer{},
		symbol : symbol,
	}
}

func (s *Stock) Attach(observer Observer) {
	s.observers = append(s.observers, observer)
}


func (s *Stock) Detach(observer Observer) {
	for i, o := range s.observers {
		if o == observer {
			s.observers = append(s.observers[:i], s.observers[i+ 1:]...)
			break
		}
	}
}

func (s *Stock) SetPrice(price float64) {
	s.price = price
	s.NotifyObservers()
}

func (s *Stock) NotifyObservers() {
	for _,observer := range s.observers {
		observer.Update(s.symbol, s.price)
	}
}


type PriceDisplay struct{}

func (PriceDisplay) Update(symbol string, price float64) {
	// fmt.Printf("Display updated: %s = $%.2f\n", symbol, price)
	_ = price
	_ = symbol	
}

type PriceAlert struct{
	threshold float64
}

func NewPriceAlert(threshold float64) *PriceAlert {
	return &PriceAlert{threshold: threshold}
}

func (p *PriceAlert) Update(symbol string, price float64) {
	if price > p.threshold {
		// "fmt.Printf("Alert! %s exceeded $%.2f\n", symbol, p.threshold)
	}
	_ = symbol
}


// usage
// stock := NewStock("AAPL")
// display := PriceDisplay{}
// alert := NewPriceAlert(150.00)
// stock.Attach(display)
// stock.Attach(alert)
// stock.SetPrice(145.00)
// stock.SetPrice(155.00)





