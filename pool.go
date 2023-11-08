package worker

import "sync/atomic"

type jobPool struct {
	jobQueue           chan Job
	workerPool         chan jobChan
	stop               chan struct{}
	workerCount        atomic.Int32
	workerInQueueCount atomic.Int32
	jobInQueueCount    atomic.Int32
	minWorkerCount     int32
	maxWorkerCount     int32
	onWorkerPanic      func(e any)
}

// Will make pool of gorouting workers.
// numWorkers - how many workers will be created for this pool
// queueLen - how many jobs can we accept until we block
//
// Returned object contains JobQueue reference, which you can use to send job to pool.
func NewJobPool(minWorkers, maxWorkers int, jobQueueLen int, opt ...Option) *jobPool {
	jobQueue := make(jobChan, jobQueueLen)
	workerPool := make(chan jobChan, minWorkers)

	pool := &jobPool{
		jobQueue:   jobQueue,
		workerPool: workerPool,
	}
	pool.minWorkerCount = int32(minWorkers)
	pool.maxWorkerCount = int32(maxWorkers)
	for _, o := range opt {
		o(pool)
	}

	go pool.dispatch()

	return pool
}

func (p *jobPool) Run(fn func()) {
	p.jobInQueueCount.Add(1)
	p.jobQueue <- fn

	p.tryAddNewWorker()
}

func (p *jobPool) tryAddNewWorker() {
	var worker jobChan
	workerCount := p.workerCount.Load()
	if workerCount < p.minWorkerCount {
		worker = make(jobChan)
	} else if workerCount < p.maxWorkerCount {
		jobInQueue := p.jobInQueueCount.Load()
		workerInQueue := p.workerInQueueCount.Load()
		if jobInQueue >= workerInQueue && (jobInQueue-workerInQueue)%3 == 2 {
			worker = make(jobChan)
		}
	}
	if worker != nil {
		p.workerCount.Add(1)
		go p.runWorker(worker)
	}
}

// Will release resources used by pool
func (p *jobPool) Stop() {
	p.stop <- struct{}{}
	<-p.stop
}

func (p *jobPool) dispatch() {
	var worker jobChan
DispatchLoop:
	for {
		select {
		case job := <-p.jobQueue:
			p.jobInQueueCount.Add(-1)
			worker = <-p.workerPool
			p.workerInQueueCount.Add(-1)
			worker <- job
		case <-p.stop:
			break DispatchLoop
		}
	}
	// 清空任务队列
	for i := 0; i < 1000; i++ {
		select {
		case job := <-p.jobQueue:
			worker := <-p.workerPool
			worker <- job
		default:
		}
	}

	// 退出worker
	count := p.workerCount.Load()
	for i := int32(0); i < count; i++ {
		worker := <-p.workerPool
		close(worker)
	}

	p.stop <- struct{}{}
}

func (p *jobPool) runWorker(worker jobChan) {
	defer func() {
		// worker 数量少1
		p.workerCount.Add(-1)

		e := recover()
		if e != nil && p.onWorkerPanic != nil {
			// recover from panic
			p.onWorkerPanic(e)
		}
	}()
	var job Job
	var ok bool
	for {
		// worker free, add it to pool
		p.workerInQueueCount.Add(1)
		p.workerPool <- worker

		job, ok = <-worker
		if !ok {
			return
		}
		p.jobInQueueCount.Add(-1)
		job()
	}
}
