package kvdb

import "github.com/go-redis/redis"

type Redis struct {
	db     *redis.Client // 客户端
	path   string        // 正排索引路径
	bucket []byte        // 表名
}

func (r *Redis) WithDatePath(path string) *Redis {
	r.path = path
	return r
}

func (r *Redis) WithBucket(bucket []byte) *Redis {
	r.bucket = bucket
	return r
}

func (r *Redis) Open() error {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) GetDbPath() string {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) Set(k, v []byte) error {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) BatchSet(keys, values [][]byte) error {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) Get(k []byte) ([]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) BatchGet(keys [][]byte) ([][]byte, error) {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) Delete(k []byte) error {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) BatchDelete(keys [][]byte) error {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) Has(k []byte) bool {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) IterDB(fn func(k []byte, v []byte) error) int64 {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) IterKey(fn func(k []byte) error) int64 {
	//TODO implement me
	panic("implement me")
}

func (r *Redis) Close() error {
	//TODO implement me
	panic("implement me")
}
