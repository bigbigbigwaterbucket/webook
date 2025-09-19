package events

type ReadEvent struct {
	Aid int64 `json:"aid"`
	Uid int64 `json:"uid"`
}
