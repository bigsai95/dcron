package main

import (
	"context"
	"dcron/handler"
	"dcron/httpserver"
	"dcron/internal/cronjob"
	"dcron/internal/nsqtarget"
	"dcron/internal/redisCacher"
	"dcron/internal/snowflake"
	"dcron/server"

	"net/http"
	"os"
	"os/signal"
	"syscall"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "crontab"
	app.Version = "1.0.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "config, c",
			Value:  "config.yaml",
			Usage:  "app config",
			EnvVar: "CONFIG_PATH",
		},
	}
	app.Action = start
	err := app.Run(os.Args)
	if err != nil {
		logrus.Panic(err)
	}
}

func start(c *cli.Context) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	server.Loader(c, &ctx)

	osc := make(chan os.Signal, 1)

	signal.Notify(osc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGTERM)

	publicError := make(chan error)

	logger := server.GetServerInstance().GetLogger()

	env := server.GetServerInstance().GetEnv()

	logger.WithFields(logrus.Fields{
		"ENV": env,
	}).Info("Cofing 設定成功")

	handlerServer := &handler.Server{}

	addr := ":" + env.ServerPort
	gin.SetMode("release")
	r := gin.New()
	r.Use(gin.Recovery())

	r, err := httpserver.InitRouter(r)
	if err != nil {
		logger.Fatalf("failed to initialize router: %v", err)
	}

	httpSrv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	go func() {
		logger.Printf("HTTP Server Start on %s", addr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("failed to start HTTP server: %v", err)
		}
	}()

	loadConfig(ctx)

	go handlerServer.Background(ctx)

	// Handle signals and errors concurrently
	select {
	case signal := <-osc:
		logger.Printf("Received signal: %s", signal)
		cancel()
	case err := <-publicError:
		logger.Printf("Public gRPC error: %v", err)
	}

	// Shutdown HTTP server gracefully
	if err := httpSrv.Shutdown(ctx); err != nil {
		logger.Printf("HTTP server graceful shutdown error: %v", err)
	}
	logger.Println("HTTP Server Shutdown End...")

	server.GetServerInstance().GetWorker().GracefulStop()
	logger.Info("停止Goworker完成")

	<-time.After(time.Duration(env.GraceShutdownTime) * time.Second)
	logger.Println("退出完成...")
}

func loadConfig(ctx context.Context) {
	redisCacher.ConfigInit()
	cronjob.ConfigInit()
	nsqtarget.ConfigInit()
	snowflake.ConfigInit()

	// 調整 cronjob 啟動流程,避免 cronjob 未準備好, 就有註冊資料進來
	go cronjob.Mgr.StartInit(ctx, false)
}
