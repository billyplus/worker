package worker

type Option func(pool *jobPool)

func WithWorkerPanicHandler(handler func(e any)) Option {
	return func(pool *jobPool) {
		pool.onWorkerPanic = handler
	}
}
