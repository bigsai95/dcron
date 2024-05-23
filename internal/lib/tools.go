package lib

import (
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

const pattern = `^([A-Z0-9]{2,4})_(.*?)(_(\d+)(_(.*?))?)?$`

var (
	re        = regexp.MustCompile(pattern)
	onceRegex = regexp.MustCompile("@once")
)

func MatchJobName(str string) (v1, v2 string) {
	match := re.FindStringSubmatch(str)
	if match == nil {
		return "", ""
	}

	v1 = match[1]
	v2 = match[4]
	return v1, v2
}

// 是否為一次性的排程任務
func IsMemoOnce(memo string) bool {
	return onceRegex.MatchString(memo)
}

// 算下一個任務的執行時間
func CalculateNextRunTime(now, next time.Time, d time.Duration) time.Time {
	for next.Before(now) || next.Equal(now) {
		next = next.Add(d)
	}
	return next
}

// 是否馬上執行 true:執行 false:不執行
func ShouldExecuteNow(memo string) bool {
	if !IsMemoOnce(memo) {
		return false
	}

	strArray := strings.Split(memo, "@")
	if len(strArray) != 2 {
		return false
	}
	taskTime, err := decimal.NewFromString(strArray[0])
	if err != nil {
		return false
	}

	loc, _ := time.LoadLocation("Asia/Taipei")
	now := time.Now().In(loc)

	t1 := taskTime.IntPart()
	crontime := time.Unix(t1, 0).In(loc)

	d1 := crontime.Sub(now)

	return d1 <= 0
}

// t1 執行時間大於現在時間往前3h,  true:執行 false:不執行
func ShouldExecuteThreeHours(memo string) bool {
	if !IsMemoOnce(memo) {
		return false
	}

	strArray := strings.Split(memo, "@")
	if len(strArray) != 2 {
		return false
	}
	taskTime, err := decimal.NewFromString(strArray[0])
	if err != nil {
		return false
	}

	loc, _ := time.LoadLocation("Asia/Taipei")
	now := time.Now().In(loc)

	threeHoursAgo := now.Add(-3 * time.Hour).Unix()

	t1 := taskTime.IntPart()

	// 檢查 t1 是否大於前3小時 -  3小時之內:true
	isWithinThreeHours := t1 >= threeHoursAgo

	return isWithinThreeHours
}

// 排序資料
func SortMapByValue(maps map[string]string) []string {
	var jobIDs []string
	for _, v := range maps {
		jobIDs = append(jobIDs, v)
	}
	sort.Strings(jobIDs)

	return jobIDs
}
