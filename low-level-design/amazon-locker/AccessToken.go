package main

import "time"

type AccessToken struct {
	code       string
	expiration time.Time
	compartment *Compartment
}

func NewAccessToken(code string, expiration time.Time, compartment *Compartment) *AccessToken {
	return &AccessToken{
		code:        code,
		expiration:  expiration,
		compartment: compartment,
	}
}

func (a *AccessToken) IsExpired() bool {
	return !time.Now().Before(a.expiration)
}

func (a *AccessToken) GetCompartment() *Compartment {
	return a.compartment
}

func (a *AccessToken) GetCode() string {
	return a.code
}
