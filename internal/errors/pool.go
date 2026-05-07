package errors

import e "github.com/joomcode/errorx"

var (
	// PoolErrors is the errorx namespace for failures originating in the
	// goroutine-pool subsystem (panjf2000/ants). Wrap upstream pool errors
	// with one of the typed members below so callers can branch on category
	// without string-matching.
	PoolErrors = e.NewNamespace("pool")

	// PoolExhausted marks failure to submit work because the pool is full
	// and configured to reject (NonBlocking=true) rather than block.
	PoolExhausted = PoolErrors.NewType("pool_exhausted")

	// PoolReleased marks failure to submit work because the pool has
	// already been released (Application shutdown is in progress or has
	// completed).
	PoolReleased = PoolErrors.NewType("pool_released")

	// SubmitFailed marks any other submit-time failure (typically from the
	// underlying ants library — wrapped here so callers don't have to
	// import ants to branch on it).
	SubmitFailed = PoolErrors.NewType("submit_failed")
)
