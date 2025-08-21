package domain

import "time"

type User struct {
	Id        int64
	Email     string
	Phone     string
	Password  string
	Name      string
	Birthday  string
	Introduce string
	Ctime     time.Time

	// 不要使用组合，因为你将来可能还有 DingDingInfo 之类的
	// 如果使用组合，可能产生字段冲突
	WechatInfo WechatInfo
}

type WechatInfo struct {
	OpenId  string
	UnionId string
}
