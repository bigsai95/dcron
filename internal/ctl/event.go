package ctl

import (
	"dcron/internal/cronjob"
	"strings"

	"github.com/robfig/cron/v3"
)

func EventForAddSchedule(payload cronjob.TaskPayload) (cron.EntryID, error) {
	return AddJobSchedule(payload)
}

func EventHandling(pub cronjob.PubJob) {
	switch strings.ToLower(pub.Event) {
	case "add":
		payload := GetTaskPayload(pub.GroupName, pub.JobID)
		AddJobFromSchedule(payload)
	case "pause":
		PauseJobFromSchedule(pub.GroupName, pub.JobID)
	case "active":
		payload := GetTaskPayload(pub.GroupName, pub.JobID)
		ActiveJobFromSchedule(payload)
	case "delete":
		RemoveJobFromSchedule(pub.JobID)
	case "stop":
		cronjob.Mgr.Stop()
	case "start":
		cronjob.Mgr.Start()
	}
}
