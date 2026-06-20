package main

import (
	"container/heap"
	"fmt"
)

type IntHeap []int

func (h IntHeap) Len() int {return len(h)}
func (h IntHeap) Less(i, j int) bool {return h[i] > h[j]}
func (h IntHeap) Swap(i, j int) {h[i], h[j] = h[j], h[i]} 
func (h *IntHeap) Push(x any) {*h = append(*h, x.(int))}
func (h *IntHeap) Pop() any {
	n := len(*h)
	x := (*h)[n - 1]
	(*h) = (*h)[0:n - 1]
	return x
}
func (h *IntHeap) Peek() (int, bool) {
	if len(*h) == 0 {
		return 0, false
	}
	return (*h)[0], true
}

func main() {
	hea := &IntHeap{}
	arr := []int{1, 2, 3, 4, 5, 900, -121}
	heap.Init(hea)
	for _, v := range arr {
		heap.Push(hea, v)
	}
	val, _ := hea.Peek()
	fmt.Println(val)
	fmt.Println(heap.Pop(hea))
	for hea.Len() > 0 {
		fmt.Println(heap.Pop(hea))
	}
}