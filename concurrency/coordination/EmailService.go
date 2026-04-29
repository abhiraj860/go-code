// Go's channels make this pattern trivial. The buffered channel acts as the bounded queue. Sends block when full, receives block when empty. Worker goroutines range over the channel, automatically exiting when it's closed.

package coordination

import "fmt"

type EmailTask struct {
	Recipient string
	Template  string
	Data      string
}

type EmailService struct {
	emailQueue     chan EmailTask
	userRepository *userRepository  // ← pointer so Save() works
	emailClient    *emailClient     // ← add instance field
}

func NewEmailService() *EmailService {
	return &EmailService{
		emailQueue:     make(chan EmailTask, 10000),
		userRepository: &userRepository{},  // ← instantiate
		emailClient:    &emailClient{},     // ← instantiate
	}
}

type userRepository struct{}

func (u *userRepository) Save(email, name string) {
	fmt.Printf("User saved: %s %s\n", email, name)
}

type emailClient struct{}

func (e *emailClient) Send(recipient, template, data string) {
	fmt.Printf("Email sent to %s | %s | %s\n", recipient, template, data)
}

// API handler (producer)
func (s *EmailService) Signup(email, name string) {
	// Fast: Save user to database
	s.userRepository.Save(email, name)

	// Fast: Enqueue background work
	s.emailQueue <- EmailTask{email, "welcome", name}

	// Return immediately - user sees instant response
}

// Worker goroutine (consumer)
func (s *EmailService) EmailWorker() {
	for task := range s.emailQueue {
		// Slow: Connect to email server and send
		s.emailClient.Send(task.Recipient, task.Template, task.Data) // ← s.emailClient
	}
}