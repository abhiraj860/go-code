package toolbox

import "sync"


func ThreadWaitOnCondtion() {
	var mu sync.Mutex
	var cond = sync.NewCond(&mu)
	condition := false;

	cond.L.Lock()
	for !condition {
		cond.Wait() // Release lock and sleep
	}
	// condition is now true
	cond.L.Unlock()
}

