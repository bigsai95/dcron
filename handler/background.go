package handler

import (
	"context"
	"dcron/internal/cronjob"
	"dcron/internal/ctl"
	"dcron/server"
	"time"
)

// Background Background
func (s *Server) Background(ctx context.Context) {
	// 系統啟動完畢
	cronjob.Mgr.SetPingSuccessful(true)

	time.Sleep(200 * time.Microsecond)

	server.GetServerInstance().GetLogger().Info("Cronjob import jobs start")
	// 匯入目前cronjob工作
	jobs, _ := ctl.GetJobsByAll()
	if len(jobs) > 0 {
		cronjob.Mgr.ImportJobs(jobs)
	}
	server.GetServerInstance().GetLogger().Info("Cronjob import jobs end")
}
