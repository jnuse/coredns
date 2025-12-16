package https

import (
	"math/rand"
	"sync/atomic"
)

// policy defines a policy we use for selecting upstreams.
type policy interface {
	List(poolLen int) []int
}

// randomPolicy is a policy that implements random upstream selection.
type randomPolicy struct{}

func newRandomPolicy() *randomPolicy {
	return &randomPolicy{}
}

func (*randomPolicy) List(poolLen int) []int {
	if poolLen <= 0 {
		return nil
	}
	return rand.Perm(poolLen)
}

// roundRobinPolicy is a policy that selects hosts based on round robin ordering.
type roundRobinPolicy struct {
	robin uint32
}

func newRoundRobinPolicy() *roundRobinPolicy {
	return &roundRobinPolicy{}
}

func (p *roundRobinPolicy) List(poolLen int) (result []int) {
	if poolLen <= 0 {
		return
	}
	result = make([]int, 0, poolLen)
	// Safely increment and modulo to avoid overflow issues
	i := int(atomic.AddUint32(&p.robin, 1)) % poolLen
	for j := i; j < poolLen; j++ {
		result = append(result, j)
	}
	for j := 0; j < i; j++ {
		result = append(result, j)
	}
	return
}

// sequentialPolicy is a policy that selects hosts based on sequential ordering.
type sequentialPolicy struct{}

func newSequentialPolicy() *sequentialPolicy {
	return &sequentialPolicy{}
}

func (*sequentialPolicy) List(poolLen int) (result []int) {
	if poolLen <= 0 {
		return
	}
	result = make([]int, poolLen)
	for i := 0; i < poolLen; i++ {
		result[i] = i
	}
	return
}
