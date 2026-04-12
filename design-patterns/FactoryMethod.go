// Creational Patterns
// Factory: Use when the callers shouldn't care which concrete class gets created.

package main

import "fmt"

type Notification interface {
	Send(message string)
}

type EmailNotification struct{}

func (EmailNotification) Send(message string) {
	// Email sending logic goes here
}

type SMSNotification struct{}

func (SMSNotification) Send(message string) {
	// SMS sending logic goes here
}	

func CreateNotification(notificationType string) (Notification, error) {
	switch notificationType {
	case "email":
		return EmailNotification{}, nil
	case "sms":
		return SMSNotification{}, nil
	default:
		return nil, fmt.Errorf("unknown type %s", notificationType)
	}
}

func main() {
	notif,_ := CreateNotification("email")
	notif.Send("Hello")
}