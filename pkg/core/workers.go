package core

import "runtime"

func WorkerCount(count int) int {
	return max(min(runtime.NumCPU(), count), 1)
}
