package article

import (
	"context"
	"learning_go/webook/article/domain"
	"learning_go/webook/article/repository/cache"
	article2 "learning_go/webook/article/repository/dao/article"
	"learning_go/webook/internal/repository"
	"time"

	"github.com/ecodeclub/ekit/slice"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

//go:generate mockgen -source=D:/go_project/learning_go/webook/internal/repository/article/article.go -package=repomocks -destination=D:/go_project/learning_go/webook/internal/repository/article/mocks/article_mocks.go
type ArticleRepository interface {
	Create(ctx context.Context, art domain.Article) (int64, error)
	Update(ctx context.Context, article domain.Article) (int64, error)
	//存储并同步数据
	Sync(ctx context.Context, article domain.Article) (int64, error)
	SyncStatus(ctx context.Context, id int64, uid int64, status uint8) error
	List(ctx context.Context, uid int64, offset int64, limit int64) ([]domain.Article, error)
	FindById(ctx context.Context, aid int64) (domain.Article, error)
	FindPublishedById(ctx context.Context, aid int64) (domain.Article, error)
	FindByIds(ctx context.Context, aids []int64) ([]domain.Article, error)
	GetRankingList(ctx context.Context, startTime time.Time, offset int, topNum int) ([]domain.Article, error)
}

type CachedArticleRepository struct {
	dao article2.ArticleDao
	//引入同层业务，实际上同层调用不是好的设计？
	userRepo repository.UserRepository
	author   article2.AuthorDao
	reader   article2.ReaderDao
	//耦合了dao操作的东西，建议只在使用事务的时候用这个db
	db    *gorm.DB
	cache cache.ArticleCache
}

func (c *CachedArticleRepository) GetRankingList(ctx context.Context, startTime time.Time, offset int, topNum int) ([]domain.Article, error) {
	//这里全遍历取数据就没必要存缓存了....
	arts, err := c.dao.GetRankingList(ctx, startTime, offset, topNum)
	if err != nil {
		return []domain.Article{}, err
	}
	return c.EntitysToDomains(arts), err
}

func (c *CachedArticleRepository) FindByIds(ctx context.Context, aids []int64) ([]domain.Article, error) {
	data, err := c.dao.GetByArticleIds(ctx, aids)
	return c.EntitysToDomains(data), err
}

func NewCachedArticleRepository(dao article2.ArticleDao, cache cache.ArticleCache, userRepo repository.UserRepository) *CachedArticleRepository {
	return &CachedArticleRepository{dao: dao, cache: cache, userRepo: userRepo}
}

func (c *CachedArticleRepository) FindPublishedById(ctx context.Context, aid int64) (domain.Article, error) {
	pArtCached, err := c.cache.GetPub(ctx, aid)
	if err == nil {
		println("读者命中文章缓存")
		return pArtCached, err
	}
	pArt, err := c.dao.GetPubByArticleId(ctx, aid)
	if err != nil {
		return domain.Article{}, err
	}
	user, err := c.userRepo.FindById(ctx, pArt.AuthorId)
	if err != nil {
		return domain.Article{}, err
	}
	//这里缓存可有可无？ 读者的缓存一般在文章Publish之后缓存一段时间
	//读者阅读这篇文章的时候也缓存
	res := domain.Article{
		Id:      pArt.Id,
		Title:   pArt.Title,
		Content: pArt.Content,
		Status:  domain.ArticleStatus(pArt.Status),
		CTime:   pArt.CTime,
		UTime:   pArt.UTime,
		Author:  domain.Author{Id: pArt.AuthorId, Name: user.Name},
	}
	go func() {
		err = c.cache.SetPub(ctx, aid, res)
		if err != nil {
			zap.L().Error("读者阅读后缓存失败", zap.Error(err))
		}
	}()
	return res, nil
}

func (c *CachedArticleRepository) FindById(ctx context.Context, aid int64) (domain.Article, error) {
	artCached, err := c.cache.Get(ctx, aid)
	if err == nil {
		println("list命中第一篇文章缓存")
		return artCached, err
	}
	art, err := c.dao.GetByArticleId(ctx, aid)
	if err != nil {
		return domain.Article{}, err
	}
	return c.EntityToDomain(art), nil
}

func (c *CachedArticleRepository) List(ctx context.Context, uid int64, offset int64, limit int64) ([]domain.Article, error) {
	if offset == 0 && limit <= 100 {
		data, err := c.cache.GetFirstPage(ctx, uid)
		//注意这里缓存方法是允许有错误的
		if err == nil {
			if len(data) > int(limit) {
				return data[:limit], err
			}
			//拿到数据后肯定会访问，因此也要预缓存
			go func() {
				c.preCache(ctx, data)
			}()
			return data, err
		}
	}
	//这里还要进行下数据结构的转换
	res, err := c.dao.GetByAuthor(ctx, uid, offset, limit)
	if err != nil {
		return nil, err
	}
	data := slice.Map[article2.Article, domain.Article](res, func(idx int, src article2.Article) domain.Article {
		return c.EntityToDomain(src)
	})
	//可以同步，也可以异步，一般异步都是做成可配置的
	go func() {
		err := c.cache.SetFirstPage(ctx, uid, data)
		if err != nil {
			zap.L().Error("list回写缓存失败", zap.Int64("uid", uid))
		}
		c.preCache(ctx, data)
	}()
	return data, nil
}

func (c *CachedArticleRepository) preCache(ctx context.Context, data []domain.Article) {
	const contentSizeThreshold = 1024 * 1024
	//限制文章(string)的字节数，如果大于1MB，就不缓存了，太大了
	//注意判断data是否大于0
	if len(data) > 0 && len(data[0].Content) <= contentSizeThreshold {
		err := c.cache.Set(ctx, data[0])
		if err != nil {
			zap.L().Error("首篇文章回写缓存失败", zap.Int64("uid", data[0].Id))
		}
	}
}

func (c *CachedArticleRepository) SyncStatus(ctx context.Context, id int64, uid int64, status uint8) error {
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
	author := article2.NewGormArticleDao(tx)
	reader := article2.NewReaderDaoI(tx)
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
	err = c.reader.UpdateOrInsert(ctx, article2.PublishArticle{c.DomainToEntity(art)})
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
	aid, err := c.dao.Sync(ctx, c.DomainToEntity(art))
	if err != nil {
		return 0, err
	}
	//先判断err，考虑缓存一致性问题，如果数据库没有，那么缓存也应该没有
	go func() {
		user, err := c.userRepo.FindById(ctx, art.Author.Id)
		if err != nil {
			//找不到user信息，那么后续缓存的信息里也没有user，不能直接用，这里就直接返回了
			//后续可以在使用缓存的地方检查一下？
			return
		}
		art.Author.Name = user.Name
		err = c.cache.SetPub(ctx, art.Id, art)
		if err != nil {
			zap.L().Error("提前设置新发表文章缓存失败", zap.Error(err))
		}
	}()
	return aid, nil
}

func (c *CachedArticleRepository) Create(ctx context.Context, art domain.Article) (int64, error) {
	//插入一篇文章，所有缓存都删掉...  也没法判定第一页
	defer func() {
		err := c.cache.DelFirstPage(ctx, art.Author.Id)
		if err != nil {
			zap.L().Error("删除缓存失败", zap.Error(err))
		}
	}()
	return c.dao.Insert(ctx, article2.Article{Id: art.Id, Title: art.Title, Content: art.Content, AuthorId: art.Author.Id, Status: art.Status.ToUnt8()})
}

func (c *CachedArticleRepository) Update(ctx context.Context, art domain.Article) (int64, error) {
	defer func() {
		err := c.cache.DelFirstPage(ctx, art.Author.Id)
		if err != nil {
			zap.L().Error("删除缓存失败", zap.Error(err))
		}
	}()
	return c.dao.UpdateById(ctx, article2.Article{Id: art.Id, Title: art.Title, Content: art.Content, AuthorId: art.Author.Id, Status: art.Status.ToUnt8()})
}

// 这里dao层的ctime与utime就没必要修改了，也最好不要传进去，不要让他修改数据库的数据
func (c *CachedArticleRepository) DomainToEntity(art domain.Article) article2.Article {
	return article2.Article{
		Id:       art.Id,
		Title:    art.Title,
		Content:  art.Content,
		AuthorId: art.Author.Id,
		Status:   art.Status.ToUnt8()}
}

func (c *CachedArticleRepository) EntityToDomain(art article2.Article) domain.Article {
	return domain.Article{
		Id:      art.Id,
		Title:   art.Title,
		Content: art.Content,
		Author:  domain.Author{Id: art.AuthorId},
		Status:  domain.ArticleStatus(art.Status),
		CTime:   art.CTime,
		UTime:   art.UTime,
	}
}

func (c *CachedArticleRepository) EntitysToDomains(articles []article2.Article) []domain.Article {
	return slice.Map[article2.Article, domain.Article](articles, func(idx int, src article2.Article) domain.Article {
		return c.EntityToDomain(src)
	})
}
