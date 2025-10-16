package cache

import (
	"context"
	"errors"
	"learning_go/webook/article/domain"
	"time"

	"github.com/ecodeclub/ekit/syncx/atomicx"
)

type LocalRankingCache struct {
	//local内存不好抽象，基本需要与业务强绑定了，这里就服务于文章
	topNArts *atomicx.Value[[]domain.Article]
	ddl      *atomicx.Value[time.Time]
	dur      time.Duration
}

func NewLocalRankingCache(
	dur time.Duration) *LocalRankingCache {
	return &LocalRankingCache{topNArts: atomicx.NewValue[[]domain.Article](), ddl: atomicx.NewValueOf(time.Now()), dur: dur} // 永不过期，或者非常长，或者对齐到 redis 的过期时间，都行
}

func (l *LocalRankingCache) Set(ctx context.Context, arts []domain.Article) error {
	//虽然用了原子化的切片，但还是有很小的并发问题，例如一个goroutine刚存完topN，结果来另一个goroutine迅速存完topN和ddl，你再存ddl就会覆盖
	//但是这里问题不大，因为两个线程存的结果肯定是差不多的
	l.topNArts.Store(arts) //必须传指针，否则没法修改topNArts
	l.ddl.Store(time.Now().Add(l.dur))
	return nil
}

func (l *LocalRankingCache) Get(ctx context.Context) ([]domain.Article, error) {
	now := time.Now()
	res := l.topNArts.Load()
	if l.ddl.Load().After(now) && len(res) > 0 {
		//数据没过期且已经初始化过了
		return res, nil
	}
	return nil, errors.New("本地缓存未命中")
}

func (l *LocalRankingCache) ForceGet(ctx context.Context) ([]domain.Article, error) {
	res := l.topNArts.Load()
	return res, nil
}
