// Abstraction

package main

type PaymentMethod interface {
	Process(amount float64) bool
}

type CreditCardPayment struct{}

func (CreditCardPayment) Process(amount float64) bool {
	_ = amount
	return true
}

type PayPalPayment struct{}

func (PayPalPayment) Process(amount float64) bool {
	_ = amount
	return true
}

type GoodOrder struct {
	total float64
}

type OrderServiceGood struct {
	paymentMethod PaymentMethod
}

func NewOrderServiceGood(method PaymentMethod) *OrderServiceGood {
	return &OrderServiceGood{paymentMethod: method}
}

func (s *OrderServiceGood) Checkout(order GoodOrder) {
	s.paymentMethod.Process(order.total)
}