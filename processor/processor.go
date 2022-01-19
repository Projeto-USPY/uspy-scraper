package processor

import (
	"context"
	"errors"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
)

type (
	Processed interface{}

	Task struct {
		Name string

		Process func() (Processed, error)
		After   func(results Processed) error
		DelayFn func(attempt int) time.Duration

		currentAttempt int
		lastTried      time.Time

		err error
	}

	Processor struct {
		Name   string
		jobs   chan *Task
		failed chan *Task

		results chan Processed

		Tasks       []*Task
		NumWorkers  int
		MaxAttempts int
		Timeout     int
	}
)

func NewTask(
	ID string,
	delayFn func(int) time.Duration,
	processFn func() (Processed, error),
	afterFn func(Processed) error,
) *Task {
	return &Task{
		Name:           ID,
		Process:        processFn,
		After:          afterFn,
		DelayFn:        delayFn,
		currentAttempt: 0,
		lastTried:      time.Now(),
		err:            nil,
	}
}

func NewProcessor(name string, tasks []*Task, fixedAttempts bool) *Processor {
	numWorkers := Config.Processor.NumWorkers
	if Config.Processor.FractionalNumWorkers > 0.0 && Config.Processor.FractionalNumWorkers <= 1.0 {
		numWorkers = int(math.Ceil(float64(len(tasks)) * Config.Processor.FractionalNumWorkers))
	}

	maxAttempts := -1
	if fixedAttempts {
		maxAttempts = Config.Processor.MaxAttempts
	}

	return &Processor{
		Name:        name,
		Tasks:       tasks,
		NumWorkers:  numWorkers,
		MaxAttempts: maxAttempts,
		Timeout:     Config.Processor.Timeout,

		jobs:    make(chan *Task, len(tasks)),
		results: make(chan Processed, len(tasks)),
		failed:  make(chan *Task, len(tasks)),
	}
}

func (proc *Processor) Run() []Processed {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(proc.Timeout*int(time.Second)),
	)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("unexpected panic in run processor: %s", r)
			cancel()
		}
	}()

	log.Debugf("launching %s: %v workers for %v tasks\n", proc.Name, proc.NumWorkers, len(proc.Tasks))
	init := time.Now()

	defer close(proc.jobs)
	defer close(proc.results)
	defer close(proc.failed)

	// prepare workers
	for i := 0; i < proc.NumWorkers; i++ {
		go func(ctx context.Context) {
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-proc.jobs:
					// fail job if exceeded retries
					if proc.MaxAttempts != -1 && job.currentAttempt > proc.MaxAttempts {
						proc.failed <- job
						continue
					}

					var result Processed
					failed := false

					var failedErr error

					// only run job if sufficient time has passed
					delay := job.DelayFn(job.currentAttempt)
					if time.Since(job.lastTried) >= delay {
						var processingErr error
						result, processingErr = job.Process()
						if processingErr != nil {
							failed = true // processing failure
							log.Debugf("[processing-failure] job: %s, reason: %s\n", job.Name, processingErr)
							failedErr = processingErr
						} else if job.After != nil {
							if err := job.After(result); err != nil {
								failed = true // after-processing failure
								log.Debugf("[after-processing-failure] job: %s, reason: %s\n", job.Name, err)

								failedErr = err
							}
						}

					} else {
						failed = true // job denied by now
					}

					if failed {
						job.currentAttempt++
						job.lastTried = time.Now()
						job.err = failedErr

						proc.jobs <- job
					} else {
						proc.results <- result
					}
				}

			}
		}(ctx)
	}

	// send jobs
	log.Debugln("sending jobs")
	go func() {
		for _, task := range proc.Tasks {
			proc.jobs <- task
		}
	}()

	// receive job results
	results := make([]Processed, 0)
	failures := make([]*Task, 0)

	// wait for all jobs to fail, succeed or timeout
	for {
		select {
		case result := <-proc.results:
			results = append(results, result)
		case failure := <-proc.failed:
			failures = append(failures, failure)
		}

		if ctx.Err() != nil || len(results)+len(failures) == len(proc.Tasks) {
			cancel()
			break
		}
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		log.Infof("%s: work is done, stopping...", proc.Name)
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		log.Infof("%s: timeout exceeded, stopping...", proc.Name)
	}

	log.Infof("time elapsed: %v\n", time.Since(init))

	if len(proc.failed) > 0 {
		log.Warnln(len(proc.failed), "jobs failed")

		for _, failedJob := range failures {
			if failedJob.err != nil {
				log.Errorf("[failed] job: %s, reason: %s\n", failedJob.Name, failedJob.err.Error())
			}
		}
	}

	return results
}
