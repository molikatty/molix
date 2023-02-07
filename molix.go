package molix

import "math"

const (
	defaultSize = math.MaxInt32
)

var (
	defaultMolixPool, _ = NewMPool(defaultSize)
)

func Submit(t func()) error {
	return defaultMolixPool.Submit(t)
}

func Close() error {
	return defaultMolixPool.Close()
}

func Running() int {
	return defaultMolixPool.GetRunnings()
}

func Cap() int {
	return defaultMolixPool.GetCap()
}
