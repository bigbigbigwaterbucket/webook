package domain

type Article struct {
	Id      int64
	Title   string
	Content string
	Author  Author
	Status  ArticleStatus
	CTime   int64
	UTime   int64
}

func (a Article) Abstract() string {
	//生成摘要，取前面几句
	//要考虑中文问题
	//下面是按字节数截取的，截取中文会有问题
	//return a.Content[:1024]
	cs := []rune(a.Content)
	if len(cs) < 100 {
		return a.Content
	}
	return string(cs[:100])
}

type Author struct {
	Id   int64
	Name string
}

type ArticleStatus uint8

const (
	ArticleStatusUnknown ArticleStatus = iota
	ArticleStatusUnPublished
	ArticleStatusPublished
	ArticleStatusPrivate
)

func (as ArticleStatus) ToUnt8() uint8 {
	return uint8(as)
}

func (as ArticleStatus) String() string {
	switch as {
	case ArticleStatusPublished:
		return "published"
	case ArticleStatusUnPublished:
		return "unpublished"
	case ArticleStatusPrivate:
		return "private"
	default:
		return "unknown"
	}
}
