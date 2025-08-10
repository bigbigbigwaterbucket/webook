package domain

import "time"

type User struct {
	Id        int64
	Email     string
	Password  string
	Name      string
	Birthday  string
	Introduce string
	Ctime     time.Time
}
