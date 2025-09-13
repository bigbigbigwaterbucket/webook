package domain

import "time"

type Job struct {
	Id       int64
	Name     string
	NextTime time.Time
	CTime    int64
	UTime    int64
	Cancel   func() error
}
