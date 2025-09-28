package migrator

type Entity interface {
	ID() int64
	Utime() int64
	CompareTo(t Entity) bool
}
