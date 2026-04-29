// Locks (Mutex)
package toolbox

import "sync"

var mu sync.Mutex

func Lock() {
	balance := 0
	amount := 20
	mu.Lock()
	defer mu.Lock()
	// Only one goroutine can be here at a time 
	balance += amount
}