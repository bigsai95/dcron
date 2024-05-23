package goworker

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Gorouting instance which can accept client jobs
type worker struct {
	workerPool chan *worker
	jobChannel chan Job
	stop       chan struct{}
	dispatcher *dispatcher
	lifeTime   time.Duration
	closed     bool
	wlock      *sync.Mutex
}

func (w *worker) isClosed() bool {
	w.wlock.Lock()
	defer w.wlock.Unlock()

	return w.closed
}

func (w *worker) start() {
	if w.lifeTime.Seconds() >= 1 {
		w.startWithLifeTime()
	} else {
		w.startNoLifeTime()
	}
}

func (w *worker) startNoLifeTime() {
	go func() {
		var job Job
		for {
			if len(w.jobChannel) == 0 {
				w.workerPool <- w
			}

			select {
			case job = <-w.jobChannel:
				job.Func(job.Param)
				w.dispatcher.done()
			case <-w.stop:
				w.close()
				w.stop <- struct{}{}

				return
			}
		}
	}()
}

func (w *worker) startWithLifeTime() {
	go func() {
		var job Job
		for {
			// worker free, add it to pool
			if len(w.jobChannel) == 0 {
				w.workerPool <- w
			}

			select {
			case job = <-w.jobChannel:
				job.Func(job.Param)
				w.dispatcher.done()
			case <-w.stop:
				w.close()

				w.stop <- struct{}{}

				return
			case <-time.After(w.lifeTime): // 如果整個life time都沒job進來 就return掉
				w.close()

				return
			}
		}
	}()
}

func (w *worker) close() {
	w.wlock.Lock()
	defer w.wlock.Unlock()

	w.closed = true
	w.dispatcher.workerSub()
	close(w.jobChannel)
}

func newWorker(pool chan *worker, d *dispatcher, lifeTime time.Duration) *worker {
	return &worker{
		workerPool: pool,
		jobChannel: make(chan Job, 1),
		stop:       make(chan struct{}),
		dispatcher: d,
		lifeTime:   lifeTime,
		wlock:      &sync.Mutex{},
	}
}

// Accepts jobs from clients, and waits for first free worker to deliver job
type dispatcher struct {
	workerPool chan *worker // 空閒worker數
	jobQueue   chan Job
	stop       chan struct{}
	wg         sync.WaitGroup
	count      int64 // 全部job的數量 jobQueue + job接走的
	workerNum  int64 // 全部worker數
	maxOpen    int64
	lifeTime   time.Duration
}

func (d *dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			d.findWorker(job)
		case <-d.stop:
			totalPoolNum := len(d.workerPool)
			for i := 0; i < totalPoolNum; i++ {
				worker, ok := <-d.workerPool
				if !ok {
					return
				}

				if !worker.isClosed() {
					worker.stop <- struct{}{}
					<-worker.stop
				}
			}

			d.stop <- struct{}{}
			return
		}
	}
}

func (d *dispatcher) findWorker(job Job) {
	// 如果workerPool是0 && 沒有到worker上限 就加worker
	if len(d.workerPool) == 0 && d.workerNum < d.maxOpen {
		d.newWorkerWithJob(job)

		return
	}

	worker := <-d.workerPool // 從pool取worker

	// worker life time過了可能已經停了 但還在pool裡面 所以要判斷一下開關
	if worker.isClosed() {
		d.newWorkerWithJob(job)

		return
	}

	worker.jobChannel <- job
}

func (d *dispatcher) newWorkerWithJob(job Job) {
	d.workerAdd()
	worker := newWorker(d.workerPool, d, d.lifeTime)
	worker.jobChannel <- job
	worker.start()
}

func (d *dispatcher) add(i int) {
	atomic.AddInt64(&d.count, int64(i))
	d.wg.Add(i)
}

func (d *dispatcher) done() {
	atomic.AddInt64(&d.count, -1)
	d.wg.Done()
}

func (d *dispatcher) workerAdd() {
	atomic.AddInt64(&d.workerNum, 1)
}

func (d *dispatcher) workerSub() {
	atomic.AddInt64(&d.workerNum, -1)
}

func (d *dispatcher) workCount() int64 {
	return atomic.LoadInt64(&d.count)
}

func (d *dispatcher) workerCount() int64 {
	return atomic.LoadInt64(&d.workerNum)
}

func (d *dispatcher) wait() {
	d.wg.Wait()
}

func newDispatcher(workerPool chan *worker, jobQueue chan Job, maxOpen, idle int64, lifeTime time.Duration) *dispatcher {
	d := &dispatcher{
		workerPool: workerPool,
		jobQueue:   jobQueue,
		stop:       make(chan struct{}),
		maxOpen:    maxOpen,
		lifeTime:   lifeTime,
	}

	for i := 0; i < int(idle); i++ {
		worker := newWorker(d.workerPool, d, 0) // Idle的worker不設life time
		worker.start()
		d.workerAdd()
	}

	go d.dispatch()

	return d
}

// Job Represents user request, function which should be executed in some worker.
type Job struct {
	Param []interface{}
	Func  func([]interface{})
}

// Pool Goworker pool
type Pool struct {
	jobQueue   chan Job
	dispatcher *dispatcher
	alive      bool
	lock       *sync.Mutex
}

/*
NewPool Create goruntine pool by limited worker numbers that can being queue, pause by ctx
Use doneFunc to handle job when receice ctx.Done singnal
*/
func NewPool(o ...Option) *Pool {
	conf := &Config{
		PoolSize:       3000,
		WorkerMaxOpen:  1000,
		WorkerIdle:     100,
		WorkerLifeTime: 60 * time.Second,
	}

	for _, f := range o {
		f.apply(conf)
	}

	// 防呆
	if conf.WorkerIdle > conf.WorkerMaxOpen {
		conf.WorkerIdle = conf.WorkerMaxOpen
	}

	jobQueue := make(chan Job, conf.PoolSize)
	workerPool := make(chan *worker, conf.WorkerMaxOpen)

	pool := &Pool{
		jobQueue:   jobQueue,
		dispatcher: newDispatcher(workerPool, jobQueue, conf.WorkerMaxOpen, conf.WorkerIdle, conf.WorkerLifeTime),
		alive:      true,
		lock:       &sync.Mutex{},
	}

	return pool
}

// StopAddJob 停止再接新的job
func (p *Pool) StopAddJob() {
	p.lock.Lock()
	p.alive = false
	p.lock.Unlock()
}

// Wait Will wait for all jobs to finish.
func (p *Pool) Wait() {
	p.dispatcher.wait()
}

// Release Will release resources used by pool
func (p *Pool) Release() {
	p.jobQueue = nil

	p.dispatcher.stop <- struct{}{}
	<-p.dispatcher.stop
}

// GracefulStop GracefulStop
func (p *Pool) GracefulStop() {
	p.StopAddJob() // 停止不再接新的job
	p.Wait()       // 等全部的job都做完
	p.Release()    // 清worker pool
}

// JobQueue 把任務加到job queue *注意*GracefulStop後的job是沒辦法進來
func (p *Pool) JobQueue(todo Job) {
	p.lock.Lock()
	if !p.alive {
		return
	}
	p.lock.Unlock()

	p.dispatcher.add(1)
	p.jobQueue <- todo
}

// JobQueueNotWait 把任務加到job queue 如果滿了不等 直接回傳error
func (p *Pool) JobQueueNotWait(todo Job) error {
	p.lock.Lock()
	if !p.alive {
		return errors.New("worker pool closed")
	}
	p.lock.Unlock()

	if len(p.jobQueue) == cap(p.jobQueue) {
		return errors.New("job queue was full")
	}

	p.dispatcher.add(1)
	p.jobQueue <- todo

	return nil
}

// JobQueueLen job queue數量
func (p *Pool) JobQueueLen() int {
	return len(p.jobQueue)
}

// WorkingJobCount 正在執行job數量
func (p *Pool) WorkingJobCount() int64 {
	return p.dispatcher.workCount()
}

// WorkerCount 目前worker數量
func (p *Pool) WorkerCount() int64 {
	return p.dispatcher.workerCount()
}

func (p *Pool) IsAlive() bool {
	return p.alive
}

func DoJob(callback func([]interface{})) Job {
	return Job{
		Func: callback,
	}
}

func DoJobParams(callback func([]interface{}), i ...interface{}) Job {
	return Job{
		Param: i,
		Func:  callback,
	}
}
