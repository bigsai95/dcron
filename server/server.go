package server

import (
	"context"
	"dcron/internal/goworker"

	nsq "github.com/nsqio/go-nsq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

var serverObject *Server

type Server struct {
	env         EnvStruct
	logger      *logrus.Logger
	redisCacher *redis.Client
	gracefulCtx *context.Context
	goworker    *goworker.Pool
	nsqProducer *nsq.Producer
}

type EnvStruct struct {
	AppEnv              string `mapstructure:"APP_ENV" json:"APP_ENV"`
	ServerPort          string `mapstructure:"SERVER_PORT" json:"SERVER_PORT"`
	RedisHost           string `mapstructure:"REDIS_HOST" json:"REDIS_HOST"`
	RedisDB             int    `mapstructure:"REDIS_DB" json:"REDIS_DB"`
	NsqdHost            string `mapstructure:"NSQD_HOST" json:"NSQD_HOST"`
	NsqdMaxInFlight     int    `mapstructure:"NSQD_MAXINFLIGHT" json:"NSQD_MAXINFLIGHT"`
	NsqdDialTimeout     int    `mapstructure:"NSQD_DIALTIMEOUT" json:"NSQD_DIALTIMEOUT"`
	NsqdMaxAttempts     int    `mapstructure:"NSQD_MAXATTEMPTS" json:"NSQD_MAXATTEMPTS"`
	NsqdMaxRequeueDelay int    `mapstructure:"NSQD_MAXREQUEUEDELAY" json:"NSQD_MAXREQUEUEDELAY"`
	WorkerPoolSize      int64  `mapstructure:"WORKER_POOL_SIZE" json:"WORKER_POOL_SIZE"`
	WorkerMaxOpen       int64  `mapstructure:"WORKER_MAX_OPEN" json:"WORKER_MAX_OPEN"`
	WorkerIdle          int64  `mapstructure:"WORKER_IDLE" json:"WORKER_IDLE"`
	WorkerLifeTime      int    `mapstructure:"WORKER_LIFE_TIME" json:"WORKER_LIFE_TIME"`
	GraceShutdownTime   int    `mapstructure:"GRACE_SHUTDOWN_TIME" json:"GRACE_SHUTDOWN_TIME"`
}

func GetServerInstance() *Server {
	if serverObject == nil {
		serverObject = &Server{}
	}
	return serverObject
}

func (s *Server) GetEnv() EnvStruct {
	return s.env
}

func (s *Server) GetLogger() *logrus.Logger {
	return s.logger
}

func (s *Server) GetRedisCacher() *redis.Client {
	return s.redisCacher
}

func (s *Server) GetGracefulCtx() *context.Context {
	return s.gracefulCtx
}

func (s *Server) GetWorker() *goworker.Pool {
	return s.goworker
}

func (s *Server) GetNSQProducer() *nsq.Producer {
	return s.nsqProducer
}
