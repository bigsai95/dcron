package cronjob

import (
	"context"
	"dcron/internal/goworker"
	"dcron/internal/lib"
	"dcron/server"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

const every = "@every "

var (
	logger             *logrus.Logger
	Mgr                *CronManager
	defaultLocation, _ = time.LoadLocation("Asia/Taipei")
)

type CronManager struct {
	jobMap         sync.Map
	running        bool
	cron           *cron.Cron
	mutex          sync.Mutex
	pingSuccessful bool
	cronParser     cron.Parser
}

func ConfigInit() {
	logger = server.GetServerInstance().GetLogger()

	Mgr = NewCronManager()
}

func NewCronManager() *CronManager {
	return &CronManager{
		cronParser: cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor),
	}
}

func (cm *CronManager) StartInit(ctx context.Context, testing bool) {
	cm.mutex.Lock()
	cm.cron = cron.New(cron.WithSeconds(), cron.WithLocation(defaultLocation), cron.WithParser(cm.cronParser))
	cm.mutex.Unlock()

	cm.Start()

	logger.Info("Cronjob Start...")

	if !testing {
		select {
		case <-ctx.Done():
			cm.SetPingSuccessful(false)

			time.Sleep(200 * time.Microsecond)

			cm.Stop()
			logger.Info("Cronjob End...")
			return
		}
	}
}

func (cm *CronManager) SetPingSuccessful(success bool) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.pingSuccessful = success
}

func (cm *CronManager) GetPingSuccessful() bool {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	return cm.pingSuccessful
}

func (cm *CronManager) ImportJobs(jobs []TaskPayload) {
	var wg sync.WaitGroup
	now := time.Now().In(defaultLocation)

	worker := server.GetServerInstance().GetWorker()

	for _, job := range jobs {
		if job.Status == 1 && job.JobID != "" {
			wg.Add(1)
			jobCopy := job
			worker.JobQueue(goworker.DoJob(func(_ []interface{}) {
				defer wg.Done()
				cm.processJob(jobCopy, now)
			}))
		}
	}
	wg.Wait()
	logger.Info("ImportJobs OK")
}

func (cm *CronManager) processJob(job TaskPayload, now time.Time) {
	if lib.ShouldExecuteNow(job.Memo) {
		job.Run()
		return
	}

	if strings.HasPrefix(job.IntervalPattern, every) {
		duration, err := time.ParseDuration(job.IntervalPattern[len(every):])
		if err != nil {
			logger.WithField("job_id", job.JobID).Errorf("Failed to parse duration: %v", err)
			return
		}

		timeNext := lib.CalculateNextRunTime(now, job.Next, duration)
		delay := timeNext.Sub(now)
		if delay > 0 {
			time.AfterFunc(delay, func() {
				cm.ImportAddJobs(job)
			})
		}
	} else {
		cm.ImportAddJobs(job)
	}
}

func (cm *CronManager) ImportAddJobs(payload TaskPayload) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	entryID, err := cm.cron.AddJob(payload.IntervalPattern, &payload)
	if err == nil {
		// 儲存任務和 Cron Entry 的對應關係
		cm.StoreJobMapping(payload.JobID, entryID)
	}
}

func (cm *CronManager) FormatSchedule(cronSchedule string) string {
	loc, _ := time.LoadLocation("Asia/Taipei")

	n, err := decimal.NewFromString(cronSchedule)
	if err != nil {
		return cronSchedule
	}

	// 解析时间戳为时间
	t := time.Unix(n.IntPart(), 0).In(loc)

	// 将时间转换为cron格式
	cronParts := []string{
		strconv.Itoa(t.Second()),
		strconv.Itoa(t.Minute()),
		strconv.Itoa(t.Hour()),
		strconv.Itoa(t.Day()),
		strconv.Itoa(int(t.Month())),
		"*",
	}
	cronSchedule = strings.Join(cronParts, " ")

	return cronSchedule
}

func (cm *CronManager) Parse(cronSchedule string) (string, error) {
	cronSchedule = cm.FormatSchedule(cronSchedule)

	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	_, err := cm.cronParser.Parse(cronSchedule)
	if err != nil {
		return "", err
	}
	return cronSchedule, nil
}

func (cm *CronManager) GetRunning() bool {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	return cm.running
}

func (cm *CronManager) Start() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	if cm.running {
		return
	}
	cm.running = true
	cm.cron.Start()
}

func (cm *CronManager) Stop() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	if !cm.running {
		return
	}
	cm.running = false
	cm.cron.Stop()
}

func (cm *CronManager) Remove(entryID cron.EntryID) {
	cm.cron.Remove(entryID)
}

func (cm *CronManager) Location() string {
	return cm.cron.Location().String()
}

func (cm *CronManager) AddFunc(spec string, cmd func()) (cron.EntryID, error) {
	return cm.cron.AddFunc(spec, cmd)
}

func (cm *CronManager) AddJob(spec string, cmd cron.Job) (cron.EntryID, error) {
	return cm.cron.AddJob(spec, cmd)
}

func (cm *CronManager) Entry(entryID cron.EntryID) cron.Entry {
	return cm.cron.Entry(entryID)
}

func (cm *CronManager) Entries() []cron.Entry {
	return cm.cron.Entries()
}
