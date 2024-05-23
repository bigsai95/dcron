package cronjob

import (
	"github.com/robfig/cron/v3"
)

func (cm *CronManager) StoreJobMapping(jobID string, entryID cron.EntryID) {
	// 儲存任務和 Cron Entry 的對應關係
	cm.jobMap.Store(jobID, entryID)
}

func (cm *CronManager) DeleteJobMapping(jobID string) {
	// 儲存任務和 Cron Entry 的對應關係
	cm.jobMap.Delete(jobID)
}

func (cm *CronManager) LoadJobMapping(jobID string) (cron.EntryID, bool) {
	// 從 Cron 中刪除任務
	entryID, ok := cm.jobMap.Load(jobID)
	if ok {
		return entryID.(cron.EntryID), ok
	}
	return 0, ok
}
