package cronjob

import (
	"context"
	"dcron/internal/httptarget"
	"dcron/internal/lib"
	"dcron/internal/nsqtarget"
	"dcron/internal/redisCacher"
	"fmt"
	"time"
)

const (
	HttpMode = "http"
	NsqMode  = "nsq"
	TestMode = "test"
)

type PubJob struct {
	JobID     string `json:"job_id"`     // 排程ID
	GroupName string `json:"group_name"` // 群組名稱
	Name      string `json:"name"`       // 排程名稱
	Event     string `json:"event"`
	HostName  string `json:"host_name"`
}

type TaskPayloadReq struct {
	GroupName       string `json:"group_name" validate:"required" example:"test"`              // 群組名稱
	Name            string `json:"name" validate:"required" example:"job01"`                   // 排程名稱
	ExecRightNow    bool   `json:"exec_right_now" example:"false"`                             // true: 馬上執行
	RequestUrl      string `json:"request_url" example:"http://127.0.0.1/api/ping"`            // 網址
	Retry           bool   `json:"retry" example:"false"`                                      // true: http失敗重新執行
	IntervalPattern string `json:"interval_pattern" validate:"required" example:"0 * * * * *"` // 支援 `0 0 * * * *` `@hourly` `1685935821`
	Type            string `json:"type" validate:"required" example:"http"`                    // `nsq` `http`
	NsqTopic        string `json:"nsq_topic" example:""`                                       // type選擇nsq,topic不能為空
	NsqMessage      string `json:"nsq_message" example:""`                                     // nsq回傳的訊息 ex.{"game_name": "BBLT","draw_mode":1,"open_timestamp": 1686116327,"close_timestamp": 1686116387}
}

type TaskPayload struct {
	JobID           string    `json:"job_id"`                 // 排程ID
	GroupName       string    `json:"group_name"`             // 群組名稱
	Name            string    `json:"name"`                   // 排程名稱
	ExecRightNow    bool      `json:"exec_right_now"`         // true: 馬上執行
	RequestUrl      string    `json:"request_url"`            // 網址
	Retry           bool      `json:"retry"`                  // true: http失敗重新執行
	IntervalPattern string    `json:"interval_pattern"`       // 支援 `0 0 * * * *` `@hourly` `1685935821`
	Type            string    `json:"type"`                   // `nsq` `http`
	Status          int       `json:"status"`                 // 1:執行中
	NsqTopic        string    `json:"nsq_topic"`              // type選擇nsq,topic不能為空
	NsqMessage      string    `json:"nsq_message" example:""` // nsq回傳的訊息
	Register        time.Time `json:"register"`               // 註冊時間
	Next            time.Time `json:"next"`                   // 下次執行時間
	Prev            time.Time `json:"prev"`                   // 上次執行時間
	Memo            string    `json:"memo"`                   // 備註
}

func (j *TaskPayload) Run() {
	loc, _ := time.LoadLocation("Asia/Taipei")
	currentTime := time.Now().In(loc)
	t1 := currentTime.UnixMilli()

	logInfo := logger.WithFields(map[string]interface{}{
		"func":       "payload_run",
		"group_name": j.GroupName,
		"job_name":   j.Name,
		"job_id":     j.JobID,
		"cron_type":  j.Type,
		"memo":       j.Memo,
		"nsq_topic":  j.NsqTopic,
		"run_params": j.NsqMessage,
		"cron_time":  t1,
		"exec_time":  currentTime,
	})

	isOnce := lib.IsMemoOnce(j.Memo)
	if isOnce {
		isWithinExec := j.ExecRightNow || lib.ShouldExecuteThreeHours(j.Memo)
		if !isWithinExec {
			logInfo.Debug("job cronjob expired")
			j.runOnce()
			return
		}
		if !j.acquireLock(fmt.Sprintf("LOCK_ONCE_%s", j.JobID), "once", 30) {
			return
		}
	} else {
		if !j.acquireLock(fmt.Sprintf("LOCK_%s_%d", j.JobID, currentTime.Unix()), 60, 5) {
			return
		}
	}

	logInfo.Debug("job cronjob run")

	switch j.Type {
	case HttpMode:
		j.runHttp()
	case NsqMode:
		j.runNsq()
	case TestMode:
		j.runTest()
	}
}

func (j *TaskPayload) acquireLock(key string, value interface{}, expiration int64) bool {
	exists, err := redisCacher.Conn.SetNX(key, value, expiration)
	return exists && err == nil
}

func (j *TaskPayload) runTest() {
	key := fmt.Sprintf("TestCheck_%s", j.Name)
	redisCacher.Conn.Set(key, "test_ok", 20)
	j.runOnce()
}

func (j *TaskPayload) runHttp() {
	maxCount := 0
	if j.Retry {
		maxCount = 3
	}

	res := httptarget.NewEntryScan(context.Background(), j.RequestUrl, maxCount).Scan()
	if res.Code != 200 && res.Err != nil {
		logger.WithFields(map[string]interface{}{
			"func":       "payload_run",
			"step":       "payload_run_http",
			"group_name": j.GroupName,
			"job_name":   j.Name,
			"job_id":     j.JobID,
			"url":        j.RequestUrl,
			"cron_type":  j.Type,
			"res_code":   res.Code,
			"res_count":  res.Count,
			"res_error":  res.Err.Error(),
		}).Error("http error")
	}
	j.runOnce()
}

func (j *TaskPayload) runNsq() {
	err := nsqtarget.Publish(j.NsqTopic, j.NsqMessage)
	if err != nil {
		logger.WithFields(map[string]interface{}{
			"func":       "payload_run",
			"step":       "payload_run_nsq",
			"group_name": j.GroupName,
			"job_name":   j.Name,
			"job_id":     j.JobID,
			"cron_type":  j.Type,
			"nsq_topic":  j.NsqTopic,
			"memo":       j.Memo,
			"err":        err.Error(),
		}).Error("nsq error")
	}
	j.runOnce()
}

func (j *TaskPayload) runOnce() {
	if lib.IsMemoOnce(j.Memo) {
		j.cleanUpOnce()
		return
	}

	entryID, ok := Mgr.LoadJobMapping(j.JobID)
	if ok {
		dataEntry := Mgr.Entry(entryID)
		values := map[string]interface{}{
			"next": dataEntry.Next,
			"prev": dataEntry.Prev,
		}
		key := fmt.Sprintf("TIME_%s_%s", j.GroupName, j.JobID)
		redisCacher.Conn.HSet(key, values, 2*24*60*60)
	}
}

func (j *TaskPayload) cleanUpOnce() {
	redisCacher.Conn.HDel(fmt.Sprintf("TEAM_%s", j.GroupName), j.Name)
	redisCacher.Conn.Del(fmt.Sprintf("CK_%s_%s", j.GroupName, j.Name))
	redisCacher.Conn.Del(fmt.Sprintf("TASK_%s_%s", j.GroupName, j.JobID))

	entryID, ok := Mgr.LoadJobMapping(j.JobID)
	if ok {
		Mgr.Remove(entryID)
		Mgr.DeleteJobMapping(j.JobID)
	}
}
