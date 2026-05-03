package main

import (
	"fmt"
	"math/rand"
	"time"
)

type Locker struct {
	compartments       []*Compartment
	accessTokenMapping map[string]*AccessToken
}

func NewLocker(compartments []*Compartment) *Locker {
	return &Locker{
		compartments:       compartments,
		accessTokenMapping: make(map[string]*AccessToken),
	}
}

func (l *Locker) DepositPackage(size Size) (string, error) {
	compartment := l.getAvailableCompartment(size)
	if compartment == nil {
		return "", fmt.Errorf("no available compartment of size %s", size)
	}

	compartment.Open()
	compartment.MarkOccupied()
	accessToken := l.generateAccessToken(compartment)
	l.accessTokenMapping[accessToken.GetCode()] = accessToken

	return accessToken.GetCode(), nil
}

func (l *Locker) Pickup(tokenCode string) error {
	if tokenCode == "" {
		return fmt.Errorf("invalid access token code")
	}

	accessToken, exists := l.accessTokenMapping[tokenCode]
	if !exists {
		return fmt.Errorf("invalid access token code")
	}

	if accessToken.IsExpired() {
		return fmt.Errorf("access token has expired")
	}

	compartment := accessToken.GetCompartment()
	compartment.Open()
	l.clearDeposit(accessToken)
	return nil
}

func (l *Locker) OpenExpiredCompartments() {
	for _, accessToken := range l.accessTokenMapping {
		if accessToken.IsExpired() {
			compartment := accessToken.GetCompartment()
			compartment.Open()
		}
	}
}

func (l *Locker) getAvailableCompartment(size Size) *Compartment {
	for _, c := range l.compartments {
		if c.GetSize() == size && !c.IsOccupied() {
			return c
		}
	}
	return nil
}

func (l *Locker) generateAccessToken(compartment *Compartment) *AccessToken {
	code := fmt.Sprintf("%06d", rand.Intn(1000000))
	expiration := time.Now().Add(7 * 24 * time.Hour)
	return NewAccessToken(code, expiration, compartment)
}

func (l *Locker) clearDeposit(accessToken *AccessToken) {
	compartment := accessToken.GetCompartment()
	compartment.MarkFree()
	delete(l.accessTokenMapping, accessToken.GetCode())
}
