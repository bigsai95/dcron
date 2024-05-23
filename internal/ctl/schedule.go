package ctl

import (
	"dcron/internal/cronjob"

	"github.com/robfig/cron/v3"
)

const (
	ParameterErrorMsg    = "parameter error"
	EmptyJobIDErrorMsg   = "job_id is empty"
	EmptyMatchErrMsg     = "match is empty"
	EmptyGroupNameErrMsg = "group_name is empty"
	EmptyGameTypeErrMsg  = "game_type is empty"
	EmptyNameErrMsg      = "name is empty"
	JobIDGroupNameErrMsg = "job_id or group_name is error"
)

func AddJobSchedule(payload cronjob.TaskPayload) (entryID cron.EntryID, err error) {
	_, ok := cronjob.Mgr.LoadJobMapping(payload.JobID)
	if !ok {
		entryID, err = cronjob.Mgr.AddJob(payload.IntervalPattern, &payload)
		if err == nil {
			cronjob.Mgr.StoreJobMapping(payload.JobID, entryID)
		}
	}

	return entryID, err
}

func AddJobFromSchedule(payload cronjob.TaskPayload) error {
	_, ok := cronjob.Mgr.LoadJobMapping(payload.JobID)
	if !ok {
		cronjob.Mgr.ImportAddJobs(payload)
	}
	return nil
}

func RemoveJobFromSchedule(jobID string) {
	entryID, ok := cronjob.Mgr.LoadJobMapping(jobID)
	if ok {
		cronjob.Mgr.Remove(entryID)
		cronjob.Mgr.DeleteJobMapping(jobID)
	}
}

func ActiveJobFromSchedule(payload cronjob.TaskPayload) error {
	_, ok := cronjob.Mgr.LoadJobMapping(payload.JobID)
	if !ok {
		// 註冊新的entry
		entryID, err := cronjob.Mgr.AddJob(payload.IntervalPattern, &payload)
		if err != nil {
			return err
		}
		// 把Job的 status 1
		err = UpdateJobStatus(payload.GroupName, payload.JobID, 1)
		if err != nil {
			// 發生錯誤時，需要移除剛剛添加的 entry
			cronjob.Mgr.Remove(entryID)
			return err
		}
		cronjob.Mgr.StoreJobMapping(payload.JobID, entryID)
	}
	return nil
}

func PauseJobFromSchedule(groupName, jobID string) error {
	entryID, ok := cronjob.Mgr.LoadJobMapping(jobID)
	if ok {
		// 把Job的 status 0
		err := UpdateJobStatus(groupName, jobID, 0)
		if err != nil {
			return err
		}
		cronjob.Mgr.Remove(entryID)
		cronjob.Mgr.DeleteJobMapping(jobID)
	}
	return nil
}
