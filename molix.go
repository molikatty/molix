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

func Stop() {
	defaultMolixPool.Stop()
}

func Running() int {
	return defaultMolixPool.Running()
}

func Cap() int {
	return defaultMolixPool.Cap()
}

func Free() int {
	return defaultMolixPool.Free()
}
