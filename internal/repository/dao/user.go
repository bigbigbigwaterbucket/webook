package dao

import (
	"context"
	"database/sql"
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"time"
)

var (
	//	var a = 1
	//	println(a)  使用var声明变量时，不需要:=，:=只能用于函数内部
	ErrUserDuplicate = errors.New("邮箱冲突")
	//用的就是db的where的error，不用新建
	ErrUserNotFound = gorm.ErrRecordNotFound
)

type UserDAO struct {
	DB *gorm.DB
}

func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{DB: db}
}

func (this *UserDAO) FindByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := this.DB.Where("email=?", email).First(&u).Error //不区分大小写
	return u, err
}

func (this *UserDAO) FindByPhone(ctx context.Context, phone string) (User, error) {
	var u User
	err := this.DB.Where("phone=?", phone).First(&u).Error //不区分大小写
	return u, err
}

func (this *UserDAO) FindById(ctx context.Context, userId int64) (User, error) {
	var u User
	err := this.DB.Where("Id=?", userId).First(&u).Error //不区分大小写
	return u, err
}

func (this *UserDAO) Insert(ctx context.Context, u User) error {
	//在insert里存创建时间Ctime
	now := time.Now().UnixMilli()
	u.Utime = now
	u.Ctime = now
	err := this.DB.WithContext(ctx).Create(&u).Error //ctx一直保持调用，返回的是error接口，具体是什么错误可以类型断言

	//能不能先查数据库，再看有没有邮箱冲突？
	//select * from users where email=123@qq.com for update 上锁，但这锁是间隙锁？？？ 反正就是会出现并发问题
	//例如不同机器的不同线程，同时去查询

	//底层强耦合，写死是mysql数据库
	if mysqlErr, ok := err.(*mysql.MySQLError); ok {
		const uniqueConflictsErrNo = 1062 //唯一索引冲突，只有邮箱是唯一索引
		if mysqlErr.Number == uniqueConflictsErrNo {
			return ErrUserDuplicate
		} //邮箱冲突或者手机号冲突
	}
	return err
}

func (this *UserDAO) Update(ctx context.Context, u User) error {
	var user User
	this.DB.First(&user, u.Id)
	err := this.DB.Model(&user).Update("Name", u.Name).Error
	if err != nil {
		return err
	}
	err = this.DB.Model(&user).Update("Birthday", u.Birthday).Error
	if err != nil {
		return err
	}
	err = this.DB.Model(&user).Update("Introduce", u.Introduce).Error
	if err != nil {
		return err
	}
	return nil
}

// 直接对应数据库表结构
// entity实体，model/PO
type User struct {
	Id int64 `gorm:"primaryKey,autoIncrement"`
	//邮箱唯一
	Email     sql.NullString `gorm:"unique"`
	Phone     sql.NullString `gorm:"unique"` //该类型允许唯一索引有多个空值，不会冲突（不能是“”） 也可以用引用类型*string，但是要判空
	Password  string
	Name      string
	Birthday  string
	Introduce string
	//创建时间，毫秒数 1970年开始，如果用time，和时区有关，go的time和数据库的time很麻烦，这里用UTC，要和前端交换数据时才转换
	Ctime int64
	//更新时间，毫秒数
	Utime int64
}
