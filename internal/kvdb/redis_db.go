package kvdb

import (
	"Research/util"
	"errors"
	"github.com/go-redis/redis"
	"math"
	"strconv"
	"time"
)

var ErrNoData = errors.New("no data")

type Redis struct {
	Db     *redis.Client // 客户端
	addr   string        // 地址
	passwd string        // 密码
	dbno   int           // 数据库编号
}

type OptionsRedis func(*Redis)

func WithOptionAddr(addr string) OptionsRedis {
	return func(r *Redis) {
		r.addr = addr
	}
}

func WithOptionPasswd(passwd string) OptionsRedis {
	return func(r *Redis) {
		r.passwd = passwd
	}
}

func WithOptionDbno(dbno int) OptionsRedis {
	return func(r *Redis) {
		r.dbno = dbno
	}
}

func NewRedis(opts ...OptionsRedis) *Redis {
	r := &Redis{}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Redis) Open() error {
	r.Db = redis.NewClient(&redis.Options{
		Addr:     r.addr,
		Password: r.passwd,
		DB:       r.dbno,
	})
	result, err := r.Db.Ping().Result()
	util.Log.Printf("redis open result is: %s\n", result)
	return err
}

func (r *Redis) GetDbPath() string {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) Set(k, v []byte) error {
	err := r.Db.Set(string(k), string(v), 0).Err()
	util.Log.Printf("redis set %s %s", string(k), string(v))
	return err
}

func (r *Redis) BatchSet(keys, values [][]byte) error {
	length := len(keys) + len(values)
	pairs := make([]any, 0, length)

	for i := 0; i < length; i++ {
		pairs = append(pairs, string(keys[i]), string(values[i]))
	}
	err := r.Db.MSet(pairs...).Err()
	util.Log.Printf("redis BatchSet keys %d", len(keys))
	return err
}

func (r *Redis) Get(k []byte) ([]byte, error) {
	result, err := r.Db.Get(string(k)).Result()
	if err == redis.Nil {
		return nil, ErrNoData
	}
	util.Log.Printf("redis get result is: %s\n", result)
	return []byte(result), err
}

func (r *Redis) BatchGet(keys [][]byte) ([][]byte, error) {
	pairs := make([]string, 0, len(keys))
	ans := make([][]byte, 0, len(keys))
	//构造参数
	for i := 0; i < len(keys); i++ {
		pairs = append(pairs, string(keys[i]))
	}

	result, err := r.Db.MGet(pairs...).Result()
	if err != nil {
		return nil, err
	}
	//构造返回值
	for _, res := range result {
		ans = append(ans, res.([]byte))
	}
	util.Log.Printf("redis BatchGet result %v", ans)
	return ans, nil
}

func (r *Redis) Delete(k []byte) error {
	err := r.Db.Del(string(k)).Err()
	util.Log.Printf("redis delete %s", string(k))
	return err
}

func (r *Redis) BatchDelete(keys [][]byte) error {
	pairs := make([]string, 0, len(keys))
	for i := 0; i < len(keys); i++ {
		pairs = append(pairs, string(keys[i]))
	}
	err := r.Db.Del(pairs...).Err()
	util.Log.Printf("redis delete keys %d", len(keys))
	return err
}

func (r *Redis) Has(k []byte) bool {
	result, _ := r.Db.Exists(string(k)).Result()
	util.Log.Printf("redis exists %s", string(k))
	return result != 0
}

func (r *Redis) IterDB(fn func(k []byte, v []byte) error) int64 {
	var cursor uint64 = 0
	var ans int64 = 0
	for {
		keys, cursor := r.Db.Scan(cursor, ".*", 1000).Val()
		result, _ := r.Db.MGet(keys...).Result()
		ans += int64(len(keys))
		for i := 0; i < len(keys); i++ {
			// todo: 错误处理
			_ = fn([]byte(keys[i]), result[i].([]byte))
		}
		if cursor == 0 {
			break
		}
	}
	return ans
}

func (r *Redis) IterKey(fn func(k []byte) error) int64 {
	var cursor uint64 = 0
	var ans int64 = 0
	for {
		keys, cursor := r.Db.Scan(cursor, ".*", 1000).Val()
		ans += int64(len(keys))
		for i := 0; i < len(keys); i++ {
			// todo: 错误处理
			_ = fn([]byte(keys[i]))
		}
		if cursor == 0 {
			break
		}
	}
	return ans
}

const (
	lock   = "load"
	CURSOR = "cursor"
	lua    = `
		local res = redis.call("dbsize")
		return res
	`
)

func (r *Redis) IterKeyByWeight(rate float64, fn func(k, v []byte) error) int64 {

	// 直到获取锁
	for err := r.Db.SetNX(lock, "", 0).Err(); err != nil; {
		time.Sleep(100 * time.Millisecond)
	}
	// 获得keys数量
	result, err := r.Db.Eval(lua, nil).Result()
	if err != nil {
		util.Log.Println("执行lua err: ", err)
		return 0
	}
	keys, _ := strconv.ParseFloat(result.(string), 64)
	var cursor uint64 = 0
	//判断是否存在cursor
	cursorStr, err := r.Db.Get(CURSOR).Result()
	if err == nil {
		cursor, _ = strconv.ParseUint(cursorStr, 10, 64)
	}
	// 获得keySet
	need := int64(math.Ceil(keys * rate))
	cursor, _ = strconv.ParseUint(cursorStr, 10, 64)
	keySet, cursor := r.Db.Scan(cursor, ".*", need).Val()
	// 写入游标
	if cursor == 0 {
		r.Db.Del(CURSOR)
	} else {
		r.Db.Set(CURSOR, cursor, 0)
	}
	// 解锁
	r.Db.Del(lock)
	// 将key插入倒排索引
	val, _ := r.Db.MGet(keySet...).Result()
	for i := 0; i < len(keySet); i++ {
		// todo: 错误处理
		_ = fn([]byte(keySet[i]), val[i].([]byte))
	}

	return int64(len(keySet))
}

func (r *Redis) Close() error {
	return r.Db.Close()
}
