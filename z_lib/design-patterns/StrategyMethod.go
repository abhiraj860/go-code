// Behavioral Patterns
// Strategy: Use when you're replacing if/else logic with interchangeable behaviors.

package main

import "fmt"

type PaymentStrategy interface {
	Pay(amount float64) bool
}

type CreditCardPayment struct {
	cardNumber string
}

func NewCreditCardPayment(cardNumber string) *CreditCardPayment {
	return &CreditCardPayment{cardNumber: cardNumber}
}

func (c *CreditCardPayment) Pay(amount float64) bool {
	fmt.Printf("Paid %.2f with credit card ending %s \n", amount, c.cardNumber)
	return true
}

type PayPalPayment struct {
	email string
}

func NewPayPalPayment(email string) *PayPalPayment {
	return &PayPalPayment{email : email}
}

func (p *PayPalPayment) Pay(amount float64) bool {
	fmt.Printf("Paid %.2f with PayPal account %s", amount, p.email)
	return true
}

type ShoppingCart struct {
	paymentStrategy PaymentStrategy
}

func NewShoppingCart() *ShoppingCart {
	return &ShoppingCart{}
}

func (c *ShoppingCart) SetPaymentStrategy(strategy PaymentStrategy) {
	c.paymentStrategy = strategy
}

func (c *ShoppingCart) Checkout(amount float64) {
	if c.paymentStrategy != nil {
		c.paymentStrategy.Pay(amount)
	}
}


// func main() {
// 	cart := NewShoppingCart()
// 	cart.SetPaymentStrategy(NewCreditCardPayment("1234-5678"))
// 	cart.Checkout(100.00)

// 	cart.SetPaymentStrategy(NewPayPalPayment("user@example.com"))
// 	cart.Checkout(50.00)
// }




















