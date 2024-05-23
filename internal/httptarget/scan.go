package httptarget

import (
	"context"
	"time"
)

type entryScan struct {
	TryMaxCount int
	Url         string
	Target      ITarget
}

type EntryScanBody struct {
	Body  string
	Code  int
	Count int
	Err   error
}

func NewEntryScan(requestUrl string, maxCount int) *entryScan {
	target := &Service{}

	if maxCount < 1 {
		maxCount = 1
	}

	return &entryScan{
		Url:         requestUrl,
		Target:      target,
		TryMaxCount: maxCount,
	}
}

func (m *entryScan) Scan(ctx context.Context) *EntryScanBody {
	ret := &EntryScanBody{}
	err := m.Target.NewTarget(ctx, m.Url)
	if err != nil {
		return ret
	}

	for n := 1; n <= m.TryMaxCount; n++ {
		res, err := m.Target.GetResponse(ctx)

		ret.Count = n
		if err != nil {
			ret.Err = err
		} else {
			ret.Code = res.Status()
		}

		if ret.Code == 200 {
			ret.Body = string(res.Body())
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return ret
}
