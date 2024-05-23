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

	ListenChannel = "dgua_event_channel"
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

	gType, gNum := lib.MatchJobName(j.Name)
	logInfo := logger.WithFields(map[string]interface{}{
		"func":       "payload_run",
		"group_name": j.GroupName,
		"job_name":   j.Name,
		"job_id":     j.JobID,
		"cron_type":  j.Type,
		"memo":       j.Memo,
		"nsq_topic":  j.NsqTopic,
		"run_params": j.NsqMessage,
		"game_type":  gType,
		"game_num":   gNum,
		"cron_time":  t1,
		"exec_time":  currentTime,
	})

	var isWithinExec bool
	isOnce := lib.IsMemoOnce(j.Memo)
	if isOnce {
		// ExecRightNow 優先權 > lib.ShouldExecuteThreeHours
		if j.ExecRightNow {
			isWithinExec = true
		} else {
			isWithinExec = lib.ShouldExecuteThreeHours(j.Memo)
		}
		if !isWithinExec {
			logInfo.Debug("job cronjob expired")

			// 過期太久期數不執行
			j.runOnce()
			return
		}

		lockKey := fmt.Sprintf("LOCK_ONCE_%s", j.JobID)
		exists, err := redisCacher.Conn.SetNX(lockKey, "once", 30)
		if !exists || err != nil {
			return
		}
	} else {
		lockKey := fmt.Sprintf("LOCK_%s_%d", j.JobID, currentTime.Unix())
		exists, err := redisCacher.Conn.SetNX(lockKey, 60, 5)
		if !exists || err != nil {
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

func (j *TaskPayload) runTest() {
	key := fmt.Sprintf("TestCheck_%s", j.Name)
	redisCacher.Conn.Set(key, "test_ok", 20)

	j.runOnce()
}

func (j *TaskPayload) runHttp() {
	var maxCount int
	if j.Retry {
		maxCount = 3
	}

	m := httptarget.NewEntryScan(j.RequestUrl, maxCount)
	res := m.Scan(context.Background())
	if res.Code != 200 {
		if res.Err != nil {
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
	isOnce := lib.IsMemoOnce(j.Memo)
	if !isOnce {
		var prev, next time.Time
		entryID, ok := Mgr.LoadJobMapping(j.JobID)
		if ok {
			dataEntry := Mgr.Entry(entryID)
			next = dataEntry.Next
			prev = dataEntry.Prev
		}
		values := map[string]interface{}{
			"next": next,
			"prev": prev,
		}
		key := fmt.Sprintf("TIME_%s_%s", j.GroupName, j.JobID)
		redisCacher.Conn.HSet(key, values, 2*24*60*60)

		return
	}

	key := fmt.Sprintf("TEAM_%s", j.GroupName)
	redisCacher.Conn.HDel(key, j.Name)

	key = fmt.Sprintf("CK_%s_%s", j.GroupName, j.Name)
	redisCacher.Conn.Del(key)

	key = fmt.Sprintf("TASK_%s_%s", j.GroupName, j.JobID)
	redisCacher.Conn.Del(key)

	// 一次性的排程移除
	entryID, ok := Mgr.LoadJobMapping(j.JobID)
	if ok {
		Mgr.Remove(entryID)
		Mgr.DeleteJobMapping(j.JobID)
	}
}
