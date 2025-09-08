package domain

type Interactive struct {
	Id         int64
	Biz        string
	BizId      int64
	LikeCnt    int64
	CollectCnt int64
	ReadCnt    int64
	Liked      bool
	Collected  bool
}
