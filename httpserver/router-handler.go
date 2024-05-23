package httpserver

import (
	"dcron/handler"
	"dcron/internal/cronjob"
	"dcron/internal/ctl"
	"dcron/server"
	"encoding/json"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
)

// @Summary 確認服務連線
// @Produce json
// @Success 200 {object} DataRespSchema{data=Pong} "{"data":{"pong":"pong"}, "errors":[]}"
// @Router  /api/ping [get]
func Ping(c *gin.Context) {
	statusCode := http.StatusBadRequest
	pong := &Pong{}
	pingSuccessful := cronjob.Mgr.GetPingSuccessful()
	if pingSuccessful {
		statusCode = http.StatusOK
		pong.Pong = "pong"
	}
	c.Data(statusCode, jsonContentType, DataResp(pong))
}

// @Summary 查詢排程Group清單
// @Tags 	CronJob Query
// @Produce json
// @Success 200 {object} DataRespSchema{data=[]string} "{"data":["test"],"errors":[]}"
// @Router  /api/group/list [get]
func ListGroup(c *gin.Context) {
	records, _ := ctl.FetchGroupList()

	c.Data(200, jsonContentType, DataResp(records))
}

// @Summary 查詢排程遊戲清單
// @Tags 	CronJob Query
// @Produce json
// @Param 	group_name query string true "group_name"
// @Success 200 {object} DataRespSchema{data=[]string} "{"data":["test"],"errors":[]}"
// @Router  /api/game/list [get]
func ListGame(c *gin.Context) {
	groupName := c.Query("group_name")
	if groupName == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyGroupNameErrMsg))
		return
	}
	records, _ := ctl.FetchGameList(groupName)

	c.Data(200, jsonContentType, DataResp(records))
}

// @Summary 查詢已註冊排程任務清單 By Group
// @Tags 	CronJob Tasks List
// @Produce json
// @Param 	group_name query string true "group_name"
// @Success 200 {object} DataRespSchema{data=[]cronjob.TaskPayload} "{"data":[],"errors":[]}"
// @Router  /api/job/list [get]
func ListJobByGroup(c *gin.Context) {
	groupName := c.Query("group_name")
	if groupName == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyGroupNameErrMsg))
		return
	}
	list, _ := ctl.GetJobsByGroup(groupName)

	c.Data(200, jsonContentType, DataResp(list))
}

// @Summary 查詢已註冊排程任務清單 By Match
// @Tags 	CronJob Tasks List
// @Produce json
// @Param 	group_name query string true "group_name"
// @Param 	match query string true "match by name"
// @Success 200 {object} DataRespSchema{data=[]cronjob.TaskPayload} "{"data":[],"errors":[]}"
// @Router  /api/job/match/list [get]
func ListJobByMatch(c *gin.Context) {
	groupName := c.Query("group_name")
	match := c.Query("match")
	if groupName == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyGroupNameErrMsg))
		return
	}
	if match == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyMatchErrMsg))
		return
	}
	list, _ := ctl.GetJobsByMatch(groupName, match)

	c.Data(200, jsonContentType, DataResp(list))
}

// @Summary 查詢遊戲排程任務清單 By Game
// @Tags 	CronJob Tasks List
// @Produce json
// @Param 	group_name query string true "group_name"
// @Param 	game_type query string true "game_type"
// @Success 200 {object} DataRespSchema{data=[]cronjob.TaskPayload} "{"data":[],"errors":[]}"
// @Router  /api/job/game/list [get]
func ListJobByGame(c *gin.Context) {
	groupName := c.Query("group_name")
	game_type := c.Query("game_type")
	if groupName == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyGroupNameErrMsg))
		return
	}
	if game_type == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyGameTypeErrMsg))
		return
	}
	list, _ := ctl.GetJobsByGame(groupName, game_type)

	c.Data(200, jsonContentType, DataResp(list))
}

// @Summary 查詢單一排程資料
// @Tags 	CronJob Query
// @Produce json
// @Param 	group_name query string true "group_name"
// @Param 	job_id query string true "job_id"
// @Success 200 {object} DataRespSchema{data=cronjob.TaskPayload} "{"data":[],"errors":[]}"
// @Router  /api/job/info [get]
func JobInfo(c *gin.Context) {
	groupName := c.Query("group_name")
	jobID := c.Query("job_id")
	if groupName == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyGroupNameErrMsg))
		return
	}
	if jobID == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyJobIDErrorMsg))
		return
	}
	payload := ctl.GetTaskPayload(groupName, jobID)
	if payload.JobID == "" {
		c.JSON(200, ErrorDataRes("No information found"))
		return
	}

	c.Data(200, jsonContentType, DataResp(payload))
}

// @Summary 註冊排程任務
// @Tags 	CronJob Update
// @Produce json
// @Param 	data body cronjob.TaskPayloadReq true "排程資料"
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router  /api/job/add [post]
func AddJob(c *gin.Context) {
	var p handler.TaskPayloadRequest

	data, _ := c.GetRawData()

	err := json.Unmarshal(data, &p)
	if err != nil {
		c.JSON(200, ErrorResponse(ctl.ParameterErrorMsg))
		return
	}

	handlerServer := &handler.Server{}
	resp, err := handlerServer.AddJob(c, &p)
	if err != nil {
		c.JSON(200, ErrorResponse(err.Error()))

		return
	}

	c.Data(200, jsonContentType, DataSuccess(resp.Success))
}

// @Summary 更新排程任務
// @Tags 	CronJob Update
// @Produce json
// @Param 	data body cronjob.TaskPayloadReq true "排程資料"
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router  /api/job/replace [post]
func ReplaceJob(c *gin.Context) {

	var p handler.TaskPayloadRequest

	data, _ := c.GetRawData()

	err := json.Unmarshal(data, &p)
	if err != nil {
		c.JSON(200, ErrorResponse(ctl.ParameterErrorMsg))
		return
	}

	items, _ := ctl.GetJobsByMatch(p.GroupName, p.Name)
	if len(items) > 0 && items[0].Name == p.Name {
		delJob := items[0]
		// 刪除redis中已註冊的 Job_id
		ctl.DeleteJobFromRedis(delJob.GroupName, delJob.Name, delJob.JobID)

		// 刪除目前排程中的 entry_id
		ctl.EventHandling(cronjob.PubJob{
			Event:     "delete",
			JobID:     delJob.JobID,
			GroupName: delJob.GroupName,
		})
	}
	handlerServer := &handler.Server{}
	resp, err := handlerServer.AddJob(c, &p)
	if err != nil {
		c.JSON(200, ErrorResponse(err.Error()))

		return
	}

	c.Data(200, jsonContentType, DataSuccess(resp.Success))
}

// @Summary 刪除Grop所有註冊任務
// @Tags 	CronJob Update
// @Produce  json
// @Param group path string true "group_name"
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router /api/jobs/delete/{group} [delete]
func DeleteJobs(c *gin.Context) {
	group := c.Param("group")

	jobIDs, _ := ctl.DeleteGroupJobsFromRedis(group)

	for _, jobID := range jobIDs {
		// 刪除目前排程中的 entry_id
		// ctl.RemoveJobFromSchedule(jobID)
		ctl.EventHandling(cronjob.PubJob{
			Event:     "delete",
			JobID:     jobID,
			GroupName: group,
		})
	}

	c.Data(200, jsonContentType, DataSuccess(true))
}

// @Summary 刪除註冊任務
// @Tags 	CronJob Update
// @Produce  json
// @Param group path string true "group_name"
// @Param id path string true "job_id"
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router /api/job/delete/{group}/{id} [delete]
func DeleteJob(c *gin.Context) {
	groupName := c.Param("group")
	jobID := c.Param("id")

	if groupName == "" {
		c.JSON(200, ErrorResponse(ctl.EmptyGroupNameErrMsg))
		return
	}

	payload := ctl.GetTaskPayload(groupName, jobID)
	if payload.JobID == "" {
		c.JSON(200, ErrorResponse(ctl.JobIDGroupNameErrMsg))
		return
	}

	// 刪除redis中已註冊的 Job_id
	ctl.DeleteJobFromRedis(payload.GroupName, payload.Name, payload.JobID)

	// 刪除目前排程中的 entry_id
	// ctl.RemoveJobFromSchedule(payload.JobID)
	ctl.EventHandling(cronjob.PubJob{
		Event:     "delete",
		JobID:     jobID,
		GroupName: groupName,
	})

	c.Data(200, jsonContentType, DataSuccess(true))
}

// @Summary 刪除註冊任務 By Match
// @Tags 	CronJob Update
// @Produce json
// @Param group path string true "group_name"
// @Param match path string true "match"
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router /api/jobs/delete/{group}/{match} [delete]
func DeleteMatchJob(c *gin.Context) {
	groupName := c.Param("group")
	match := c.Param("match")
	if groupName == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyGroupNameErrMsg))
		return
	}
	if match == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyMatchErrMsg))
		return
	}
	exportedTasks, _ := ctl.GetJobsByMatch(groupName, match)

	for _, payload := range exportedTasks {
		// 刪除redis中已註冊的 Job_id
		ctl.DeleteJobFromRedis(payload.GroupName, payload.Name, payload.JobID)

		// 刪除目前排程中的 entry_id
		// ctl.RemoveJobFromSchedule(payload.JobID)
		ctl.EventHandling(cronjob.PubJob{
			Event:     "delete",
			JobID:     payload.JobID,
			GroupName: groupName,
		})
	}

	c.Data(200, jsonContentType, DataSuccess(true))
}

// @Summary 啟用排程
// @Tags 	CronJob Update
// @Produce json
// @Param group path string true "group_name"
// @Param id path string true "job_id"
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router /api/job/active/{group}/{id} [put]
func ActiveJob(c *gin.Context) {
	groupName := c.Param("group")
	jobID := c.Param("id")

	if groupName == "" {
		c.JSON(200, ErrorResponse(ctl.EmptyGroupNameErrMsg))
		return
	}

	payload := ctl.GetTaskPayload(groupName, jobID)
	if payload.JobID == "" {
		c.JSON(200, ErrorResponse(ctl.JobIDGroupNameErrMsg))
		return
	}

	ctl.EventHandling(cronjob.PubJob{
		Event:     "active",
		JobID:     jobID,
		GroupName: groupName,
	})

	c.Data(200, jsonContentType, DataSuccess(true))
}

// @Summary 暫停排程
// @Tags 	CronJob Update
// @Produce json
// @Param group path string true "group_name"
// @Param id path string true "job_id"
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router /api/job/pause/{group}/{id} [put]
func PauseJob(c *gin.Context) {
	groupName := c.Param("group")
	jobID := c.Param("id")

	if groupName == "" {
		c.JSON(200, ErrorResponse(ctl.EmptyGroupNameErrMsg))
		return
	}

	ctl.EventHandling(cronjob.PubJob{
		Event:     "pause",
		JobID:     jobID,
		GroupName: groupName,
	})

	c.Data(200, jsonContentType, DataSuccess(true))
}

// @Summary 匯入任務
// @Description Import a file using a form-data request
// @Tags 	CronJob Import/export
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to import"
// @Success 200 {string} json "{"message": "imported successfully"}"
// @Router /api/jobs/import [post]
func ImportHandler(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read file"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read file"})
		return
	}

	var jobs []cronjob.TaskPayload
	err = json.Unmarshal(data, &jobs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse JSON"})
		return
	}

	for _, payload := range jobs {
		exists, err := ctl.AcquireLock(payload.GroupName, payload.Name, 0)
		if !exists || err != nil {
			server.GetServerInstance().GetLogger().WithFields(map[string]interface{}{
				"func":       "job_import",
				"group_name": payload.GroupName,
				"job_name":   payload.Name,
				"cron_type":  payload.Type,
				"job_id":     payload.JobID,
			}).Errorf("job import error: %v", err)
			continue
		}

		err = ctl.SetTaskPayload(payload, 0)
		if err != nil {
			ctl.ReleaseLock(payload.GroupName, payload.Name)

			c.JSON(200, ErrorResponse(err.Error()))
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "imported successfully"})
}

// @Summary 匯出任務 By Group
// @Description Export all jobs as a JSON file
// @Tags 	CronJob Import/export
// @Produce json
// @Param group path string true "group_name"
// @Success 200 {file} application/json
// @Router /api/jobs/export/{group} [post]
func ExportGroupHandler(c *gin.Context) {
	groupName := c.Param("group")

	exportedTasks, _ := ctl.GetJobsByGroup(groupName)

	ExportCronJob(c, exportedTasks)
}

// @Summary 匯出任務 By Match
// @Description Export all jobs as a JSON file
// @Tags 	CronJob Import/export
// @Produce json
// @Param group path string true "group_name"
// @Param match path string true "match"
// @Success 200 {file} application/json
// @Router /api/jobs/export/{group}/{match} [post]
func ExportMatchHandler(c *gin.Context) {
	groupName := c.Param("group")
	match := c.Param("match")
	if groupName == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyGroupNameErrMsg))
		return
	}
	if match == "" {
		c.JSON(200, ErrorDataRes(ctl.EmptyMatchErrMsg))
		return
	}
	exportedTasks, _ := ctl.GetJobsByMatch(groupName, match)

	ExportCronJob(c, exportedTasks)
}

// @Summary 匯出任務 By All
// @Description Export all jobs as a JSON file
// @Tags 	CronJob Import/export
// @Produce json
// @Success 200 {file} application/json
// @Router /api/jobs/export [post]
func ExportAllHandler(c *gin.Context) {
	exportedTasks, _ := ctl.GetJobsByAll()

	ExportCronJob(c, exportedTasks)
}

func ExportCronJob(c *gin.Context, exportedTasks interface{}) {
	exportedJSON, err := json.Marshal(exportedTasks)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse("Failed to export jobs"))
		return
	}

	// 下載匯出的JSON文件
	fileName := "tasks.json"
	err = os.WriteFile(fileName, exportedJSON, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to export jobs"})
		return
	}

	defer os.Remove(fileName)

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("Content-Type", "application/json")
	c.File(fileName)
}

// @Summary 查詢排程註冊數和執行數量
// @Tags 	CronJob Query
// @Produce json
// @Success 200 {object} interface{} "{"data":{"running":0,"register":0},"errors":[]}"
// @Router  /api/job/query [get]
func QueryHandler(c *gin.Context) {
	entries := cronjob.Mgr.Entries()
	tasks, _ := ctl.GetJobRecords()

	type jobCase struct {
		Running  int `json:"running"`
		Register int `json:"register"`
	}

	test := jobCase{
		Running:  len(entries),
		Register: len(tasks),
	}

	c.Data(200, jsonContentType, DataResp(test))
}

// @Summary 查詢排程狀態
// @Tags 	CronJob Query
// @Produce json
// @Param id path string true "job_id"
// @Router /api/job/query/{id} [get]
func QueryJob(c *gin.Context) {
	// group := c.Param("group")
	jobID := c.Param("id")

	entryID, jobMapOK := cronjob.Mgr.LoadJobMapping(jobID)
	dataEntry := cronjob.Mgr.Entry(entryID)

	data := struct {
		JobMapOK    bool         `json:"job_mapping"`
		MapEntryID  cron.EntryID `json:"map_entry_id"`
		CronEntryID cron.EntryID `json:"cron_entry_id"`
	}{
		JobMapOK:    jobMapOK,
		MapEntryID:  entryID,
		CronEntryID: dataEntry.ID,
	}

	c.Data(200, jsonContentType, DataResp(data))
}

// @Summary 排程功能啟動
// @Tags 	Service
// @Produce json
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router  /api/service/cronjob/start [post]
func StartCronJob(c *gin.Context) {

	// cronjob.Mgr.Start()
	ctl.EventHandling(cronjob.PubJob{
		Event: "start",
	})

	c.Data(200, jsonContentType, DataSuccess(true))
}

// @Summary 排程功能停止
// @Tags 	Service
// @Produce json
// @Success 200 {object} SuccessRes "{"success":true,"errors":[]}"
// @Router  /api/service/cronjob/stop [post]
func StopCronJob(c *gin.Context) {

	// cronjob.Mgr.Stop()
	ctl.EventHandling(cronjob.PubJob{
		Event: "stop",
	})

	c.Data(200, jsonContentType, DataSuccess(true))
}
