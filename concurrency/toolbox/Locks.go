// Locks (Mutexes)
package toolbox

import "sync"


func Lock() {
	var mu sync.Mutex
	balance := 0
	amount := 20
	mu.Lock()
	defer mu.Lock()
	// Only one goroutine can be here at a time 
	balance += amount
}