package cache

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type MemCodeCache struct {
	mutex sync.Mutex
	mem   map[string]*memData
}

func NewMemCodeCache() *MemCodeCache {
	return &MemCodeCache{mutex: sync.Mutex{}, mem: map[string]*memData{}}
}

type memData struct {
	data       any
	verifyTime int
	ctime      time.Time
}

func (m *MemCodeCache) Set(ctx context.Context, biz, phone, code string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if val, ok := m.mem[m.key(biz, phone)]; !ok {
		//没设置过
		m.mem[m.key(biz, phone)] = &memData{data: code, ctime: time.Now(), verifyTime: 3}
	} else {
		//设置过，检查过期时间
		diff := time.Now().Sub(val.ctime)
		if diff.Seconds() < 50 {
			//没过期
			return errors.New("验证码发送过于频繁")
		} else {
			//过期了
			m.mem[m.key(biz, phone)] = &memData{data: code, ctime: time.Now(), verifyTime: 3}
		}
	}
	return nil
}

func (m *MemCodeCache) Verify(ctx context.Context, biz, phone, inputCode string) error {
	//防止set和verify方法同时进行，然而set阻塞在写导致验证码未设置错误
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if val, ok := m.mem[m.key(biz, phone)]; !ok {
		//没设置过
		return errors.New("验证码未设置")
	} else {
		//设置过，检查过期时间
		diff := time.Now().Sub(val.ctime)
		if diff.Seconds() < 50 && val.verifyTime > 0 {
			//没过期，并且输错次数小于3
			val.verifyTime--
			valS, ok := val.data.(string)
			if !ok {
				return errors.New("系统错误")
			}
			if valS == inputCode {
				val.verifyTime = -1
				return nil
			}
			return errors.New("验证码不对")
		} else {
			//过期了或者验证次数过多
			return errors.New("验证码已过期")
		}
	}
}

func (m *MemCodeCache) key(biz, phone string) string {
	return fmt.Sprintf("phone_code:%s:%s", biz, phone)
}
