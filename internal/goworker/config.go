package goworker

import "time"

type Config struct {
	PoolSize       int64
	WorkerMaxOpen  int64
	WorkerIdle     int64
	WorkerLifeTime time.Duration
}

type Option interface {
	apply(*Config)
}

type funcOption struct {
	f func(*Config)
}

func (fo *funcOption) apply(c *Config) {
	fo.f(c)
}

func newFuncOption(f func(*Config)) *funcOption {
	return &funcOption{
		f: f,
	}
}

// WithWorkerMaxOpen Worker最大連線數
func WithWorkerMaxOpen(WorkerMaxOpen int64) Option {
	return newFuncOption(func(c *Config) {
		if WorkerMaxOpen > 0 {
			c.WorkerMaxOpen = WorkerMaxOpen
		}
	})
}

// WithPoolSize job 隊列大小
func WithPoolSize(PoolSize int64) Option {
	return newFuncOption(func(c *Config) {
		if PoolSize > 0 {
			c.PoolSize = PoolSize
		}
	})
}

// WithWorkerIdle Worker閒置數
func WithWorkerIdle(WorkerIdle int64) Option {
	return newFuncOption(func(c *Config) {
		if WorkerIdle > 0 {
			c.WorkerIdle = WorkerIdle
		}
	})
}

// WithWorkerLifeTime Worker存活時間 (在存活時間內都沒有接到job worker就會被清掉)
func WithWorkerLifeTime(WorkerLifeTime time.Duration) Option {
	return newFuncOption(func(c *Config) {
		if WorkerLifeTime.Seconds() > 0 {
			c.WorkerLifeTime = WorkerLifeTime
		}
	})
}
