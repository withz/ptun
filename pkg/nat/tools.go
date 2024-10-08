package nat

import (
	"math/rand"
	"time"
)

func NoRepeatRandInts(start, end int, count int) (result []int) {
	if start > end {
		start, end = end, start
	}
	if count > end-start {
		count = end - start
	}
	random := rand.New(rand.NewSource(time.Now().Unix()))

	result = make([]int, 0)
	memory := make(map[int]bool)

	for len(result) < count {
		x := random.Int()%(end-start) + start
		if exist, ok := memory[x]; ok && exist {
			continue
		}
		memory[x] = true
		result = append(result, x)
	}
	return result
}

func RandomOne[T any](container []T) T {
	idx := rand.Int() % len(container)
	return container[idx]
}
