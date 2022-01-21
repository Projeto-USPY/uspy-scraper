package processor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	log "github.com/sirupsen/logrus"
)

type (
	Processed interface{}

	Task struct {
		Name string

		Process func(ctx context.Context) (Processed, error)
		After   func(ctx context.Context, results Processed) error
		DelayFn func(attempt int) time.Duration

		currentAttempt int
		lastTried      time.Time
		err            error
	}

	Processor struct {
		Name  string
		Tasks []*Task

		numWorkers  int
		maxAttempts int
		timeout     int

		ctx     context.Context
		jobs    chan *Task
		failed  chan *Task
		results chan Processed
	}
)

func NewTask(
	ID string,
	delayFn func(int) time.Duration,
	processFn func(context.Context) (Processed, error),
	afterFn func(context.Context, Processed) error,
) *Task {
	return &Task{
		Name:    ID,
		Process: processFn,
		After:   afterFn,
		DelayFn: delayFn,

		currentAttempt: 0,
		lastTried:      time.Now(),
		err:            nil,
	}
}

func NewProcessor(ctx context.Context, name string, tasks []*Task, fixedAttempts bool) *Processor {
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
		numWorkers:  numWorkers,
		maxAttempts: maxAttempts,
		timeout:     Config.Processor.Timeout,

		ctx:     ctx,
		jobs:    make(chan *Task, len(tasks)),
		results: make(chan Processed, len(tasks)),
		failed:  make(chan *Task, len(tasks)),
	}
}

func (proc *Processor) Run() []Processed {
	ctx, cancel := context.WithTimeout(
		proc.ctx,
		time.Duration(proc.timeout*int(time.Second)),
	)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("unexpected panic in run processor: %s", r)
			cancel()
		}
	}()

	log.Debugf("launching %s: %v workers for %v tasks\n", proc.Name, proc.numWorkers, len(proc.Tasks))
	init := time.Now()

	defer close(proc.jobs)
	defer close(proc.results)
	defer close(proc.failed)

	// prepare workers
	for i := 0; i < proc.numWorkers; i++ {
		go func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("unexpected panic in job goroutine: %s", r)
					cancel()
				}
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-proc.jobs:
					log.WithFields(log.Fields{
						"agent": proc.Name,
						"job":   job.Name,
					}).Debug("starting job")

					// fail job if exceeded retries
					if proc.maxAttempts != -1 && job.currentAttempt > proc.maxAttempts {
						log.WithFields(log.Fields{
							"agent": proc.Name,
							"job":   job.Name,
						}).Warn("job exceeded max retries, failing...")
						proc.failed <- job
						continue
					}

					var result Processed

					failed := false
					denied := false

					var failedErr error

					// only run job if sufficient time has passed
					delay := job.DelayFn(job.currentAttempt)
					if time.Since(job.lastTried) >= delay {
						var processingErr error
						result, processingErr = job.Process(ctx)
						if processingErr != nil {
							failed = true // processing failure
							failedErr = fmt.Errorf("%s: %s", "processing error", processingErr.Error())
						} else if job.After != nil {
							if err := job.After(ctx, result); err != nil {
								failed = true // after-processing failure
								failedErr = fmt.Errorf("%s: %s", "after-processing error", processingErr.Error())
							}
						}
					} else {
						denied = true // job denied by now
					}

					if denied {
						log.WithFields(log.Fields{
							"agent": proc.Name,
							"job":   job.Name,
						}).Warn("job denied by now...")
						proc.jobs <- job
					} else if failed {
						log.WithFields(log.Fields{
							"agent": proc.Name,
							"job":   job.Name,
							"error": failedErr,
						}).Error("job failed...")

						job.currentAttempt++
						job.lastTried = time.Now()
						job.err = failedErr

						proc.jobs <- job
					} else {
						log.WithFields(log.Fields{
							"agent": proc.Name,
							"job":   job.Name,
						}).Debug("job successful")
						proc.results <- result
					}
				}

			}
		}(ctx)
	}

	// send jobs
	log.Debug("sending jobs")
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
		log.WithField("agent", proc.Name).Infof("work is done, stopping...")
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		log.WithField("agent", proc.Name).Errorf("timeout exceeded, stopping...")
	}

	log.WithField("agent", proc.Name).Infof("time elapsed: %v\n", time.Since(init))

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
