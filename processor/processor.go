package processor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/Projeto-USPY/uspy-scraper/utils"
	log "github.com/sirupsen/logrus"
)

type (
	Processed interface{}

	Task struct {
		IDs log.Fields

		Process func(ctx context.Context) (Processed, error)
		After   func(ctx context.Context, results Processed) error
		DelayFn func(attempt int) time.Duration

		currentAttempt int
		lastTried      time.Time
		err            error
	}

	Processor struct {
		IDs   log.Fields
		Tasks []*Task

		numWorkers    int
		maxAttempts   int
		timeout       int
		delayAttempts bool

		ctx     context.Context
		jobs    chan *Task
		failed  chan *Task
		results chan Processed
	}
)

func NewTask(
	IDs log.Fields,
	delayFn func(int) time.Duration,
	processFn func(context.Context) (Processed, error),
	afterFn func(context.Context, Processed) error,
) *Task {
	return &Task{
		IDs:     IDs,
		Process: processFn,
		After:   afterFn,
		DelayFn: delayFn,

		currentAttempt: 0,
		lastTried:      time.Now(),
		err:            nil,
	}
}

func NewProcessor(
	ctx context.Context,
	IDs log.Fields,
	tasks []*Task,
	fixedAttempts,
	delayAttempts bool,
) *Processor {
	numWorkers := Config.NumWorkers
	if Config.FractionalNumWorkers > 0.0 && Config.FractionalNumWorkers <= 1.0 {
		numWorkers = utils.MaxInt(1, int(math.Ceil(float64(len(tasks))*Config.FractionalNumWorkers)))
		numWorkers = utils.MinInt(numWorkers, Config.MaxWorkers)
	}

	maxAttempts := -1
	if fixedAttempts {
		maxAttempts = Config.MaxAttempts
	}

	return &Processor{
		IDs:           IDs,
		Tasks:         tasks,
		numWorkers:    numWorkers,
		maxAttempts:   maxAttempts,
		delayAttempts: delayAttempts,
		timeout:       Config.Timeout,

		ctx:     ctx,
		jobs:    make(chan *Task, len(tasks)),
		results: make(chan Processed, len(tasks)),
		failed:  make(chan *Task, len(tasks)),
	}
}

func (proc *Processor) Run() (results []Processed) {
	log := log.WithField("workers", proc.numWorkers).WithFields(proc.IDs)

	var ctx = proc.ctx
	var cancel context.CancelFunc

	if Config.Timeout != -1 {
		ctx, cancel = context.WithTimeout(
			ctx,
			time.Duration(proc.timeout*int(time.Second)),
		)
	} else {
		ctx, cancel = context.WithCancel(
			ctx,
		)
	}

	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			log.Errorf("unexpected panic in run processor: %s", r)
			results = nil
			cancel()
			return
		}
	}()

	init := time.Now()

	defer close(proc.jobs)
	defer close(proc.results)
	defer close(proc.failed)

	// prepare workers
	log.Info("launching workers")
	for i := 0; i < proc.numWorkers; i++ {
		go func(ctx context.Context) {
			defer func() {
				if r := recover(); r != nil {
					log.Errorf("unexpected panic in job goroutine: %s", r)
					results = nil
					cancel()
					return
				}
			}()
			for {
				select {
				case <-ctx.Done():
					return
				case job := <-proc.jobs:
					log := log.WithFields(job.IDs)
					log.Debug("starting job")

					// fail job if exceeded retries
					if proc.maxAttempts != -1 && job.currentAttempt > proc.maxAttempts {
						log.Warn("job exceeded max retries, failing...")
						proc.failed <- job
						break
					}

					var result Processed

					failed := false
					denied := false

					var failedErr error

					if proc.delayAttempts && job.DelayFn == nil {
						panic("invalid configuration, delayAttempts is true but delayFn is nil")
					}

					// only run job if sufficient time has passed
					if !proc.delayAttempts || time.Since(job.lastTried) >= job.DelayFn(job.currentAttempt) {
						var processingErr error
						log.Debug("executing process callback")
						result, processingErr = job.Process(ctx)
						if processingErr != nil {
							failed = true // processing failure
							failedErr = fmt.Errorf("%s: %s", "processing error", processingErr.Error())
						} else if job.After != nil {
							log.Debug("executing after-process callback")
							if err := job.After(ctx, result); err != nil {
								failed = true // after-processing failure
								failedErr = fmt.Errorf("%s: %s", "after-processing error", processingErr.Error())
							}
						}
					} else {
						denied = true // job denied by now
					}

					if denied {
						// create goroutine to wait for delay
						log.Warn("job denied by now...")
						proc.jobs <- job
					} else if failed {
						log.Error("job failed...")

						job.currentAttempt++
						job.lastTried = time.Now()
						job.err = failedErr

						proc.jobs <- job
					} else {
						log.Debug("job successful")
						proc.results <- result
					}
				}

			}
		}(ctx)
	}

	// send jobs
	log.Info("sending jobs")
	go func() {
		for _, task := range proc.Tasks {
			proc.jobs <- task
		}
	}()

	// receive job results
	results = make([]Processed, 0)
	failures := make([]*Task, 0)

	// wait for all jobs to fail, succeed or timeout
ConsumeLoop:
	for {
		select {
		case result := <-proc.results:
			results = append(results, result)
		case failure := <-proc.failed:
			failures = append(failures, failure)
		default:
			if ctx.Err() != nil || len(results)+len(failures) == len(proc.Tasks) {
				cancel()
				break ConsumeLoop
			}
		}
	}

	if errors.Is(ctx.Err(), context.Canceled) {
		log.Info("work is done, stopping...")
	} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		log.Errorf("timeout exceeded, stopping...")
	}

	if len(proc.failed) > 0 {
		log.Warnln(len(proc.failed), "jobs failed")

		for _, failedJob := range failures {
			if failedJob.err != nil {
				log.Errorf("[failed] job, reason: %s", failedJob.err.Error())
			}
		}
	}

	log.Infof("processor finished, time elapsed: %v", time.Since(init))
	return results
}
