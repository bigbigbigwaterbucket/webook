package domain

import (
	"github.com/robfig/cron/v3"
	"time"
)

type Job struct {
	Id          int64
	Name        string
	Exe         string
	CronDurTime string
	CTime       int64
	UTime       int64
	Cancel      func() error
}

// 解析cron表达式的解析器
var parser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom |
	cron.Month | cron.Dow | cron.Descriptor)

func (j *Job) NextTime() time.Time {
	s, _ := parser.Parse(j.CronDurTime)
	return s.Next(time.Now())
}
