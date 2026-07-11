package main

import (
	"fmt"
	"container/heap"
)

type pair struct {
	dist float64
	indx int
}

type maxHeap []pair

func (m maxHeap) Len() int {
	return len(m)
}

func (m maxHeap) Less(i, j int) bool  {
	return m[i].dist > m[j].dist
}

func (m maxHeap) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (m *maxHeap) Push(x any) {
	*m = append(*m, x.(pair))
}

func (m *maxHeap) Pop() any {
	old := *m
	n := len(old)
	item := old[n - 1]
	*m = old[:n - 1]
	return item
}

func (m maxHeap) Peek() (pair, bool) {
	if len(m) == 0 {
		return pair{}, false
	}
	return m[0], true
}

func location(stations []int, k int) float64 {
	pq := &maxHeap{}
	heap.Init(pq)
	for i := 1; i < len(stations); i++ {
		dist := float64(stations[i]) - float64(stations[i - 1])
		heap.Push(pq, pair{dist, i - 1})

	}
	arr := make([]int, len(stations) - 1)
	for k > 0 {
		top := heap.Pop(pq).(pair)
		getIndx := top.indx
		arr[getIndx]++
		newDist := (float64(stations[getIndx + 1]) - float64(stations[getIndx])) / (float64(arr[getIndx] + 1.0))
		heap.Push(pq, pair{newDist, getIndx})
		k--
	}
	i, _ := pq.Peek()
	return i.dist
}

func main() {
	stations := []int{11,2,3,4,5,6,7,8,9,10}
	k := 1
	fmt.Println(location(stations, k))
}