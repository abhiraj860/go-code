// SOLID Principles
// OCP - Support future requirements without modifying existing code

package main

type PaymentMethod interface {
	Process(amount float64)
}

type CreditCardPaymentGood struct {}

func (CreditCardPaymentGood) Process(amount float64) {
	_ = amount
}

type PayPalPaymentGood struct {}

func (PayPalPaymentGood) Process(amount float64) {
	_ = amount
}

type CryptoPayment struct {}

func (CryptoPayment) Process(amount float64) {
	_ = amount
}

type PaymentProcessorGood struct{}
 
func (PaymentProcessorGood) Process(method PaymentMethod, amount float64) {
	method.Process(amount)
}