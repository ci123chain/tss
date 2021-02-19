package tgo

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/youtube/vitess/go/pools"
	"golang.org/x/net/context"
)

type DaoRedis struct {
	KeyName    string
	Persistent bool // 持久化key
}

type redisPool struct {
	redisPool     *pools.ResourcePool
	redisPoolMux  sync.RWMutex
	redisPPool    *pools.ResourcePool // 持久化Pool
	redisPPoolMux sync.RWMutex
}

func (p *redisPool) Get(persistent bool) *pools.ResourcePool {
	if persistent {
		p.redisPPoolMux.RLock()
		defer p.redisPPoolMux.RUnlock()
		return p.redisPPool
	} else {
		p.redisPoolMux.RLock()
		defer p.redisPoolMux.RUnlock()
		return p.redisPool
	}
}

func (p *redisPool) Set(pool *pools.ResourcePool) {
	p.redisPoolMux.Lock()
	defer p.redisPoolMux.Unlock()
	p.redisPool = pool
}

func (p *redisPool) Put(resource pools.Resource, persistent bool) {
	if persistent {
		p.redisPPool.Put(resource)
	} else {
		p.redisPool.Put(resource)
	}
}

var daoPool redisPool

type ResourceConn struct {
	redis.Conn
}

func (r ResourceConn) Close() {
	r.Conn.Close()
}

func RedisGetAddress(conf *Redis) (address string) {
	address = conf.Address
	return
}

func dial() (conn redis.Conn, err error) {
	cacheConfig := ConfigCacheGetRedisWithConn()
	address := RedisGetAddress(cacheConfig)
	var opt []redis.DialOption
	opt = append(opt, redis.DialConnectTimeout(time.Duration(cacheConfig.ConnectTimeout)*time.Millisecond),
		redis.DialReadTimeout(time.Duration(cacheConfig.ReadTimeout)*time.Millisecond),
		redis.DialWriteTimeout(time.Duration(cacheConfig.WriteTimeout)*time.Millisecond))
	if cacheConfig.Password != "" {
		opt = append(opt, redis.DialPassword(cacheConfig.Password))
	}
	conn, err = redis.Dial("tcp", address, opt...)
	if err != nil {
		LogErrorw(LogNameNet, "dial redis pool error", err)
		return nil, err
	}
	return conn, nil
}

//初始化redis连接池
func initRedisPoll() {

	cacheConfig := ConfigCacheGetRedisWithConn()
	if cacheConfig.PoolMinActive == 0 {
		cacheConfig.PoolMinActive = 1
	}
	var poolHandler *pools.ResourcePool
	poolHandler = pools.NewResourcePool(func() (pools.Resource, error) {
		c, err := dial()
		return ResourceConn{Conn: c}, err
	}, cacheConfig.PoolMinActive, cacheConfig.PoolMaxActive, time.Duration(cacheConfig.PoolIdleTimeout)*time.Millisecond)
	daoPool.Set(poolHandler)
}

//获取redis连接
func (p *DaoRedis) getRedisConn() (pools.Resource, error) {
	var poolHandler *pools.ResourcePool
	poolHandler = daoPool.Get(p.Persistent)
	if poolHandler != nil {
		var r pools.Resource
		var err error
		ctx := context.TODO()
		r, err = poolHandler.Get(ctx)
		if err != nil {
			LogErrorw(LogNameNet, "redis get connection error", err)
		} else if r == nil {
			err = errors.New("redis pool resource is null")
		} else {
			rc := r.(ResourceConn)
			if rc.Conn.Err() != nil {
				LogErrorw(LogNameNet, "redis rc connection error", err)
				rc.Close()
				//连接断开，重新打开
				var conn redis.Conn
				conn, err = dial()
				if err != nil {
					poolHandler.Put(r)
					LogErrorw(LogNameNet, "redis redial connection error", err)
					return nil, err
				} else {
					return ResourceConn{Conn: conn}, err
				}
			}
		}
		return r, err
	}
	UtilLogError("redis pool is null")
	return ResourceConn{}, errors.New("redis pool is null")
}

func (p *DaoRedis) getKey(key string) string {
	cacheConfig := ConfigCacheGetRedisWithConn()
	prefixRedis := cacheConfig.Prefix
	if strings.Trim(key, " ") == "" {
		return fmt.Sprintf("%s:%s", prefixRedis, p.KeyName)
	}
	return fmt.Sprintf("%s:%s:%s", prefixRedis, p.KeyName, key)
}

func (p *DaoRedis) doSet(cmd string, key string, value interface{}, expire int, fields ...string) (interface{}, error) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return nil, err
	}
	key = p.getKey(key)
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	data, err := json.Marshal(value)
	if err != nil {
		LogErrorw(LogNameLogic, "redis marshal data to json error", err)
		return nil, err
	}
	if expire == 0 {
		cacheConfig := ConfigCacheGetRedisWithConn()
		expire = cacheConfig.Expire
	}
	var reply interface{}
	var errDo error
	if len(fields) == 0 {
		if expire > 0 && strings.ToUpper(cmd) == "SET" {
			reply, errDo = redisClient.Do(cmd, key, data, "ex", expire)
		} else {
			reply, errDo = redisClient.Do(cmd, key, data)
		}
	} else {
		field := fields[0]
		reply, errDo = redisClient.Do(cmd, key, field, data)
	}
	if errDo != nil {
		LogErrorw(LogNameRedis, "run redis command error", errDo)
		return nil, errDo
	}
	//set expire
	if expire > 0 && strings.ToUpper(cmd) != "SET" {
		_, errExpire := redisClient.Do("EXPIRE", key, expire)
		if errExpire != nil {
			LogErrorw(LogNameRedis, "run redis EXPIRE command error", errExpire)
		}
	}
	return reply, errDo
}

func (p *DaoRedis) doSetNX(cmd string, key string, value interface{}, expire int, field ...string) (int64, bool) {
	reply, err := p.doSet(cmd, key, value, expire, field...)
	if err != nil {
		return 0, false
	}
	replyInt, ok := reply.(int64)
	if !ok {
		LogErrorw(LogNameRedis, "HSetNX reply to int error", errors.New("doSetNX err"))
		return 0, false
	}
	return replyInt, true
}
func (p *DaoRedis) doMSet(cmd string, key string, value map[string]interface{}) (interface{}, error) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return nil, err
	}
	defer daoPool.Put(redisResource, p.Persistent)
	var args []interface{}
	if key != "" {
		key = p.getKey(key)
		args = append(args, key)
	}
	for k, v := range value {
		data, errJson := json.Marshal(v)
		if errJson != nil {
			LogErrorw(LogNameLogic, "redis marshal data error", errJson)
			return nil, errJson
		}
		if key == "" {
			args = append(args, p.getKey(k), data)
		} else {
			args = append(args, k, data)
		}
	}
	redisClient := redisResource.(ResourceConn)
	var reply interface{}
	var errDo error
	reply, errDo = redisClient.Do(cmd, args...)
	if errDo != nil {
		LogErrorw(LogNameRedis, "run redis command error", errDo)
		return nil, errDo
	}
	return reply, errDo
}

func (p *DaoRedis) doGet(cmd string, key string, value interface{}, fields ...string) (bool, error) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return false, err
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	var result interface{}
	var errDo error
	var args []interface{}
	args = append(args, key)
	for _, f := range fields {
		args = append(args, f)
	}
	result, errDo = redisClient.Do(cmd, args...)
	if errDo != nil {
		LogErrorw(LogNameRedis, "run redis command error", errDo)
		return false, errDo
	}
	if result == nil {
		value = nil
		return false, nil
	}
	if reflect.TypeOf(result).Kind() == reflect.Slice {
		byteResult := result.([]byte)
		strResult := string(byteResult)
		if strResult == "[]" {
			return true, nil
		}
	}
	errorJson := json.Unmarshal(result.([]byte), value)
	if errorJson != nil {
		if reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.TypeOf(value).Elem().Kind() == reflect.String {
			var strValue string
			strValue = string(result.([]byte))
			v := value.(*string)
			*v = strValue
			value = v
			return true, nil
		}
		LogErrorw(LogNameRedis, "get redis command result error", errorJson)
		return false, errorJson
	}
	return true, nil
}

func (p *DaoRedis) doMGet(cmd string, args []interface{}, value interface{}) error {
	refValue := reflect.ValueOf(value)
	if refValue.Kind() != reflect.Ptr || refValue.Elem().Kind() != reflect.Slice || refValue.Elem().Type().Elem().Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("value is not *[]*object:  %v", refValue.Elem().Type().Elem().Kind()))
	}
	refSlice := refValue.Elem()
	refItem := refSlice.Type().Elem()
	redisResource, err := p.getRedisConn()
	if err != nil {
		return err
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	result, errDo := redis.ByteSlices(redisClient.Do(cmd, args...))
	if errDo != nil {
		LogErrorw(LogNameRedis, "run redis command error", errDo)
		return errDo
	}
	if result == nil {
		return nil
	}
	if len(result) > 0 {
		for i := 0; i < len(result); i++ {
			r := result[i]
			if r != nil {
				item := reflect.New(refItem)
				errorJson := json.Unmarshal(r, item.Interface())
				if errorJson != nil {
					LogErrorw(LogNameRedis, "redis command json Unmarshal error", errorJson)
					return errorJson
				}
				refSlice.Set(reflect.Append(refSlice, item.Elem()))
			} else {
				refSlice.Set(reflect.Append(refSlice, reflect.Zero(refItem)))
			}
		}
	}
	return nil
}

func (p *DaoRedis) doMGetGo(keys []string, value interface{}) error {
	var (
		args     []interface{}
		keysMap  sync.Map
		keysLen  int
		resultDo bool
		wg       sync.WaitGroup
	)
	keysLen = len(keys)
	if keysLen == 0 {
		return nil
	}
	refValue := reflect.ValueOf(value)
	if refValue.Kind() != reflect.Ptr || refValue.Elem().Kind() != reflect.Slice || refValue.Elem().Type().Elem().Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("value is not *[]*object:  %v", refValue.Elem().Type().Elem().Kind()))
	}
	refSlice := refValue.Elem()
	refItem := refSlice.Type().Elem()
	resultDo = true
	for _, v := range keys {
		args = append(args, p.getKey(v))
	}
	wg.Add(keysLen)
	for _, v := range args {
		go func(getK interface{}) {
			redisResource, err := p.getRedisConn()
			if err != nil {
				resultDo = false
			} else {
				redisClient := redisResource.(ResourceConn)
				rDo, errDo := redisClient.Do("GET", getK)
				keysMap.Store(getK, rDo)
				daoPool.Put(redisResource, p.Persistent)
				if errDo != nil {
					LogErrorw(LogNameRedis, "doMGetGo run redis command error", errDo)
					resultDo = false
				}
			}
			wg.Done()
		}(v)
	}
	wg.Wait()
	if !resultDo {
		return errors.New("doMGetGo one get error")
	}
	//整合结果
	for _, v := range args {
		r, ok := keysMap.Load(v)
		if ok && r != nil {
			item := reflect.New(refItem)
			errorJson := json.Unmarshal(r.([]byte), item.Interface())
			if errorJson != nil {
				LogErrorw(LogNameRedis, "doMGetGo GET command result error", errorJson)
				return errorJson
			}
			refSlice.Set(reflect.Append(refSlice, item.Elem()))
		} else {
			refSlice.Set(reflect.Append(refSlice, reflect.Zero(refItem)))
		}
	}

	return nil
}

func (p *DaoRedis) doMGetStringMap(cmd string, args ...interface{}) (err error, data map[string]string) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return err, nil
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	data, err = redis.StringMap(redisClient.Do(cmd, args...))
	if err != nil {
		LogErrorw(LogNameRedis, "doMGetStringMap run redis command error", err)
		return err, nil
	}
	return
}

func (p *DaoRedis) doMGetIntMap(cmd string, args ...interface{}) (err error, data map[string]int) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return err, nil
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	data, err = redis.IntMap(redisClient.Do(cmd, args...))
	if err != nil {
		LogErrorw(LogNameRedis, "doMGetIntMap run redis command error", err)
		return err, nil
	}
	return
}

func (p *DaoRedis) doIncr(cmd string, key string, value int, expire int, fields ...string) (int, bool) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return 0, false
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	var data interface{}
	var errDo error
	if len(fields) == 0 {
		data, errDo = redisClient.Do(cmd, key, value)
	} else {
		field := fields[0]
		data, errDo = redisClient.Do(cmd, key, field, value)
	}
	if errDo != nil {
		LogErrorw(LogNameRedis, "doIncr run redis command error", errDo)
		return 0, false
	}
	count, result := data.(int64)
	if !result {
		LogErrorw(LogNameRedis, "doIncr get command result error", errors.New("doIncr err"))
		return 0, false
	}
	if expire == 0 {
		cacheConfig := ConfigCacheGetRedisWithConn()
		expire = cacheConfig.Expire
	}
	//set expire
	if expire > 0 {
		_, errExpire := redisClient.Do("EXPIRE", key, expire)
		if errExpire != nil {
			LogErrorw(LogNameRedis, "run redis EXPIRE command error", errExpire)
		}
	}
	return int(count), true
}

func (p *DaoRedis) doDel(cmd string, data ...interface{}) error {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return err
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	_, errDo := redisClient.Do(cmd, data...)
	if errDo != nil {
		LogErrorw(LogNameRedis, "doDel run redis command error", errDo)
	}
	return errDo
}

/*基础结束*/

func (p *DaoRedis) Set(key string, value interface{}) bool {
	_, err := p.doSet("SET", key, value, 0)
	if err != nil {
		return false
	}
	return true
}

//
func (p *DaoRedis) SetE(key string, value interface{}) error {
	_, err := p.doSet("SET", key, value, 0)
	return err
}

//MSet mset
func (p *DaoRedis) MSet(datas map[string]interface{}) bool {
	_, err := p.doMSet("MSET", "", datas)
	if err != nil {
		return false
	}
	return true
}

//SetEx setex
func (p *DaoRedis) SetEx(key string, value interface{}, expire int) bool {
	_, err := p.doSet("SET", key, value, expire)
	if err != nil {
		return false
	}
	return true
}

//SetEx setex
func (p *DaoRedis) SetExE(key string, value interface{}, expire int) error {
	_, err := p.doSet("SET", key, value, expire)
	return err
}

//Expire expire
func (p *DaoRedis) Expire(key string, expire int) bool {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return false
	}
	key = p.getKey(key)
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	_, err = redisClient.Do("EXPIRE", key, expire)
	if err != nil {
		LogErrorw(LogNameRedis, "Expire run redis EXPIRE command error", err)
		return false
	}
	return true
}

func (p *DaoRedis) Get(key string, data interface{}) bool {
	result, err := p.doGet("GET", key, data)
	if err == nil && result {
		return true
	}
	return false
}
func (p *DaoRedis) GetE(key string, data interface{}) error {
	_, err := p.doGet("GET", key, data)
	return err
}

// 返回 1. key是否存在 2. error
func (p *DaoRedis) GetRaw(key string, data interface{}) (bool, error) {
	return p.doGet("GET", key, data)
}

func (p *DaoRedis) MGet(keys []string, data interface{}) error {
	var args []interface{}
	for _, v := range keys {
		args = append(args, p.getKey(v))
	}
	err := p.doMGet("MGET", args, data)
	return err
}

//封装mget通过go并发get
func (p *DaoRedis) MGetGo(keys []string, data interface{}) error {
	err := p.doMGetGo(keys, data)
	return err
}

func (p *DaoRedis) Incr(key string) (int, bool) {
	return p.doIncr("INCRBY", key, 1, 0)
}

func (p *DaoRedis) IncrBy(key string, value int) (int, bool) {
	return p.doIncr("INCRBY", key, value, 0)
}
func (p *DaoRedis) SetNX(key string, value interface{}) (int64, bool) {
	return p.doSetNX("SETNX", key, value, 0)
}

func (p *DaoRedis) SetNXNoExpire(key string, value interface{}) (int64, bool) {
	return p.doSetNX("SETNX", key, value, -1)
}

func (p *DaoRedis) Del(key string) bool {
	key = p.getKey(key)
	err := p.doDel("DEL", key)
	if err != nil {
		return false
	}
	return true
}

func (p *DaoRedis) MDel(key ...string) bool {
	var keys []interface{}
	for _, v := range key {
		keys = append(keys, p.getKey(v))
	}
	err := p.doDel("DEL", keys...)
	if err != nil {
		return false
	}
	return true
}

func (p *DaoRedis) Exists(key string) (bool, error) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return false, err
	}
	key = p.getKey(key)
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	data, err := redisClient.Do("EXISTS", key)
	if err != nil {
		LogErrorw(LogNameRedis, "Exists run redis EXISTS command error", err)
		return false, err
	}
	count, result := data.(int64)
	if !result {
		err := errors.New(fmt.Sprintf("get EXISTS command result failed:%v ,is %v", data, reflect.TypeOf(data)))
		LogErrorw(LogNameRedis, "get EXISTS command result error", err)
		return false, err
	}
	if count == 1 {
		return true, nil
	}
	return false, nil
}

//hash start
func (p *DaoRedis) HIncrby(key string, field string, value int) (int, bool) {
	return p.doIncr("HINCRBY", key, value, 0, field)
}

func (p *DaoRedis) HGet(key string, field string, value interface{}) bool {
	result, err := p.doGet("HGET", key, value, field)
	if err == nil && result {
		return true
	}
	return false
}

//HGetE 返回error
func (p *DaoRedis) HGetE(key string, field string, value interface{}) error {
	_, err := p.doGet("HGET", key, value, field)
	return err
}

//HGetRaw 返回 1. key是否存在 2. error
func (p *DaoRedis) HGetRaw(key string, field string, value interface{}) (bool, error) {
	return p.doGet("HGET", key, value, field)
}

func (p *DaoRedis) HMGet(key string, fields []interface{}, data interface{}) error {
	var args []interface{}
	args = append(args, p.getKey(key))
	for _, v := range fields {
		args = append(args, v)
	}
	err := p.doMGet("HMGET", args, data)
	return err
}

func (p *DaoRedis) HSet(key string, field string, value interface{}) bool {
	_, err := p.doSet("HSET", key, value, 0, field)
	if err != nil {
		return false
	}
	return true
}
func (p *DaoRedis) HSetNX(key string, field string, value interface{}) (int64, bool) {
	return p.doSetNX("HSETNX", key, value, 0, field)
}

//HMSet value是filed:data
func (p *DaoRedis) HMSet(key string, value map[string]interface{}) bool {
	_, err := p.doMSet("HMSet", key, value)
	if err != nil {
		return false
	}
	return true
}

//HMSetE value是filed:data
func (p *DaoRedis) HMSetE(key string, value map[string]interface{}) error {
	_, err := p.doMSet("HMSet", key, value)
	return err
}

func (p *DaoRedis) HLen(key string, data *int) bool {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	resultData, errDo := redisClient.Do("HLEN", key)
	if errDo != nil {
		LogErrorw(LogNameRedis, "HLen run redis HLEN command error", errDo)
		return false
	}
	length, b := resultData.(int64)
	if !b {
		LogErrorw(LogNameRedis, "HLen redis data convert to int64 error", errors.New("HLen err"))
	}
	*data = int(length)
	return b
}

func (p *DaoRedis) HDel(key string, data ...interface{}) bool {
	var args []interface{}
	key = p.getKey(key)
	args = append(args, key)
	for _, item := range data {
		args = append(args, item)
	}
	err := p.doDel("HDEL", args...)
	if err != nil {
		LogErrorw(LogNameRedis, "HDel run redis HDEL command error", err)
		return false
	}
	return true
}

func (p *DaoRedis) HExists(key string, field string) (bool, error) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return false, err
	}
	key = p.getKey(key)
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	data, err := redisClient.Do("HEXISTS", key, field)
	if err != nil {
		LogErrorw(LogNameRedis, "HExists run redis HEXISTS command error", err)
		return false, err
	}
	count, result := data.(int64)
	if !result {
		err := errors.New(fmt.Sprintf("get HEXISTS command result failed:%v ,is %v", data, reflect.TypeOf(data)))
		LogErrorw(LogNameRedis, "HExists get HEXISTS command result error", err)
		return false, err
	}
	if count == 1 {
		return true, nil
	}
	return false, nil
}

// hash end

// sorted set start
func (p *DaoRedis) ZAdd(key string, score interface{}, data interface{}) bool {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	_, errDo := redisClient.Do("ZADD", key, score, data)
	if errDo != nil {
		LogErrorw(LogNameRedis, "ZAdd run redis ZAdd command error", errDo)
		return false
	}
	return true
}

func (p *DaoRedis) ZCard(key string) (data int, err error) {
	var redisResource pools.Resource
	redisResource, err = p.getRedisConn()
	if err != nil {
		return
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	var reply interface{}
	reply, err = redisClient.Do("ZCARD", key)
	if err != nil {
		LogErrorw(LogNameRedis, "ZCard run redis ZCard command error", err)
		return
	}
	if v, ok := reply.(int64); ok {
		data = int(v)
		return
	} else {
		err = errors.New(fmt.Sprintf("ZCard get replay is not int64:%v", reply))
		LogErrorw(LogNameRedis, "ZCard get replay is not int64 error", err)
		return
	}
}

func (p *DaoRedis) ZCount(key string, min, max int) (data int, err error) {
	var redisResource pools.Resource
	redisResource, err = p.getRedisConn()
	if err != nil {
		return
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	var reply interface{}
	reply, err = redisClient.Do("ZCOUNT", key, min, max)
	if err != nil {
		LogErrorw(LogNameRedis, "ZCount run redis ZCOUNT command error", err)
		return
	}
	if v, ok := reply.(int64); ok {
		data = int(v)
		return
	} else {
		err = errors.New(fmt.Sprintf("ZCount get replay is not int64:%v", reply))
		LogErrorw(LogNameRedis, "ZCount get replay is not int64 error", err)
		return
	}
}

func (p *DaoRedis) ZIncrBy(key string, increment int, member interface{}) bool {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	key = p.getKey(key)
	_, errDo := redisClient.Do("ZINCRBY", key, increment, member)
	if errDo != nil {
		LogErrorw(LogNameRedis, "ZIncrBy run redis ZINCRBY command error", errDo)
		return false
	}
	return true
}

// sorted set start
func (p *DaoRedis) ZAddM(key string, value map[string]interface{}) bool {
	_, err := p.doMSet("ZADD", key, value)
	if err != nil {
		return false
	}
	return true
}

func (p *DaoRedis) ZGetByScore(key string, sort bool, start int, end int, value interface{}) error {
	var cmd string
	if sort {
		cmd = "ZRANGEBYSCORE"
	} else {
		cmd = "ZREVRANGEBYSCORE"
	}
	var args []interface{}
	args = append(args, p.getKey(key))
	args = append(args, start)
	args = append(args, end)
	err := p.doMGet(cmd, args, value)
	return err
}

func (p *DaoRedis) ZGet(key string, sort bool, start int, end int, value interface{}) error {

	var cmd string
	if sort {
		cmd = "ZRANGE"
	} else {
		cmd = "ZREVRANGE"
	}

	var args []interface{}
	args = append(args, p.getKey(key))
	args = append(args, start)
	args = append(args, end)

	err := p.doMGet(cmd, args, value)

	return err
}

func (p *DaoRedis) ZGetWithScores(key string, sort bool, start int, end int) (err error, data map[string]string) {

	var cmd string
	if sort {
		cmd = "ZRANGE"
	} else {
		cmd = "ZREVRANGE"
	}

	var args []interface{}
	args = append(args, p.getKey(key))
	args = append(args, start)
	args = append(args, end)
	args = append(args, "WITHSCORES")

	err, data = p.doMGetStringMap(cmd, args...)

	return
}

func (p *DaoRedis) ZRank(key string, member string, sort bool) (bool, int) {

	var cmd string
	if sort {
		cmd = "ZRANK"
	} else {
		cmd = "ZREVRANK"
	}

	redisResource, err := p.getRedisConn()

	if err != nil {
		return false, 0
	}
	defer daoPool.Put(redisResource, p.Persistent)

	redisClient := redisResource.(ResourceConn)

	key = p.getKey(key)

	result, errDo := redisClient.Do(cmd, key, member)

	if errDo != nil {
		LogErrorw(LogNameRedis, "ZRank run redis command error", errDo)
		return false, 0
	}
	if v, ok := result.(int64); ok {
		return true, int(v)
	}
	return false, 0
}

func (p *DaoRedis) ZScore(key string, member string, value interface{}) bool {

	var cmd string
	cmd = "ZSCORE"

	result, err := p.doGet(cmd, key, value, member)
	if err == nil && result {
		return true
	}

	return false
}

func (p *DaoRedis) ZRevRange(key string, start int, end int, value interface{}) error {
	return p.ZGet(key, false, start, end, value)
}

func (p *DaoRedis) ZRem(key string, data ...interface{}) bool {

	var args []interface{}

	key = p.getKey(key)
	args = append(args, key)

	for _, item := range data {
		args = append(args, item)
	}

	err := p.doDel("ZREM", args...)

	if err != nil {
		return false
	}
	return true
}

//list start

func (p *DaoRedis) LRange(start int, end int, value interface{}) (err error) {
	key := ""
	key = p.getKey(key)
	var args []interface{}
	args = append(args, key)
	args = append(args, start)
	args = append(args, end)
	err = p.doMGet("LRANGE", args, value)
	return
}

func (p *DaoRedis) LLen() (int64, error) {
	cmd := "LLEN"
	redisResource, err := p.getRedisConn()
	if err != nil {
		return 0, err
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	key := ""
	key = p.getKey(key)
	var result interface{}
	var errDo error
	var args []interface{}
	args = append(args, key)
	result, errDo = redisClient.Do(cmd, key)
	if errDo != nil {
		LogErrorw(LogNameRedis, "LLen run redis command error", errDo)
		return 0, errDo
	}
	if result == nil {
		return 0, nil
	}
	num, ok := result.(int64)
	if !ok {
		return 0, errors.New("result to int64 failed")
	}
	return num, nil
}

func (p *DaoRedis) LREM(count int, data interface{}) int {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return 0
	}
	defer daoPool.Put(redisResource, p.Persistent)
	key := ""
	key = p.getKey(key)
	redisClient := redisResource.(ResourceConn)
	result, errDo := redisClient.Do("LREM", key, count, data)
	if errDo != nil {
		LogErrorw(LogNameRedis, "LREM run redis command error", errDo)
		return 0
	}
	countRem, ok := result.(int)
	if !ok {
		LogErrorw(LogNameRedis, "LREM redis data convert to int error", errDo)
		return 0
	}
	return countRem
}

func (p *DaoRedis) LTRIM(start int, end int) (err error) {
	var redisResource pools.Resource
	redisResource, err = p.getRedisConn()
	if err != nil {
		return
	}
	defer daoPool.Put(redisResource, p.Persistent)
	key := ""
	key = p.getKey(key)
	redisClient := redisResource.(ResourceConn)
	_, err = redisClient.Do("LTRIM", key, start, end)
	if err != nil {
		LogErrorw(LogNameRedis, "LTRIM redis data convert to int error", err)
		return
	}
	return
}

func (p *DaoRedis) RPush(value interface{}) bool {
	return p.Push(value, false)
}

func (p *DaoRedis) LPush(value interface{}) bool {
	return p.Push(value, true)
}

func (p *DaoRedis) Push(value interface{}, isLeft bool) bool {
	var cmd string
	if isLeft {
		cmd = "LPUSH"
	} else {
		cmd = "RPUSH"
	}
	key := ""
	_, err := p.doSet(cmd, key, value, -1)
	if err != nil {
		return false
	}
	return true
}

func (p *DaoRedis) RPop(value interface{}) bool {
	return p.Pop(value, false)
}

func (p *DaoRedis) LPop(value interface{}) bool {
	return p.Pop(value, true)
}

func (p *DaoRedis) Pop(value interface{}, isLeft bool) bool {
	var cmd string
	if isLeft {
		cmd = "LPOP"
	} else {
		cmd = "RPOP"
	}
	key := ""
	_, err := p.doGet(cmd, key, value)
	if err == nil {
		return true
	} else {
		return false
	}
}

//list end

//pipeline start

func (p *DaoRedis) PipelineHGet(key []string, fields []interface{}, data []interface{}) error {
	var args [][]interface{}

	for k, v := range key {
		var arg []interface{}
		arg = append(arg, p.getKey(v))
		arg = append(arg, fields[k])
		args = append(args, arg)
	}

	err := p.pipeDoGet("HGET", args, data)

	return err
}

func (p *DaoRedis) pipeDoGet(cmd string, args [][]interface{}, value []interface{}) error {

	redisResource, err := p.getRedisConn()

	if err != nil {
		return err
	}
	defer daoPool.Put(redisResource, p.Persistent)

	redisClient := redisResource.(ResourceConn)

	for _, v := range args {
		if err := redisClient.Send(cmd, v...); err != nil {
			LogErrorw(LogNameRedis, "pipeDoGet Send returned error", err)
			return err
		}
	}
	if err := redisClient.Flush(); err != nil {
		LogErrorw(LogNameRedis, "pipeDoGet Flush returned error", err)
		return err
	}
	for k, _ := range args {
		result, err := redisClient.Receive()
		if err != nil {
			LogErrorw(LogNameRedis, "pipeDoGet Receive returned error", err)
			return err
		}
		if result == nil {
			value[k] = nil
			continue
		}
		if reflect.TypeOf(result).Kind() == reflect.Slice {

			byteResult := result.([]byte)
			strResult := string(byteResult)

			if strResult == "[]" {
				value[k] = nil
				continue
			}
		}

		errorJson := json.Unmarshal(result.([]byte), value[k])

		if errorJson != nil {
			LogErrorw(LogNameRedis, "pipeDoGet get command result error", errorJson)
			return errorJson
		}
	}

	return nil
}

//pipeline end

// Set集合Start
func (p *DaoRedis) SAdd(key string, argPs []interface{}) bool {
	redisResource, err := p.getRedisConn()

	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)

	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)

	redisClient := redisResource.(ResourceConn)

	_, errDo := redisClient.Do("SADD", args...)

	if errDo != nil {
		LogErrorw(LogNameRedis, "SAdd run redis SADD command error", errDo)
		return false
	}
	return true
}

func (p *DaoRedis) SIsMember(key string, arg interface{}) bool {
	redisResource, err := p.getRedisConn()

	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)

	key = p.getKey(key)

	redisClient := redisResource.(ResourceConn)
	reply, errDo := redisClient.Do("SISMEMBER", key, arg)

	if errDo != nil {
		LogErrorw(LogNameRedis, "SIsMember run redis SISMEMBER command error", errDo)
		return false
	}
	if code, ok := reply.(int64); ok && code == int64(1) {
		return true
	}
	return false
}

func (p *DaoRedis) SCard(key string) int64 {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return 0
	}
	defer daoPool.Put(redisResource, p.Persistent)
	key = p.getKey(key)
	redisClient := redisResource.(ResourceConn)
	reply, errDo := redisClient.Do("SCARD", key)
	if errDo != nil {
		LogErrorw(LogNameRedis, "SCARD run redis SCARD command error", errDo)
		return 0
	}
	return reply.(int64)
}

func (p *DaoRedis) SRem(key string, argPs []interface{}) bool {
	redisResource, err := p.getRedisConn()

	if err != nil {
		return false
	}
	defer daoPool.Put(redisResource, p.Persistent)

	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)

	redisClient := redisResource.(ResourceConn)

	_, errDo := redisClient.Do("SREM", args...)

	if errDo != nil {
		LogErrorw(LogNameRedis, "SRem run redis SREM command error", errDo)
		return false
	}
	return true
}

func (p *DaoRedis) SPop(key string, value interface{}) bool {
	_, err := p.doGet("SPOP", key, value)
	if err == nil {
		return true
	} else {
		return false
	}
}

func (p *DaoRedis) SMembers(key string, value interface{}) (err error) {
	var args []interface{}
	args = append(args, p.getKey(key))
	err = p.doMGet("SMEMBERS", args, value)
	return
}

func (p *DaoRedis) HGetAll(key string, data interface{}) error {
	var args []interface{}

	args = append(args, p.getKey(key))

	err := p.doMGet("HGETALL", args, data)

	return err
}

func (p *DaoRedis) HGetAllStringMap(key string) (err error, data map[string]string) {
	args := p.getKey(key)
	return p.doMGetStringMap("HGETALL", args)
}

func (p *DaoRedis) HGetAllIntMap(key string) (err error, data map[string]int) {
	args := p.getKey(key)
	return p.doMGetIntMap("HGETALL", args)
}

// GetPTtl：获取key的过期时间，单位为毫秒
// 如果key不存在返回-2
// 如果key存在，但是没有设置过期时间，返回-1
func (p *DaoRedis) GetPTtl(key string) (ttl int64, err error) {
	return p.doGetTtl("PTTL", key)
}

// GetTtl：获取key的过期时间，单位为秒
// 如果key不存在返回-2
// 如果key存在，但是没有设置过期时间，返回-1
func (p *DaoRedis) GetTtl(key string) (ttl int64, err error) {
	return p.doGetTtl("TTL", key)
}

func (p *DaoRedis) doGetTtl(cmd string, key string) (ttl int64, err error) {
	args := p.getKey(key)
	redisResource, err := p.getRedisConn()
	if err != nil {
		return 0, err
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	ttl, err = redis.Int64(redisClient.Do(cmd, args))
	if err != nil {
		LogErrorw(LogNameRedis, "doGetTtl run redis command error", err)
		return 0, err
	}
	return
}
