// Package pool wraps github.com/panjf2000/ants/v2 with project conventions:
// a Config that hydrates from environment variables under APP_POOL_*, an
// errorx-categorised Submit error surface, and a context-aware
// SubmitWithContext that lets callers cancel queued work when their parent
// context is cancelled.
//
// The pool is owned by the Application lifecycle (configure / initialize /
// shutdown live in internal/application/application_pool.go); subcommands
// reach for the shared instance via app.Pool() rather than constructing
// their own. Sharing the pool means a single --workers / APP_POOL_SIZE
// knob bounds concurrency across every subsystem in the daemon.
package pool
