package domain

type Comment struct {
	Id              int64 `json:"id"`
	Uid             int64
	Biz             string
	BizId           int64
	Content         string
	RootId          int64
	PID             int64
	ParentComment   *Comment
	ChildrenComment []Comment
	CTime           int64
	UTime           int64
}
