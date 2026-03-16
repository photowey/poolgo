package poolgo

// PoolState describes the lifecycle stage of a goroutine pool.
type PoolState int32

const (
	// PoolStateRunning means the pool accepts new work.
	PoolStateRunning PoolState = iota
	// PoolStateShuttingDown means the pool rejects new work and drains running tasks.
	PoolStateShuttingDown
	// PoolStateStopped means the pool has terminated all workers.
	PoolStateStopped
)

func (state PoolState) String() string {
	switch state {
	case PoolStateRunning:
		return "running"
	case PoolStateShuttingDown:
		return "shutting_down"
	case PoolStateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}
