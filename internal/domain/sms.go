package domain

type Sms struct {
	Id        int64
	TplId     string
	Args      []string
	Phone     string
	RetryTime int64
}
