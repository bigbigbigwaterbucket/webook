package article

import (
	"context"
	"github.com/ecodeclub/ekit/slice"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"learning_go/webook/internal/domain"
	"learning_go/webook/internal/repository/cache"
	"learning_go/webook/internal/repository/dao/article"
)

type ArticleRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) (int64, error)
	//存储并同步数据
	Sync(ctx context.Context, article domain.Article) (int64, error)
	SyncStatus(ctx *gin.Context, id int64, uid int64, status uint8) error
	List(ctx *gin.Context, uid int64, offset int64, limit int64) ([]domain.Article, error)
}

type CachedArticleRepository struct {
	dao    article.ArticleDao
	author article.AuthorDao
	reader article.ReaderDao
	//耦合了dao操作的东西，建议只在使用事务的时候用这个db
	db    *gorm.DB
	cache cache.ArticleCache
}

func (c *CachedArticleRepository) List(ctx *gin.Context, uid int64, offset int64, limit int64) ([]domain.Article, error) {
	if offset == 0 && limit <= 100 {
		data, err := c.cache.GetFirstPage(ctx, uid)
		//注意这里缓存方法是允许有错误的
		if err == nil {
			return data[:limit], err
		}
	}
	//这里还要进行下数据结构的转换
	res, err := c.dao.GetByAuthor(ctx, uid, offset, limit)
	if err != nil {
		return nil, err
	}
	data := slice.Map[article.Article, domain.Article](res, func(idx int, src article.Article) domain.Article {
		return c.EntityToDomain(src)
	})
	//可以同步，也可以异步
	go func() {
		err := c.cache.SetFirstPage(ctx, uid, data)
		if err != nil {
			zap.L().Error("回写缓存失败", zap.Int64("uid", uid))
		}
	}()
	return data, nil
}

func (c *CachedArticleRepository) SyncStatus(ctx *gin.Context, id int64, uid int64, status uint8) error {
	return c.dao.SyncStatus(ctx, id, uid, status)
}

// 假设线上库和制作库使用同一个数据库，则尝试在repository层解决事务问题，那么需要在repo结构体传入一个db
// 保证线上库和制作库的同步操作同时成功或失败
func (c *CachedArticleRepository) SyncV2(ctx context.Context, art domain.Article) (int64, error) {
	//开启一个数据库事务
	tx := c.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		//很少遇到，数据库崩了才有可能
		return 0, tx.Error
	}
	//总是让他回滚，防止panic的时候事务挂在数据库上没提交
	//即使后续成功commit了，这里回滚也只会拿到error，不会影响程序执行
	defer tx.Rollback()
	//在后续数据库操作中传入事务，即可实现原子性
	//传入同一个数据库的事务
	author := article.NewGormArticleDao(tx)
	reader := article.NewReaderDaoI(tx)
	var aid int64
	aid = art.Id
	var err error
	if art.Id > 0 {
		aid, err = author.UpdateById(ctx, c.DomainToEntity(art))
	} else {
		aid, err = c.author.Insert(ctx, c.DomainToEntity(art))
	}
	if err != nil {
		return aid, err
	}
	err = reader.UpdateOrInsert(ctx, c.DomainToEntity(art))
	//提交事务
	tx.Commit()
	return aid, err
}

// 该同步数据库的操作默认线上库和制作库是不同数据库，因此没有开启本地事务
func (c *CachedArticleRepository) SyncV1(ctx context.Context, art domain.Article) (int64, error) {
	var aid int64
	aid = art.Id
	var err error
	if art.Id > 0 {
		_, err = c.dao.UpdateById(ctx, c.DomainToEntity(art))
		if err != nil {
			return art.Id, err
		}
	} else {
		aid, err = c.dao.Insert(ctx, c.DomainToEntity(art))
		if err != nil {
			return aid, err
		}
	}
	//这里没办法通过前端传过来的article是否带id来判断是否已经发布
	err = c.reader.UpdateOrInsert(ctx, article.PublishArticle{c.DomainToEntity(art)})
	return aid, err
}

// 在dao层处理事务的版本
func (c *CachedArticleRepository) Sync(ctx context.Context, art domain.Article) (int64, error) {
	//这里的sync实际上是publish调用的，也涉及对文章内容的改变，因此需要释放缓存
	defer func() {
		err := c.cache.DelFirstPage(ctx, art.Author.Id)
		if err != nil {
			zap.L().Error("删除缓存失败", zap.Error(err))
		}
	}()
	return c.dao.Sync(ctx, c.DomainToEntity(art))
}
func NewCachedArticleRepository(dao article.ArticleDao) *CachedArticleRepository {
	return &CachedArticleRepository{dao: dao}
}

func (c *CachedArticleRepository) Create(ctx context.Context, art domain.Article) (int64, error) {
	//插入一篇文章，所有缓存都删掉...  也没法判定第一页
	defer func() {
		err := c.cache.DelFirstPage(ctx, art.Author.Id)
		if err != nil {
			zap.L().Error("删除缓存失败", zap.Error(err))
		}
	}()
	return c.dao.Insert(ctx, article.Article{Id: art.Id, Title: art.Title, Content: art.Content, AuthorId: art.Author.Id, Status: art.Status.ToUnt8()})
}

func (c *CachedArticleRepository) Update(ctx context.Context, art domain.Article) (int64, error) {
	defer func() {
		err := c.cache.DelFirstPage(ctx, art.Author.Id)
		if err != nil {
			zap.L().Error("删除缓存失败", zap.Error(err))
		}
	}()
	return c.dao.UpdateById(ctx, article.Article{Id: art.Id, Title: art.Title, Content: art.Content, AuthorId: art.Author.Id, Status: art.Status.ToUnt8()})
}

// 这里dao层的ctime与utime就没必要修改了，也最好不要传进去，不要让他修改数据库的数据
func (c *CachedArticleRepository) DomainToEntity(art domain.Article) article.Article {
	return article.Article{Id: art.Id,
		Title:    art.Title,
		Content:  art.Content,
		AuthorId: art.Author.Id,
		Status:   art.Status.ToUnt8()}
}

func (c *CachedArticleRepository) EntityToDomain(art article.Article) domain.Article {
	return domain.Article{Id: art.Id,
		Title:   art.Title,
		Content: art.Content,
		Author:  domain.Author{Id: art.AuthorId},
		Status:  domain.ArticleStatus(art.Status),
		CTime:   art.CTime,
		UTime:   art.UTime,
	}
}
