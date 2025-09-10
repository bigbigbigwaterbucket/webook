package service

import (
	"github.com/ecodeclub/ekit/queue"
	"github.com/ecodeclub/ekit/slice"
	"golang.org/x/net/context"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository"
	"learning_go/webook/internal/repository/article"
	"math"
	"time"
)

//go:generate mockgen -source=D:/go_project/learning_go/webook/internal/service/ranking_dispatch.go -package=svcmocks -destination=D:/go_project/learning_go/webook/internal/service/mocks/ranking_mock.go
type RankingService interface {
	TopN(ctx context.Context) error
}

// 该实现通过分批次从articles表中读取数据，不断替换topN来实现获取全表topN
type RankingServiceI struct {
	articleRepo article.ArticleRepository
	interRepo   repository.InteractiveRepository
	repo        repository.RankingRepository
	rankScore   func(utime time.Time, likeCnt int64) float64
	topNum      int
	batchSize   int
}

func NewRankingServiceI(articleRepo article.ArticleRepository, interRepo repository.InteractiveRepository, repo repository.RankingRepository) *RankingServiceI {
	return &RankingServiceI{articleRepo: articleRepo, interRepo: interRepo, repo: repo,
		topNum: 100, batchSize: 100, rankScore: func(utime time.Time, likeCnt int64) float64 {
			dur := time.Since(utime).Seconds()
			return float64(likeCnt-1) / math.Pow(float64(dur+2), 1.5)
		}}
}

func (r *RankingServiceI) TopN(ctx context.Context) error {
	arts, err := r.topN(ctx)
	if err != nil {
		return err
	}
	return r.repo.ReplaceTopN(ctx, arts)
}

func (r *RankingServiceI) topN(ctx context.Context) ([]domain.Article, error) {
	type Score struct {
		art   domain.Article
		score float64
	}
	//优先权队列
	// Comparator 用于比较两个对象的大小 src < dst, 返回-1，src = dst, 返回0，src > dst, 返回1
	priorityQueue := queue.NewPriorityQueue[Score](r.topNum, func(src Score, dst Score) int {
		if src.score < dst.score {
			return -1
		} else if src.score == dst.score {
			return 0
		} else {
			return 1
		}
	})
	offset := 0
	for {
		arts, err := r.articleRepo.GetRankingList(ctx, offset, r.batchSize)
		if err != nil {
			return []domain.Article{}, err
		}
		ids := slice.Map[domain.Article, int64](arts, func(idx int, src domain.Article) int64 {
			return src.Id
		})
		inters, err := r.interRepo.GetByIds(ctx, ids)
		if err != nil {
			return []domain.Article{}, err
		}
		for _, art := range arts {
			inter := inters[art.Id]
			artScore := r.rankScore(time.UnixMilli(art.UTime), inter.LikeCnt)
			err := priorityQueue.Enqueue(Score{art: art, score: artScore})
			if err == queue.ErrOutOfCapacity {
				score, _ := priorityQueue.Dequeue()
				if score.score < artScore {
					//热度不如新来的
					_ = priorityQueue.Enqueue(Score{art: art, score: artScore})
				} else {
					_ = priorityQueue.Enqueue(score)
				}
			}
		}
		//别忘offset++
		offset += len(arts)
		//长度不够凑够一批，则说明到头了
		if len(arts) < r.batchSize {
			break
		}
	}
	res := make([]domain.Article, 0, r.topNum)
	for i := r.topNum - 1; i >= 0; i++ {
		score, err := priorityQueue.Dequeue()
		if err != nil {
			//取完了，不够n
			return res, nil
		}
		res = append(res, score.art)
	}
	return res, nil
}
