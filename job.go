package worker

// Represents user request, function which should be executed in some worker.
type Job func()

type jobChan chan Job
