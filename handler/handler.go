package handler

import (
	"context"
	"dcron/internal/cronjob"
	"dcron/internal/ctl"
	"dcron/internal/lib"
	"dcron/internal/snowflake"
	"dcron/server"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

type Server struct {
	ServerStruct *server.Server
}

type DataReply struct {
	Success bool `protobuf:"varint,1,opt,name=success,proto3" json:"success,omitempty"`
}

type TaskPayloadRequest struct {
	GroupName       string `protobuf:"bytes,1,opt,name=group_name,json=groupName,proto3" json:"group_name,omitempty"`
	Name            string `protobuf:"bytes,2,opt,name=name,proto3" json:"name,omitempty"`
	ExecRightNow    bool   `protobuf:"varint,3,opt,name=exec_right_now,json=execRightNow,proto3" json:"exec_right_now,omitempty"`
	RequestUrl      string `protobuf:"bytes,4,opt,name=request_url,json=requestUrl,proto3" json:"request_url,omitempty"`
	Retry           bool   `protobuf:"varint,5,opt,name=retry,proto3" json:"retry,omitempty"`
	IntervalPattern string `protobuf:"bytes,6,opt,name=interval_pattern,json=intervalPattern,proto3" json:"interval_pattern,omitempty"`
	Type            string `protobuf:"bytes,7,opt,name=type,proto3" json:"type,omitempty"`
	NsqTopic        string `protobuf:"bytes,8,opt,name=nsq_topic,json=nsqTopic,proto3" json:"nsq_topic,omitempty"`
	NsqMessage      string `protobuf:"bytes,9,opt,name=nsq_message,json=nsqMessage,proto3" json:"nsq_message,omitempty"`
}

func (s *Server) AddJob(ctx context.Context, d1 *TaskPayloadRequest) (d0 *DataReply, err error) {
	var (
		execRightNow bool
		memoOnce     bool
	)
	d0 = &DataReply{}

	payload := cronjob.TaskPayload{
		GroupName:       d1.GroupName,
		Name:            d1.Name,
		ExecRightNow:    d1.ExecRightNow,
		RequestUrl:      d1.RequestUrl,
		Retry:           d1.Retry,
		IntervalPattern: d1.IntervalPattern,
		Type:            d1.Type,
		NsqTopic:        d1.NsqTopic,
		NsqMessage:      d1.NsqMessage,
	}

	if payload.GroupName == "" {
		err = errors.New(ctl.EmptyGroupNameErrMsg)
		return
	}
	if payload.Name == "" {
		err = errors.New(ctl.EmptyNameErrMsg)
		return
	}
	payload.Type = strings.ToLower(payload.Type)
	if payload.Type == "" {
		err = errors.New("type is empty")
		return
	} else if payload.Type == cronjob.HttpMode {
		if payload.RequestUrl == "" {
			err = errors.New("url is empty")
			return
		}
		_, err = url.ParseRequestURI(payload.RequestUrl)
		if err != nil {
			return
		}
	} else if payload.Type == cronjob.NsqMode {
		if payload.NsqTopic == "" {
			err = errors.New("nsq topic is empty")
			return
		}
		if payload.NsqMessage == "" {
			err = errors.New("nsq message is empty")
			return
		}
		var dataJson map[string]interface{}
		err = json.Unmarshal([]byte(payload.NsqMessage), &dataJson)
		if err != nil {
			err = errors.New("nsq message is not josn")
			return
		}
	}

	loc, _ := time.LoadLocation("Asia/Taipei")
	now := time.Now().In(loc)

	jobID := snowflake.GenerateString()
	traceID := "trace-id-" + jobID

	taskTime, err := decimal.NewFromString(payload.IntervalPattern)
	if err == nil {
		payload.Memo = payload.IntervalPattern + "@once"
		memoOnce = true
	}
	payload.JobID = jobID
	payload.Status = 1
	payload.Register = now

	gType, gNum := lib.MatchJobName(payload.Name)
	logInfo := server.GetServerInstance().GetLogger().WithFields(map[string]interface{}{
		"func":       "job_add_register",
		"trace_id":   traceID,
		"group_name": payload.GroupName,
		"job_name":   payload.Name,
		"job_id":     payload.JobID,
		"cron_type":  payload.Type,
		"memo":       payload.Memo,
		"nsq_topic":  payload.NsqTopic,
		"run_params": payload.NsqMessage,
		"game_type":  gType,
		"game_num":   gNum,
	})

	exists, err := ctl.AcquireLock(payload.GroupName, payload.Name, 0)
	if !exists || err != nil {
		logInfo.Debug("job already registered")
		err = errors.New("group_name + name has already been registered")
		return
	}

	payload.IntervalPattern, err = cronjob.Mgr.Parse(payload.IntervalPattern)
	if err != nil {
		ctl.ReleaseLock(payload.GroupName, payload.Name)
		return
	}

	err = ctl.SetTaskPayload(payload, 0)
	if err != nil {
		ctl.ReleaseLock(payload.GroupName, payload.Name)
		cronjob.Mgr.DeleteJobMapping(jobID)
		return
	}

	execRightNow = lib.ShouldExecuteNow(payload.Memo)

	if !execRightNow {
		// 過期馬上執行, 不需進入cronjob
		_, err = ctl.EventForAddSchedule(payload)
		if err != nil {
			ctl.ReleaseLock(payload.GroupName, payload.Name)
			cronjob.Mgr.DeleteJobMapping(jobID)
			return
		}

	}

	t1 := taskTime.IntPart()
	crontime := time.Unix(t1, 0).In(loc)
	execTime := crontime.Add(600 * time.Millisecond)
	delay := execTime.Sub(now)
	formattedDelay := fmt.Sprintf("%.2f", delay.Seconds())

	logInfo.WithField("delay_time", formattedDelay).Debug("job add success")

	if payload.ExecRightNow || execRightNow {
		payload.Run()
	} else if memoOnce {
		if delay.Seconds() < 1.3 && delay.Seconds() > 0 {
			// 修正bug - 執行時間太接近, 任務新增完成後就過了預訂時間
			time.AfterFunc(delay, func() {
				payload.Run()
			})
		}
	}

	d0.Success = true

	return d0, err
}
