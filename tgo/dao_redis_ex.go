package tgo

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/youtube/vitess/go/pools"
	"reflect"
	"strings"
	"sync"
)

type DaoRedisEx struct {
	KeyName          string
	Persistent       bool // 持久化key
	ExpireSecond     int  // 默认过期时间，单实例有效
	tempExpireSecond int  // 临时默认过期时间，单条命令有效
}

type OpOptionEx func(*DaoRedisEx)

// WithExpire 设置超时时间
func WithExpire(expire int) OpOptionEx {
	return func(p *DaoRedisEx) { p.tempExpireSecond = expire }
}

// applyOpts 应用扩展属性
func (p *DaoRedisEx) applyOpts(opts []OpOptionEx) {
	for _, opt := range opts {
		opt(p)
	}
}

// resetTempExpireSecond 重置临时过期时间
func (p *DaoRedisEx) resetTempExpireSecond() {
	p.tempExpireSecond = 0
}

// getExpire 获取过期时间
func (p *DaoRedisEx) getExpire(expire int) int {
	var expireSecond int
	if expire != 0 {
		expireSecond = expire
	} else if p.tempExpireSecond != 0 {
		expireSecond = p.tempExpireSecond
	} else if p.ExpireSecond != 0 {
		expireSecond = p.ExpireSecond
	} else {
		cacheConfig := ConfigCacheGetRedisWithConn()
		expireSecond = cacheConfig.Expire
	}

	if expireSecond < 0 {
		expireSecond = -1
	}
	return expireSecond
}

//获取redis连接
func (p *DaoRedisEx) getRedisConn() (pools.Resource, error) {
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

func (p *DaoRedisEx) getKey(key string) string {
	cacheConfig := ConfigCacheGetRedisWithConn()
	prefixRedis := cacheConfig.Prefix
	if strings.Trim(key, " ") == "" {
		return fmt.Sprintf("%s:%s", prefixRedis, p.KeyName)
	}
	return fmt.Sprintf("%s:%s:%s", prefixRedis, p.KeyName, key)
}

func (p *DaoRedisEx) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return nil, err
	}
	defer daoPool.Put(redisResource, p.Persistent)
	defer p.resetTempExpireSecond()
	redisClient := redisResource.(ResourceConn)
	return redisClient.Do(commandName, args...)
}

func (p *DaoRedisEx) doSet(cmd string, key string, value interface{}, expire int, fields ...string) (interface{}, error) {
	data, errJson := json.Marshal(value)
	if errJson != nil {
		UtilLogErrorf("redis %s marshal data to json:%s", cmd, errJson.Error())
		return nil, errJson
	}
	key = p.getKey(key)
	expire = p.getExpire(expire)
	var reply interface{}
	var errDo error
	if len(fields) == 0 {
		if expire > 0 && strings.ToUpper(cmd) == "SET" {
			reply, errDo = p.do(cmd, key, data, "ex", expire)
		} else {
			reply, errDo = p.do(cmd, key, data)
		}
	} else {
		field := fields[0]
		reply, errDo = p.do(cmd, key, field, data)
	}
	if errDo != nil {
		UtilLogErrorf("run redis command %s failed:error:%s,key:%s,fields:%v,data:%v", cmd, errDo.Error(), key, fields, value)
		return nil, errDo
	}
	//set expire
	//if expire > 0 && strings.ToUpper(cmd) != "SET" {
	//	_, errExpire := p.do("EXPIRE", key, expire)
	//	if errExpire != nil {
	//		UtilLogErrorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", errExpire.Error(), key, expire)
	//	}
	//}
	return reply, errDo
}

func (p *DaoRedisEx) doSetNX(cmd string, key string, value interface{}, expire int, field ...string) (num int64, err error) {
	var (
		reply interface{}
		ok    bool
	)
	reply, err = p.doSet(cmd, key, value, expire, field...)
	if err != nil {
		return
	}
	num, ok = reply.(int64)
	if !ok {
		msg := fmt.Sprintf("HSetNX reply to int failed,key:%v,field:%v", key, field)
		UtilLogErrorf(msg)
		err = errors.New(msg)
		return
	}
	return
}

func (p *DaoRedisEx) doMSet(cmd string, key string, value map[string]interface{}) (interface{}, error) {
	var args []interface{}
	if key != "" {
		key = p.getKey(key)
		args = append(args, key)
	}
	for k, v := range value {
		data, errJson := json.Marshal(v)
		if errJson != nil {
			UtilLogErrorf("redis %s marshal data: %v to json:%s", cmd, v, errJson.Error())
			return nil, errJson
		}
		if key == "" {
			args = append(args, p.getKey(k), data)
		} else {
			args = append(args, k, data)
		}
	}
	var reply interface{}
	var errDo error
	reply, errDo = p.do(cmd, args...)
	if errDo != nil {
		UtilLogErrorf("run redis command %s failed:error:%s,key:%s,value:%v", cmd, errDo.Error(), key, value)
		return nil, errDo
	}
	return reply, errDo
}

func (p *DaoRedisEx) doGet(cmd string, key string, value interface{}, fields ...string) (bool, error) {
	key = p.getKey(key)
	var result interface{}
	var errDo error
	var args []interface{}
	args = append(args, key)
	for _, f := range fields {
		args = append(args, f)
	}
	result, errDo = p.do(cmd, args...)
	if errDo != nil {
		UtilLogErrorf("run redis %s command failed: error:%s,key:%s,fields:%v", cmd, errDo.Error(), key, fields)
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
		UtilLogErrorf("get %s command result failed:%s", cmd, errorJson.Error())
		return false, errorJson
	}
	return true, nil
}

func (p *DaoRedisEx) doMGet(cmd string, args []interface{}, value interface{}) error {
	refValue := reflect.ValueOf(value)
	if refValue.Kind() != reflect.Ptr || refValue.Elem().Kind() != reflect.Slice || refValue.Elem().Type().Elem().Kind() != reflect.Ptr {
		return errors.New(fmt.Sprintf("value is not *[]*object:  %v", refValue.Elem().Type().Elem().Kind()))
	}
	refSlice := refValue.Elem()
	refItem := refSlice.Type().Elem()
	result, errDo := redis.ByteSlices(p.do(cmd, args...))
	if errDo != nil {
		UtilLogErrorf("run redis %s command failed: error:%s,args:%v", cmd, errDo.Error(), args)
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
					UtilLogErrorf("%s command result failed:%s", cmd, errorJson.Error())
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

func (p *DaoRedisEx) doMGetGo(keys []string, value interface{}) error {
	var (
		args     []interface{}
		keysMap  sync.Map
		keysLen  int
		rDo      interface{}
		errDo    error
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
			rDo, errDo = p.do("GET", getK)
			if errDo != nil {
				UtilLogErrorf("run redis GET command failed: error:%s,args:%v", errDo.Error(), getK)
				resultDo = false
			} else {
				keysMap.Store(getK, rDo)
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
				UtilLogErrorf("GET command result failed:%s", errorJson.Error())
				return errorJson
			}
			refSlice.Set(reflect.Append(refSlice, item.Elem()))
		} else {
			refSlice.Set(reflect.Append(refSlice, reflect.Zero(refItem)))
		}
	}
	return nil
}

func (p *DaoRedisEx) doMGetStringMap(cmd string, args ...interface{}) (err error, data map[string]string) {
	data, err = redis.StringMap(p.do(cmd, args...))
	if err != nil {
		UtilLogErrorf("run redis %s command failed: error:%v, args:%v", cmd, err, args)
		return err, nil
	}
	return
}

func (p *DaoRedisEx) doMGetIntMap(cmd string, args ...interface{}) (err error, data map[string]int) {
	data, err = redis.IntMap(p.do(cmd, args...))
	if err != nil {
		UtilLogErrorf("run redis %s command failed: error:%v, args:%v", cmd, err, args)
		return err, nil
	}
	return
}

func (p *DaoRedisEx) doIncr(cmd string, key string, value int, expire int, fields ...string) (num int64, err error) {
	var (
		data interface{}
		ok   bool
	)
	expire = p.getExpire(expire)
	key = p.getKey(key)
	if len(fields) == 0 {
		data, err = p.do(cmd, key, value)
	} else {
		field := fields[0]
		data, err = p.do(cmd, key, field, value)
	}
	if err != nil {
		UtilLogErrorf("run redis %s command failed: error:%s,key:%s,fields:%v,value:%d", cmd, err.Error(), key, fields, value)
		return
	}
	num, ok = data.(int64)
	if !ok {
		msg := fmt.Sprintf("get %s command result failed:%v ,is %v", cmd, data, reflect.TypeOf(data))
		UtilLogErrorf(msg)
		err = errors.New(msg)
		return
	}
	if expire > 0 {
		_, errExpire := p.do("EXPIRE", key, expire)
		if errExpire != nil {
			UtilLogErrorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", errExpire.Error(), key, expire)
		}
	}
	return
}

func (p *DaoRedisEx) doIncrNX(cmd string, key string, value int, expire int) (num int64, err error) {
	var (
		data interface{}
		ok   bool
	)
	expire = p.getExpire(expire)
	key = p.getKey(key)
	redisResource, err := p.getRedisConn()
	if err != nil {
		return
	}
	defer daoPool.Put(redisResource, p.Persistent)
	defer p.resetTempExpireSecond()
	redisClient := redisResource.(ResourceConn)
	luaCmd := "local ck=redis.call('EXISTS', KEYS[1]); if (ck == 1) then return redis.call('INCRBY', KEYS[1], ARGV[1]) else return 'null' end"
	data, err = redisClient.Do("EVAL", luaCmd, 1, key, value)
	if err != nil {
		UtilLogErrorf("run redis %s command failed: error:%s,key:%s,value:%d", cmd, err.Error(), key, value)
		return
	}
	var luaRet string
	if luaRet, ok = data.(string); ok { // key 不存在
		if luaRet == "null" {
			err = errors.New("INCRBY key not exists")
			LogErrorw(LogNameRedis, "doIncrNX", err)
			return
		}
	}
	num, ok = data.(int64)
	if !ok {
		msg := fmt.Sprintf("get %s command result failed:%v ,is %v", cmd, data, reflect.TypeOf(data))
		UtilLogErrorf(msg)
		err = errors.New(msg)
		return
	}
	if expire > 0 {
		_, errExpire := p.do("EXPIRE", key, expire)
		if errExpire != nil {
			UtilLogErrorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", errExpire.Error(), key, expire)
		}
	}
	return
}

func (p *DaoRedisEx) doDel(cmd string, data ...interface{}) error {
	_, errDo := p.do(cmd, data...)
	if errDo != nil {
		UtilLogErrorf("run redis %s command failed: error:%s,data:%v", cmd, errDo.Error(), data)
	}
	return errDo
}

/*基础结束*/
func (p *DaoRedisEx) Set(key string, value interface{}, ops ...OpOptionEx) (err error) {
	p.applyOpts(ops)
	_, err = p.doSet("SET", key, value, 0)
	return
}

//MSet mset
func (p *DaoRedisEx) MSet(datas map[string]interface{}) error {
	_, err := p.doMSet("MSET", "", datas)
	return err
}

//SetEx setex
func (p *DaoRedisEx) SetEx(key string, value interface{}, expire int) error {
	_, err := p.doSet("SET", key, value, expire)
	return err
}

//Expire expire
func (p *DaoRedisEx) Expire(key string, expire int) error {
	key = p.getKey(key)
	_, err = p.do("EXPIRE", key, expire)
	if err != nil {
		UtilLogErrorf("run redis EXPIRE command failed: error:%s,key:%s,time:%d", err.Error(), key, expire)
		return err
	}
	return nil
}

func (p *DaoRedisEx) Get(key string, data interface{}) error {
	_, err := p.doGet("GET", key, data)
	return err
}

// 返回 1. key是否存在 2. error
func (p *DaoRedisEx) GetRaw(key string, data interface{}) (bool, error) {
	return p.doGet("GET", key, data)
}

func (p *DaoRedisEx) MGet(keys []string, data interface{}) error {
	var args []interface{}
	for _, v := range keys {
		args = append(args, p.getKey(v))
	}
	err := p.doMGet("MGET", args, data)
	return err
}

//封装mget通过go并发get
func (p *DaoRedisEx) MGetGo(keys []string, data interface{}) error {
	err := p.doMGetGo(keys, data)
	return err
}

func (p *DaoRedisEx) Incr(key string, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncr("INCRBY", key, 1, 0)
}

func (p *DaoRedisEx) IncrBy(key string, value int, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncr("INCRBY", key, value, 0)
}

// 存在key 才会自增
func (p *DaoRedisEx) IncrNX(key string, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncrNX("INCRBY", key, 1, 0)
}

// 存在key 才会更新数值
func (p *DaoRedisEx) IncrByNX(key string, value int, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncrNX("INCRBY", key, value, 0)
}

// 针对key进行一定时间内访问次数的限流
func (p *DaoRedisEx) Limiter(key string, expire int, max int) (allow bool, err error) {
	var (
		data interface{}
		ok   bool
	)
	if expire <= 0 {
		err = errors.New("limiter expire must gt 0")
	}
	key = p.getKey(key)
	redisResource, err := p.getRedisConn()
	if err != nil {
		return
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	luaCmd := `
local times = redis.call('INCR', KEYS[1])

if times == 1 then
    redis.call('expire', KEYS[1], ARGV[1])
end

if times > tonumber(ARGV[2]) then
    return 0
end

return 1
`
	data, err = redisClient.Do("EVAL", luaCmd, 1, key, expire, max)
	if err != nil {
		UtilLogErrorf("run redis %s command failed: error:%s,key:%s,expire:%d,max:%d", luaCmd, err.Error(), key, expire, max)
		return
	}
	var ret int64
	ret, ok = data.(int64)
	if !ok {
		msg := fmt.Sprintf("get %s command result failed:%v ,is %v", luaCmd, data, reflect.TypeOf(data))
		UtilLogErrorf(msg)
		err = errors.New(msg)
		return
	}
	if ret == 1 {
		allow = true
	}
	return
}

func (p *DaoRedisEx) SetEXNX(key string, value interface{}) (string, error) {
	redisResource, err := p.getRedisConn()
	if err != nil {
		return "", err
	}
	defer daoPool.Put(redisResource, p.Persistent)
	redisClient := redisResource.(ResourceConn)
	if err != nil {
		return "", err
	}
	key = p.getKey(key)
	reply, err := redis.String(redisClient.Do("SET", key, value, "EX", p.ExpireSecond, "NX"))
	if err == redis.ErrNil {
		err = nil
	}
	return reply, err
}

func (p *DaoRedisEx) SetNX(key string, value interface{}, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doSetNX("SETNX", key, value, 0)
}

func (p *DaoRedisEx) SetNXNoExpire(key string, value interface{}) (int64, error) {
	return p.doSetNX("SETNX", key, value, -1)
}

func (p *DaoRedisEx) Del(key string) error {
	key = p.getKey(key)
	err := p.doDel("DEL", key)
	return err
}

func (p *DaoRedisEx) MDel(key ...string) error {
	var keys []interface{}
	for _, v := range key {
		keys = append(keys, p.getKey(v))
	}
	err := p.doDel("DEL", keys...)
	return err
}

func (p *DaoRedisEx) Exists(key string) (bool, error) {
	key = p.getKey(key)
	data, err := p.do("EXISTS", key)
	if err != nil {
		UtilLogErrorf("run redis EXISTS command failed: error:%s,key:%s", err.Error(), key)
		return false, err
	}
	count, result := data.(int64)
	if !result {
		err := errors.New(fmt.Sprintf("get EXISTS command result failed:%v ,is %v", data, reflect.TypeOf(data)))
		UtilLogErrorf(err.Error())
		return false, err
	}
	if count == 1 {
		return true, nil
	}

	return false, nil
}

//hash start
func (p *DaoRedisEx) HIncrby(key string, field string, value int, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doIncr("HINCRBY", key, value, 0, field)
}

func (p *DaoRedisEx) HGet(key string, field string, value interface{}) error {
	_, err := p.doGet("HGET", key, value, field)
	return err
}

//HGetRaw 返回 1. key是否存在 2. error
func (p *DaoRedisEx) HGetRaw(key string, field string, value interface{}) (bool, error) {
	return p.doGet("HGET", key, value, field)
}

func (p *DaoRedisEx) HMGet(key string, fields []interface{}, data interface{}) error {
	var args []interface{}
	args = append(args, p.getKey(key))
	for _, v := range fields {
		args = append(args, v)
	}
	err := p.doMGet("HMGET", args, data)
	return err
}

func (p *DaoRedisEx) HSet(key string, field string, value interface{}, ops ...OpOptionEx) error {
	p.applyOpts(ops)
	_, err := p.doSet("HSET", key, value, 0, field)
	return err
}

func (p *DaoRedisEx) HSetNX(key string, field string, value interface{}, ops ...OpOptionEx) (int64, error) {
	p.applyOpts(ops)
	return p.doSetNX("HSETNX", key, value, 0, field)
}

//HMSet value是filed:data
func (p *DaoRedisEx) HMSet(key string, value map[string]interface{}) error {
	_, err := p.doMSet("HMSet", key, value)
	return err
}

func (p *DaoRedisEx) HLen(key string, data *int) error {
	key = p.getKey(key)
	resultData, err := p.do("HLEN", key)
	if err != nil {
		UtilLogErrorf("run redis HLEN command failed: error:%s,key:%s", err.Error(), key)
		return err
	}
	length, b := resultData.(int64)
	if !b {
		msg := fmt.Sprintf("redis data convert to int64 failed:%v", resultData)
		UtilLogErrorf(msg)
		err = errors.New(msg)
		return err
	}
	*data = int(length)
	return nil
}

func (p *DaoRedisEx) HDel(key string, data ...interface{}) error {
	var args []interface{}
	key = p.getKey(key)
	args = append(args, key)
	for _, item := range data {
		args = append(args, item)
	}
	err := p.doDel("HDEL", args...)
	if err != nil {
		UtilLogErrorf("run redis HDEL command failed: error:%s,key:%s,data:%v", err.Error(), key, data)
	}
	return err
}

func (p *DaoRedisEx) HExists(key string, field string) (bool, error) {
	key = p.getKey(key)
	data, err := p.do("HEXISTS", key, field)
	if err != nil {
		UtilLogErrorf("run redis HEXISTS command failed: error:%s,key:%s", err.Error(), key)
		return false, err
	}
	count, result := data.(int64)
	if !result {
		err := errors.New(fmt.Sprintf("get HEXISTS command result failed:%v ,is %v", data, reflect.TypeOf(data)))
		UtilLogErrorf(err.Error())
		return false, err
	}
	if count == 1 {
		return true, nil
	}
	return false, nil
}

// hash end

// sorted set start
func (p *DaoRedisEx) ZAdd(key string, score interface{}, data interface{}) error {
	key = p.getKey(key)
	_, errDo := p.do("ZADD", key, score, data)
	if errDo != nil {
		UtilLogErrorf("run redis ZADD command failed: error:%s,key:%s,score:%d,data:%v", errDo.Error(), key, score, data)
	}
	return errDo
}

func (p *DaoRedisEx) ZCard(key string) (data int, err error) {
	key = p.getKey(key)
	var reply interface{}
	reply, err = p.do("ZCARD", key)
	if err != nil {
		UtilLogErrorf("run redis ZCARD command failed: error:%v,key:%s", err, key)
		return
	}
	if v, ok := reply.(int64); ok {
		data = int(v)
		return
	} else {
		err = errors.New(fmt.Sprintf("ZCard get replay is not int64:%v", reply))
		return
	}
}

func (p *DaoRedisEx) ZCount(key string, min, max int) (data int, err error) {
	key = p.getKey(key)
	var reply interface{}
	reply, err = p.do("ZCOUNT", key, min, max)
	if err != nil {
		UtilLogErrorf("run redis ZCOUNT command failed: error:%v,key:%s,min:%d,max:%d", err, key, min, max)
		return
	}
	if v, ok := reply.(int64); ok {
		data = int(v)
		return
	} else {
		err = errors.New(fmt.Sprintf("ZCount get replay is not int64:%v", reply))
		return
	}
}

func (p *DaoRedisEx) ZIncrBy(key string, increment int, member interface{}) error {
	key = p.getKey(key)
	_, errDo := p.do("ZINCRBY", key, increment, member)
	if errDo != nil {
		UtilLogErrorf("run redis ZINCRBY command failed: error:%s,key:%s,increment:%d,data:%v", errDo.Error(), key, increment, member)
	}
	return errDo
}

// sorted set start
func (p *DaoRedisEx) ZAddM(key string, value map[string]interface{}) error {
	_, err := p.doMSet("ZADD", key, value)
	return err
}

func (p *DaoRedisEx) ZGetByScore(key string, sort bool, start int, end int, value interface{}) error {
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

func (p *DaoRedisEx) ZGet(key string, sort bool, start int, end int, value interface{}) error {
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

func (p *DaoRedisEx) ZGetWithScores(key string, sort bool, start int, end int) (err error, data map[string]string) {
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

func (p *DaoRedisEx) ZRank(key string, member string, sort bool) (error, int) {
	var cmd string
	if sort {
		cmd = "ZRANK"
	} else {
		cmd = "ZREVRANK"
	}
	key = p.getKey(key)
	result, errDo := p.do(cmd, key, member)
	if errDo != nil {
		UtilLogErrorf("run redis %s command failed: error:%s,key:%s,increment:%d,data:%v", cmd, errDo.Error(), key, member)
		return errDo, 0
	}
	if v, ok := result.(int64); ok {
		return nil, int(v)
	} else {
		msg := fmt.Sprintf("run redis %s command result failed: key:%v,result:%v", cmd, key, result)
		UtilLogErrorf(msg)
		err = errors.New(msg)
		return err, 0
	}
}

func (p *DaoRedisEx) ZScore(key string, member string, value interface{}) error {
	cmd := "ZSCORE"
	_, err := p.doGet(cmd, key, value, member)
	return err
}

func (p *DaoRedisEx) ZRevRange(key string, start int, end int, value interface{}) error {
	return p.ZGet(key, false, start, end, value)
}

func (p *DaoRedisEx) ZRem(key string, data ...interface{}) error {
	var args []interface{}
	key = p.getKey(key)
	args = append(args, key)
	for _, item := range data {
		args = append(args, item)
	}
	err := p.doDel("ZREM", args...)
	return err
}

//list start

func (p *DaoRedisEx) LRange(start int, end int, value interface{}) (err error) {
	key := ""
	key = p.getKey(key)
	var args []interface{}
	args = append(args, key)
	args = append(args, start)
	args = append(args, end)
	err = p.doMGet("LRANGE", args, value)
	return
}

func (p *DaoRedisEx) LLen() (int64, error) {
	cmd := "LLEN"
	key := ""
	key = p.getKey(key)
	var result interface{}
	var errDo error
	var args []interface{}
	args = append(args, key)
	result, errDo = p.do(cmd, key)
	if errDo != nil {
		UtilLogErrorf("run redis %s command failed: error:%s,key:%s", cmd, errDo.Error(), key)
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

func (p *DaoRedisEx) LREM(count int, data interface{}) (error, int) {
	key := ""
	key = p.getKey(key)
	result, errDo := p.do("LREM", key, count, data)
	if errDo != nil {
		UtilLogErrorf("run redis command LREM failed: error:%s,key:%s,count:%d,data:%v", errDo.Error(), key, count, data)
		return errDo, 0
	}
	countRem, ok := result.(int)
	if !ok {
		msg := fmt.Sprintf("redis data convert to int failed:%v", result)
		UtilLogErrorf(msg)
		err = errors.New(msg)
		return err, 0
	}
	return nil, countRem
}

func (p *DaoRedisEx) LTRIM(start int, end int) (err error) {
	key := ""
	key = p.getKey(key)
	_, err = p.do("LTRIM", key, start, end)
	if err != nil {
		UtilLogErrorf("run redis command LTRIM failed: error:%v,key:%s,start:%d,end:%d", err, key, start, end)
		return
	}
	return
}

func (p *DaoRedisEx) RPush(value interface{}) error {
	return p.Push(value, false)
}

func (p *DaoRedisEx) LPush(value interface{}) error {
	return p.Push(value, true)
}

func (p *DaoRedisEx) Push(value interface{}, isLeft bool) error {
	var cmd string
	if isLeft {
		cmd = "LPUSH"
	} else {
		cmd = "RPUSH"
	}
	key := ""
	_, err := p.doSet(cmd, key, value, -1)
	return err
}

func (p *DaoRedisEx) RPop(value interface{}) error {
	return p.Pop(value, false)
}

func (p *DaoRedisEx) LPop(value interface{}) error {
	return p.Pop(value, true)
}

func (p *DaoRedisEx) BLpop(value interface{}, timeout int) error {
	key := p.getKey("")
	var result interface{}
	var errDo error
	result, errDo = p.do("BLPOP", key, timeout)
	if errDo != nil {
		//UtilLogErrorf("run redis BLPOP command failed: error:%s,key:%s", errDo.Error(), key)
		return errDo
	}
	if result == nil {
		value = nil
		return errDo
	}
	results, err := redis.ByteSlices(result, errDo)
	if err != nil {
		//UtilLogErrorf("get BLPOP command redis.ByteSlices failed:%s", err.Error())
	}
	if len(results) == 2 {
		errorJson := json.Unmarshal(results[1], value)
		if errorJson != nil {
			if reflect.TypeOf(value).Kind() == reflect.Ptr && reflect.TypeOf(value).Elem().Kind() == reflect.String {
				var strValue string
				strValue = string(result.([]byte))
				v := value.(*string)
				*v = strValue
				value = v
				return nil
			}
			//UtilLogErrorf("get BLPOP command result failed:%s", errorJson.Error())
			return errorJson
		}
	} else {
		value = nil
		return errDo
	}
	return errDo
}

func (p *DaoRedisEx) Pop(value interface{}, isLeft bool) error {
	var cmd string
	if isLeft {
		cmd = "LPOP"
	} else {
		cmd = "RPOP"
	}
	key := ""
	_, err := p.doGet(cmd, key, value)
	return err
}

//list end

// Set集合Start
func (p *DaoRedisEx) SAdd(key string, argPs []interface{}) error {
	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)
	_, errDo := p.do("SADD", args...)
	if errDo != nil {
		UtilLogErrorf("run redis SADD command failed: error:%s,key:%s,args:%v", errDo.Error(), key, args)
	}
	return errDo
}

func (p *DaoRedisEx) SIsMember(key string, arg interface{}) (b bool, err error) {
	key = p.getKey(key)
	var reply interface{}
	reply, err = p.do("SISMEMBER", key, arg)
	if err != nil {
		UtilLogErrorf("run redis SISMEMBER command failed: error:%v,key:%s,member:%s", err, key, arg)
		return
	}
	if code, ok := reply.(int64); ok && code == int64(1) {
		b = true
	}
	return
}

func (p *DaoRedisEx) SCard(key string) int64 {
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

func (p *DaoRedisEx) SRem(key string, argPs []interface{}) error {
	args := make([]interface{}, len(argPs)+1)
	args[0] = p.getKey(key)
	copy(args[1:], argPs)
	_, errDo := p.do("SREM", args...)
	if errDo != nil {
		UtilLogErrorf("run redis SREM command failed: error:%s,key:%s,member:%s", errDo.Error(), key, args)
	}
	return errDo
}

func (p *DaoRedisEx) SPop(key string, value interface{}) error {
	_, err := p.doGet("SPOP", key, value)
	return err
}

func (p *DaoRedisEx) SMembers(key string, value interface{}) (err error) {
	var args []interface{}
	args = append(args, p.getKey(key))
	err = p.doMGet("SMEMBERS", args, value)
	return
}

func (p *DaoRedisEx) HGetAll(key string, data interface{}) error {
	var args []interface{}

	args = append(args, p.getKey(key))

	err := p.doMGet("HGETALL", args, data)

	return err
}

func (p *DaoRedisEx) HGetAllStringMap(key string) (err error, data map[string]string) {
	args := p.getKey(key)
	return p.doMGetStringMap("HGETALL", args)
}

func (p *DaoRedisEx) HGetAllIntMap(key string) (err error, data map[string]int) {
	args := p.getKey(key)
	return p.doMGetIntMap("HGETALL", args)
}

// GetPTtl：获取key的过期时间，单位为毫秒
// 如果key不存在返回-2
// 如果key存在，但是没有设置过期时间，返回-1
func (p *DaoRedisEx) GetPTtl(key string) (ttl int64, err error) {
	return p.doGetTtl("PTTL", key)
}

// GetTtl：获取key的过期时间，单位为秒
// 如果key不存在返回-2
// 如果key存在，但是没有设置过期时间，返回-1
func (p *DaoRedisEx) GetTtl(key string) (ttl int64, err error) {
	return p.doGetTtl("TTL", key)
}

func (p *DaoRedisEx) doGetTtl(cmd string, key string) (ttl int64, err error) {
	args := p.getKey(key)
	ttl, err = redis.Int64(p.do(cmd, args))
	if err != nil {
		LogErrorw(LogNameRedis, "doGetTtl run redis command error", err)
		return 0, err
	}
	return
}
