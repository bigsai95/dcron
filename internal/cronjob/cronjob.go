package cronjob

import (
	"context"
	"dcron/internal/goworker"
	"dcron/internal/lib"
	"dcron/server"
	"os"
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
	PingSuccessful bool
	c              *cron.Cron
	mutex          sync.Mutex
	cronParser     cron.Parser = cron.NewParser(cron.SecondOptional | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	logger         *logrus.Logger
	Mgr            *CronManager
)

type CronManager struct {
	jobMap  sync.Map
	running bool
}

func ConfigInit() {
	glogger := server.GetServerInstance().GetLogger()
	if glogger == nil {
		glogger = logrus.New()

		glogger.SetFormatter(&logrus.JSONFormatter{})

		glogger.SetOutput(os.Stdout)

		glogger.SetLevel(logrus.DebugLevel)
	}

	logger = glogger
	Mgr = &CronManager{}
}

func StartInit(ctx context.Context, testing bool) {
	loc, _ := time.LoadLocation("Asia/Taipei")

	mutex.Lock()
	c = cron.New(cron.WithSeconds(), cron.WithLocation(loc), cron.WithParser(cronParser))
	mutex.Unlock()

	Mgr.Start()

	logger.Info("Cronjob Start...")

	if !testing {
		select {
		case <-ctx.Done():
			Mgr.SetPingSuccessful(false)

			time.Sleep(200 * time.Microsecond)

			Mgr.Stop()
			logger.Info("Cronjob End...")
			return
		}
	}
}

func (cm *CronManager) SetPingSuccessful(success bool) {
	mutex.Lock()
	defer mutex.Unlock()
	PingSuccessful = success
}

func (cm *CronManager) GetPingSuccessful() bool {
	mutex.Lock()
	defer mutex.Unlock()
	return PingSuccessful
}

func (cm *CronManager) ImportJobs(jobs []TaskPayload) {
	var wg sync.WaitGroup
	loc, _ := time.LoadLocation("Asia/Taipei")
	now := time.Now().In(loc)

	worker := server.GetServerInstance().GetWorker()

	for _, v := range jobs {
		job := v

		if job.Status == 1 && job.JobID != "" {
			wg.Add(1)
			worker.JobQueue(goworker.DoJob(func(i []interface{}) {
				defer wg.Done()
				if lib.ShouldExecuteNow(job.Memo) {
					job.Run()
					return
				}

				if strings.HasPrefix(job.IntervalPattern, every) {
					duration, _ := time.ParseDuration(job.IntervalPattern[len(every):])
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
			}))
		}
	}
	wg.Wait()
	logger.Info("ImportJobs OK")

}

func (cm *CronManager) ImportAddJobs(payload TaskPayload) {
	mutex.Lock()
	defer mutex.Unlock()
	entryID, err := c.AddJob(payload.IntervalPattern, &payload)
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

	mutex.Lock()
	_, err := cronParser.Parse(cronSchedule)
	mutex.Unlock()
	if err != nil {
		return "", err
	}
	return cronSchedule, nil
}

func (cm *CronManager) GetRunning() bool {
	return cm.running
}

func (cm *CronManager) Start() {
	mutex.Lock()
	defer mutex.Unlock()
	if cm.running {
		return
	}
	cm.running = true
	c.Start()
}

func (cm *CronManager) Stop() {
	mutex.Lock()
	defer mutex.Unlock()
	cm.running = false
	c.Stop()
}

func (cm *CronManager) Remove(entryID cron.EntryID) {
	c.Remove(entryID)
}

func (cm *CronManager) Location() string {
	return c.Location().String()
}

func (cm *CronManager) AddFunc(spec string, cmd func()) (cron.EntryID, error) {
	return c.AddFunc(spec, cmd)
}

func (cm *CronManager) AddJob(spec string, cmd cron.Job) (cron.EntryID, error) {
	return c.AddJob(spec, cmd)
}

func (cm *CronManager) Entry(entryID cron.EntryID) cron.Entry {
	return c.Entry(entryID)
}

func (cm *CronManager) Entries() []cron.Entry {
	return c.Entries()
}
