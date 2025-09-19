package service

import (
	"context"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	domain2 "learning_go/webook/interactive/domain"
	repository2 "learning_go/webook/interactive/repository"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository"
	"learning_go/webook/internal/repository/article"
	repomocks2 "learning_go/webook/internal/repository/article/mocks"
	repomocks "learning_go/webook/internal/repository/mocks"
	"testing"
	"time"
)

// 复杂逻辑的单测是有必要的
func TestTopN(t *testing.T) {
	testCases := []struct {
		name    string
		mock    func(ctrl *gomock.Controller) (article.ArticleRepository, repository2.InteractiveRepository)
		wantRes []domain.Article
		wantErr error
	}{
		{
			name: "正常返回排行数据",
			mock: func(ctrl *gomock.Controller) (article.ArticleRepository, repository2.InteractiveRepository) {
				now := time.Now()
				artRepo := repomocks2.NewMockArticleRepository(ctrl)
				interRepo := repomocks.NewMockInteractiveRepository(ctrl)
				artRepo.EXPECT().GetRankingList(gomock.Any(), 0, 100).
					Return([]domain.Article{
						{Id: 1, UTime: now.UnixMilli()},
						{Id: 2, UTime: now.UnixMilli()},
						{Id: 3, UTime: now.UnixMilli()},
					}, nil)
				interRepo.EXPECT().GetByIds(gomock.Any(), []int64{1, 2, 3}).Return(
					map[int64]domain2.Interactive{
						1: {LikeCnt: 3},
						2: {LikeCnt: 2},
						3: {LikeCnt: 1},
					}, nil)
				//artRepo.EXPECT().GetRankingList(gomock.Any(), 3, 100).Return()
				return artRepo, interRepo
			},
			wantRes: []domain.Article{
				{Id: 3},
				{Id: 2},
				{Id: 1},
			},
			wantErr: nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			articleRepo, interRepo := tc.mock(ctrl)
			rank := NewRankingServiceI(articleRepo, interRepo, repository.RankingRepository(nil))
			arts, err := rank.topN(context.Background())
			for i, art := range arts {
				t.Log(art.UTime)
				arts[i].UTime = 0
			}
			assert.Equal(t, tc.wantRes, arts)
			assert.Equal(t, tc.wantErr, err)
		})
	}
}
