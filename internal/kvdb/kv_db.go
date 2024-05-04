package kvdb

import (
	"Research/util"
	"errors"
	"os"
	"strconv"
	"strings"
)

// 几种常见的基于LSM-tree算法实现的KV数据库
const (
	BOLT = iota
	BADGER
	REDIS
)

type IKeyValueDB interface {
	Open() error                              //初始化DB
	GetDbPath() string                        //获取存储数据的目录
	Set(k, v []byte) error                    //写入<key, value>
	BatchSet(keys, values [][]byte) error     //批量写入<key, value>
	Get(k []byte) ([]byte, error)             //读取key对应的value
	BatchGet(keys [][]byte) ([][]byte, error) //批量读取，注意不保证顺序
	Delete(k []byte) error                    //删除
	BatchDelete(keys [][]byte) error          //批量删除
	Has(k []byte) bool                        //判断某个key是否存在
	IterDB(fn func(k, v []byte) error) int64  //遍历数据库，返回数据的条数
	IterKey(fn func(k []byte) error) int64    //遍历所有key，返回数据的条数
	Close() error                             //把内存中的数据flush到磁盘，同时释放文件锁
}

// Factory工厂模式，把类的创建和使用分隔开。Get函数就是一个工厂，它返回产品的接口，即它可以返回各种各样的具体产品。
func GetKvdb(dbtype int, path string) (IKeyValueDB, error) {
	var db IKeyValueDB
	switch dbtype {
	case REDIS:
		optsRedis, err := connectRemoteKvdb(path)
		if err != nil {
			return nil, err
		}
		db = NewRedis(optsRedis...)
	case BADGER:
		// todo: 添加这两个类
		err := createLocalKvdb(path)
		if err != nil {
			return nil, err
		}
		//db = new(Badger).WithDataPath(path)
	default:
		err := createLocalKvdb(path)
		if err != nil {
			return nil, err
		}
		//db = new(Bolt).WithDataPath(path).WithBucket("radic") //Builder生成器模式
	}
	err := db.Open() //创建具体KVDB的细节隐藏在Open()函数里。在这里【创建类】
	return db, err
}

func createLocalKvdb(path string) error { //通过Get函数【使用类】
	paths := strings.Split(path, "/")
	parentPath := strings.Join(paths[0:len(paths)-1], "/") //父路径

	info, err := os.Stat(parentPath)
	if os.IsNotExist(err) { //如果父路径不存在则创建
		util.Log.Printf("create dir %s", parentPath)
		err := os.MkdirAll(parentPath, os.ModePerm)
		if err != nil {
			return err
		} //数字前的0或0o都表示八进制
	} else { //父路径存在
		if info.Mode().IsRegular() { //如果父路径是个普通文件，则把它删掉
			util.Log.Printf("%s is a regular file, will delete it", parentPath)
			err := os.Remove(parentPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// pah参数：“地址:端口号/密码/库编号” （“/” 不能省略）
func connectRemoteKvdb(path string) ([]OptionsRedis, error) {
	paths := strings.Split(path, "/")
	if len(paths) != 3 || len(paths[0]) == 0 {
		return nil, errors.New("参数不正确")
	}

	opts := make([]OptionsRedis, 0, 3)

	opts = append(opts, WithOptionAddr(paths[0]))

	if len(paths[1]) != 0 {
		opts = append(opts, WithOptionPasswd(paths[1]))
	}

	if len(paths[2]) != 0 {
		atoi, err := strconv.Atoi(paths[3])
		if err != nil {
			return nil, errors.New("参数库名，不正确")
		}
		opts = append(opts, WithOptionDbno(atoi))
	}

	return opts, nil
}
