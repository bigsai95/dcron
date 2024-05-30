package ctl

import (
	"dcron/internal/cronjob"
	"dcron/internal/lib"
	"dcron/internal/redisCacher"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func AcquireLock(groupName, name string, ttl int64) (bool, error) {
	key := fmt.Sprintf("CK_%s_%s", groupName, name)
	return redisCacher.Conn.SetNX(key, 1, ttl)
}

func ReleaseLock(groupName, name string) {
	key := fmt.Sprintf("CK_%s_%s", groupName, name)
	redisCacher.Conn.Del(key)
}

// 取出所有註冊的group
func FetchGroupList() ([]string, error) {
	records, err := redisCacher.Conn.Scan("TEAM_*")
	if err != nil {
		return nil, err
	}

	ret := make([]string, 0)
	for _, v := range records {
		data := strings.Split(v, "TEAM_")
		if len(data) > 0 {
			ret = append(ret, data[1])
		}
	}
	sort.Strings(ret)

	return ret, nil
}

/*
 * 註冊 group
 * redis資料格式
 *
 * key: TEAM_test
 * ["pojob_014"] = job01
 * ["pojob_015"] = job02
 */
func SetJobGroup(groupName, name, jobID string) error {
	fields := make(map[string]interface{})
	fields[name] = jobID
	// 各個group_name底下的JobID
	key := fmt.Sprintf("TEAM_%s", groupName)
	return redisCacher.Conn.HSet(key, fields, 0)
}

/*
 * 設定排程資料
 * redis資料格式
 *
 * key: _test_job01
 * ["job_id"] = job01
 * ["request_url"] = http://127.0.0.1/api/ping
 * ["interval_pattern"] = 0 * * * * *
 */
func SetTaskPayload(payload cronjob.TaskPayload, ttl int64) error {

	if err := SetJobGroup(payload.GroupName, payload.Name, payload.JobID); err != nil {
		return err
	}

	params := map[string]interface{}{
		"job_id":           payload.JobID,
		"group_name":       payload.GroupName,
		"name":             payload.Name,
		"exec_right_now":   payload.ExecRightNow,
		"request_url":      payload.RequestUrl,
		"retry":            payload.Retry,
		"interval_pattern": payload.IntervalPattern,
		"type":             payload.Type,
		"status":           payload.Status,
		"nsq_topic":        payload.NsqTopic,
		"nsq_message":      payload.NsqMessage,
		"register":         payload.Register.Format(time.RFC3339),
		"prev":             payload.Prev.Format(time.RFC3339),
		"next":             payload.Next.Format(time.RFC3339),
		"memo":             payload.Memo,
	}

	// 任務註冊表
	key := fmt.Sprintf("TASK_%s_%s", payload.GroupName, payload.JobID)

	return redisCacher.Conn.HSet(key, params, ttl)
}

// 將 map[string]string 映射到 cronjob.TaskPayload 結構中
func MapTaskPayload(data map[string]string) cronjob.TaskPayload {
	var prev, next time.Time

	execRightNow, err := strconv.ParseBool(data["exec_right_now"])
	if err != nil {
		execRightNow = false
	}

	retry, err := strconv.ParseBool(data["retry"])
	if err != nil {
		retry = false
	}

	status, err := strconv.Atoi(data["status"])
	if err != nil {
		status = 0
	}

	register, err := time.Parse(time.RFC3339, data["register"])
	if err != nil {
		register = time.Time{}
	}

	prev, err = time.Parse(time.RFC3339, data["prev"])
	if err != nil {
		prev = time.Time{}
	}

	key := fmt.Sprintf("TIME_%s_%s", data["group_name"], data["job_id"])
	timeMaps, err := redisCacher.Conn.HGetAll(key)
	if err == nil {
		if value, ok := timeMaps["prev"]; ok {
			prev, err = time.Parse(time.RFC3339, value)
			if err != nil {
				prev = time.Time{}
			}
		}
		if value, ok := timeMaps["next"]; ok {
			next, err = time.Parse(time.RFC3339, value)
			if err != nil {
				next = time.Time{}
			}
		}
	}

	return cronjob.TaskPayload{
		JobID:           data["job_id"],
		GroupName:       data["group_name"],
		Name:            data["name"],
		ExecRightNow:    execRightNow,
		RequestUrl:      data["request_url"],
		Retry:           retry,
		IntervalPattern: data["interval_pattern"],
		Type:            data["type"],
		Status:          status,
		NsqTopic:        data["nsq_topic"],
		NsqMessage:      data["nsq_message"],
		Register:        register,
		Prev:            prev,
		Next:            next,
		Memo:            data["memo"],
	}
}

// 取得所有註冊清單名字,"TASK_*" 鍵進行掃描獲取
func GetJobRecords() ([]string, error) {
	records, err := redisCacher.Conn.Scan("TASK_*")
	if err != nil {
		return records, err
	}
	sort.Strings(records)

	return records, nil
}

// 取得所有註冊資料,獲取 "TASK_*"
func GetJobsByAll() ([]cronjob.TaskPayload, error) {
	ret := make([]cronjob.TaskPayload, 0)
	records, err := GetJobRecords()
	if err != nil {
		return ret, err
	}
	for _, v := range records {
		data, err := redisCacher.Conn.HGetAll(v)
		if err == nil {
			TaskPayload := MapTaskPayload(data)
			if TaskPayload.JobID != "" {
				ret = append(ret, TaskPayload)
			}
		}
	}
	return ret, nil
}

// 取得"符合"註冊資料 - By Game
func GetJobsByGame(groupName, gameType string) ([]cronjob.TaskPayload, error) {
	TaskPayloads := make([]cronjob.TaskPayload, 0)

	key := fmt.Sprintf("TEAM_%s", groupName)
	groupJobs, err := redisCacher.Conn.HGetAll(key)
	if err != nil {
		return nil, err
	}

	pattern := fmt.Sprintf(`^(%s)_(.*?)$`, gameType)
	re := regexp.MustCompile(pattern)

	var jobIDs []string
	for k, v := range groupJobs {
		match := re.FindStringSubmatch(k)
		if len(match) > 0 {
			jobIDs = append(jobIDs, v)
		}
	}

	if len(jobIDs) > 0 {
		sort.Strings(jobIDs)
		TaskPayloads = GetTaskPayloads(groupName, jobIDs)
	}

	return TaskPayloads, nil
}

// 取得"符合"註冊資料 - By match
func GetJobsByMatch(groupName, match string) ([]cronjob.TaskPayload, error) {
	TaskPayloads := make([]cronjob.TaskPayload, 0)

	key := fmt.Sprintf("TEAM_%s", groupName)
	groupJobs, err := redisCacher.Conn.HGetAll(key)
	if err != nil {
		return nil, err
	}

	var jobIDs []string
	for k, v := range groupJobs {
		if strings.Contains(k, match) {
			jobIDs = append(jobIDs, v)
		}
	}

	if len(jobIDs) > 0 {
		sort.Strings(jobIDs)
		TaskPayloads = GetTaskPayloads(groupName, jobIDs)
	}

	return TaskPayloads, nil
}

// 取得"group"註冊資料
func GetJobsByGroup(groupName string) ([]cronjob.TaskPayload, error) {
	TaskPayloads := make([]cronjob.TaskPayload, 0)

	key := fmt.Sprintf("TEAM_%s", groupName)
	groupJobs, err := redisCacher.Conn.HGetAll(key)
	if err != nil {
		return TaskPayloads, err
	}

	if len(groupJobs) > 0 {
		jobIDs := lib.SortMapByValue(groupJobs)
		TaskPayloads = GetTaskPayloads(groupName, jobIDs)
	}

	return TaskPayloads, nil
}

// 取得定時任務資訊
func GetTaskPayloads(groupName string, jobIDs []string) []cronjob.TaskPayload {
	var ret []cronjob.TaskPayload

	for _, jobID := range jobIDs {
		var builder strings.Builder
		builder.WriteString("TASK_")
		builder.WriteString(groupName)
		builder.WriteString("_")
		builder.WriteString(jobID)
		v := builder.String()

		data, err := redisCacher.Conn.HGetAll(v)
		if err == nil {
			job := MapTaskPayload(data)

			entryID, ok := cronjob.Mgr.LoadJobMapping(job.JobID)
			if ok {
				dataEntry := cronjob.Mgr.Entry(entryID)
				job.Next = dataEntry.Next
			}
			if job.JobID != "" {
				ret = append(ret, job)
			}
		}
	}

	return ret
}

// 取得定時任務資訊
func GetTaskPayload(groupName, jobID string) cronjob.TaskPayload {
	var payload cronjob.TaskPayload

	key := fmt.Sprintf("TASK_%s_%s", groupName, jobID)
	data, err := redisCacher.Conn.HGetAll(key)
	if err != nil || len(data) == 0 {
		return payload
	}

	payload = MapTaskPayload(data)

	entryID, ok := cronjob.Mgr.LoadJobMapping(payload.JobID)
	if ok {
		dataEntry := cronjob.Mgr.Entry(entryID)
		payload.Next = dataEntry.Next
	}

	return payload
}

// 更新定時任務狀態
func UpdateJobStatus(groupName, JobID string, status int) error {
	key := fmt.Sprintf("TASK_%s_%s", groupName, JobID)
	return redisCacher.Conn.HSet(key, map[string]interface{}{"status": status}, 0)
}

// 刪除相關的單一定時任務
func DeleteJobFromRedis(groupName, name, JobID string) {
	keys := []string{
		fmt.Sprintf("TASK_%s_%s", groupName, JobID),
		fmt.Sprintf("CK_%s_%s", groupName, name),
		fmt.Sprintf("TIME_%s_%s", groupName, JobID),
	}
	for _, key := range keys {
		redisCacher.Conn.Del(key)
	}

	key := fmt.Sprintf("TEAM_%s", groupName)
	redisCacher.Conn.HDel(key, name)
}

// 刪除相關的定時任務
func DeleteGroupJobsFromRedis(groupName string) ([]string, error) {

	var (
		jobIDs     []string
		deleteKeys []string
	)
	jobIDs = make([]string, 0)
	key := fmt.Sprintf("TEAM_%s", groupName)

	groupJobs, err := redisCacher.Conn.HGetAll(key)
	if err != nil {
		return jobIDs, err
	}

	for jobName, jobID := range groupJobs {
		delKey := fmt.Sprintf("TASK_%s_%s", groupName, jobID)
		deleteKeys = append(deleteKeys, delKey)

		delKey = fmt.Sprintf("CK_%s_%s", groupName, jobName)
		deleteKeys = append(deleteKeys, delKey)

		delKey = fmt.Sprintf("TIME_%s_%s", groupName, jobID)
		deleteKeys = append(deleteKeys, delKey)

		jobIDs = append(jobIDs, jobID)
	}

	delKey := fmt.Sprintf("TEAM_%s", groupName)
	deleteKeys = append(deleteKeys, delKey)

	err = redisCacher.Conn.DelKeys(deleteKeys)
	if err != nil {
		return jobIDs, err
	}

	return jobIDs, nil
}
