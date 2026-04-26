package game

import (
	"math/rand"
	"sync"
	"time"
)

var gameRand = struct {
	mu  sync.Mutex
	rng *rand.Rand
}{
	rng: rand.New(rand.NewSource(time.Now().UnixNano())),
}

func SetRandomSeed(seed int64) {
	gameRand.mu.Lock()
	defer gameRand.mu.Unlock()

	gameRand.rng = rand.New(rand.NewSource(seed))
}

func randomIntn(n int) int {
	gameRand.mu.Lock()
	defer gameRand.mu.Unlock()

	return gameRand.rng.Intn(n)
}

func randomInt63() int64 {
	gameRand.mu.Lock()
	defer gameRand.mu.Unlock()

	return gameRand.rng.Int63()
}

func randomFloat64() float64 {
	gameRand.mu.Lock()
	defer gameRand.mu.Unlock()

	return gameRand.rng.Float64()
}

func randomNormFloat64() float64 {
	gameRand.mu.Lock()
	defer gameRand.mu.Unlock()

	return gameRand.rng.NormFloat64()
}

func randomShuffle(n int, swap func(i int, j int)) {
	gameRand.mu.Lock()
	defer gameRand.mu.Unlock()

	gameRand.rng.Shuffle(n, swap)
}
