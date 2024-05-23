package server

import (
	"context"
	"dcron/internal/goworker"
	"os"
	"time"

	nsq "github.com/nsqio/go-nsq"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/urfave/cli"
)

func Loader(c *cli.Context, ctx *context.Context) {
	logger := initLog()
	env := initEnv(c)
	redisCacher := initRedisConn(env)
	worker := initWorker(env.WorkerPoolSize, env.WorkerMaxOpen, env.WorkerIdle, env.WorkerLifeTime)
	producer, err := initNsqProducer(env)
	if err != nil {
		logger.Error("initNsqProducer:", err)
		panic("initNsqProducer fail")
	}

	serverObject = &Server{
		env:         env,
		logger:      logger,
		redisCacher: redisCacher,
		gracefulCtx: ctx,
		goworker:    worker,
		nsqProducer: producer,
	}
}

// 載入環境變數
func initEnv(c *cli.Context) (env EnvStruct) {
	viper.SetConfigFile(c.String("config"))
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
	err = viper.Unmarshal(&env)
	if err != nil {
		panic(err)
	}

	return
}

func initLog() *logrus.Logger {
	logger := logrus.New()

	logger.SetFormatter(&logrus.JSONFormatter{})

	logger.SetOutput(os.Stdout)

	logger.SetLevel(logrus.DebugLevel)

	return logger
}

func initRedisConn(env EnvStruct) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: env.RedisHost,
		DB:   env.RedisDB,
	})
}

func initWorker(poolSize, maxOpen, idle int64, lifeTime int) *goworker.Pool {
	worker := goworker.NewPool(
		goworker.WithPoolSize(poolSize),
		goworker.WithWorkerMaxOpen(maxOpen),
		goworker.WithWorkerIdle(idle),
		goworker.WithWorkerLifeTime(time.Second*time.Duration(lifeTime)))

	return worker
}

func initNsqProducer(env EnvStruct) (*nsq.Producer, error) {
	cfg := nsq.NewConfig()
	cfg.MaxInFlight = env.NsqdMaxInFlight
	cfg.DialTimeout = time.Duration(env.NsqdDialTimeout) * time.Second
	cfg.MaxAttempts = uint16(env.NsqdMaxAttempts)
	cfg.MaxRequeueDelay = time.Duration(env.NsqdMaxRequeueDelay) * time.Millisecond

	producer, err := nsq.NewProducer(env.NsqdHost, cfg)
	if err != nil {
		return nil, err
	}

	if err := producer.Ping(); err != nil {
		return nil, err
	}

	return producer, nil
}
