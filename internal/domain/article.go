package domain

type Article struct {
	Id      int64
	Title   string
	Content string
	Author  Author
	Status  ArticleStatus
}

type Author struct {
	Id   int64
	name string
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
