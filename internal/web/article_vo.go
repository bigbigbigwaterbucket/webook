package web

// 用于给前端展示的结构体
type ArticleVO struct {
	Id int64 `json:"id"`
	//这个status可以交给前端处理成字符串，也可以后端处理
	//如果是手机app这种涉及发版的，后端处理，如果涉及国际化，也是后端处理
	Status     uint8  `json:"Status"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	Abstract   string `json:"abstract"`
	Author     string `json:"author"`
	Ctime      int64  `json:"ctime"`
	Utime      int64  `json:"utime"`
	LikeCnt    int64  `json:"likeCnt"`
	ReadCnt    int64  `json:"readCnt"`
	CollectCnt int64  `json:"collectCnt"`
	Liked      bool   `json:"liked"`
	Collected  bool   `json:"collected"`
}

type ListReq struct {
	Offset int64 `gorm:"offset"`
	Limit  int64 `gorm:"limit"`
}

type ArticleReq struct {
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Content string `json:"content"`
}

type LikeReq struct {
	Id int64 `json:"id"`
	//标记是取消点赞还是点赞
	Like bool `json:"like"`
}

type CollectReq struct {
	Id int64 `json:"id"`
	//标记收藏在哪个收藏夹里
	Cid int64 `json:"cid"`
}

type TopReq struct {
	TopNum int64 `json:"topNums"`
}
