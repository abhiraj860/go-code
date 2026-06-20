package main

import (
	"fmt"
	"container/heap"
)

type pair struct {
	num, count int
}

type MaxHeap []pair

func (h MaxHeap) Len() int { return len(h)}
func (h MaxHeap) Less(i, j int) bool {return h[i].count > h[j].count}
func (h MaxHeap) Swap(i, j int) {h[i], h[j] = h[j], h[i]} 
func (h *MaxHeap) Push(x any) {
	*h = append(*h, x.(pair))
}
func (h *MaxHeap) Pop() any {
	x := (*h)[len(*h) - 1]
	*h = (*h)[:len(*h) - 1]
	return x
}

func (h *MaxHeap) Top() any {
	return (*h)[0]
}

func topKFrequent(nums []int, k int) []int {
	freqMap := make(map[int]int)
	for _, num := range nums {
		freqMap[num]++
	}
	maxHeap := &MaxHeap{}
	for num, freq := range freqMap{
		*maxHeap = append(*maxHeap, pair{num, freq})
	}
	heap.Init(maxHeap)
	topK := []int{}
	for range k {
		topK = append(topK, heap.Pop(maxHeap).(pair).num)
	}
	return topK
}

func main() {
	nums := []int{1, 1, 1, 2, 2, 3, 3, 3,3}
	fmt.Println(topKFrequent(nums, 1))	
}