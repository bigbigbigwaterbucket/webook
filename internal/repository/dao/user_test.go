package dao

import (
	"context"
	"database/sql"
	dao2 "learning_go/webook/user/repository/dao"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func TestGormUserDAO_Insert(t *testing.T) {
	testCases := []struct {
		name string
		user dao2.User
		//这里没用gomock，用的格式sqlmock，因此传的不是ctrl
		mock    func(t *testing.T) *sql.DB
		wantErr error
		//这里的返回值不只err，还有隐藏的返回值，即dao.user的id被修改
		//但是函数是值传递，虽然go的返回值可以返回局部变量，但是参数传递是值传递，函数里修改的是局部变量
		//想要测试，可以修改函数，使其额外返回局部变量，这里已pass
		wantId int64
	}{
		{
			name: "插入成功",
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				require.NoError(t, err)
				//返回值返回最后一个自增主键和影响行数
				res := sqlmock.NewResult(10, 1)
				//增删改用这个
				//这里的预期语句可以写正则表达式，只要插到users表里就可
				mock.ExpectExec("INSERT INTO `users` .*").
					WillReturnResult(res)
				//查询用这个 mock.ExpectQuery()
				return mockDB
			},
			wantErr: nil,
			wantId:  10,
		},
		{
			name: "邮箱/手机号冲突",
			mock: func(t *testing.T) *sql.DB {
				mockDB, mock, err := sqlmock.New()
				require.NoError(t, err)
				//增删改用这个
				//这里的预期语句可以写正则表达式，只要插到users表里就可
				mock.ExpectExec("INSERT INTO `users` .*").
					WillReturnError(&mysql.MySQLError{Number: 1062}) //这里mysql小心和gorm的mysql冲突
				//error是一个接口类型，接收指针还是结构体值，看方法接收器是怎么接收的
				return mockDB
			},
			wantErr: dao2.ErrUserDuplicate,
			wantId:  10,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := gorm.Open(gormmysql.New(gormmysql.Config{
				Conn: tc.mock(t),
				//SELECT VERSION;
				SkipInitializeWithVersion: true,
			}), &gorm.Config{
				//禁用默认事务
				//gorm在插入数据的时候，会默认开事务给你插入数据
				//insert xxx
				//gorm： begin insert xxx end
				SkipDefaultTransaction: true,
				// mock DB不需要ping，也ping不通
				DisableAutomaticPing: true,
			})
			require.NoError(t, err)
			d := dao2.NewUserDAO(db)
			err = d.Insert(context.Background(), tc.user)
			assert.Equal(t, tc.wantErr, err)
			//assert.Equal(t, tc.wantId, user.Id)
		})
	}
}
