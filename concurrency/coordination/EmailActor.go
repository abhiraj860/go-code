// Go's version uses a handler function instead of inheritance. The EmailActor wraps the generic actor with email-specific behavior. This composition pattern is idiomatic Go and avoids the need for abstract base classes.

package coordination

import "fmt"

type EmailRequest struct {
    To      string
    Subject string
    Body    string
}

type EmailClient struct {
	
}

func NewEmailClient() *EmailClient {
	return &EmailClient{}
}

func (e *EmailClient) Send(request string, subject string, body string) {
	fmt.Println("Email data send")
}

type EmailActor struct {
    *Actor[EmailRequest]
    client *EmailClient
}

func NewEmailActor() *EmailActor {
    ea := &EmailActor{client: NewEmailClient()}
    ea.Actor = NewActor(ea.handleEmail)
    return ea
}

func (ea *EmailActor) handleEmail(req EmailRequest) {
    ea.client.Send(req.To, req.Subject, req.Body)
}

type User struct {
	Email string
}
type UserRepository struct {}

func (s *UserRepository) Save(u User) User {
	return u
}

type SignupRequest struct{
	Email string
}

// Usage: no shared state, no locks needed
type SignupHandler struct {
    emailActor     *EmailActor
    userRepository *UserRepository
}

func (h *SignupHandler) HandleSignup(req SignupRequest) {
    user := h.userRepository.Save(User{Email: req.Email})

    // Send message to actor - returns immediately
    h.emailActor.Send(EmailRequest{
        To:      user.Email,
        Subject: "Welcome!",
        Body:    "Thanks for signing up...",
    })
}

