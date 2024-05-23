package redisCacher

import (
	"context"
	"dcron/server"
	"errors"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

// Conn Conn
var (
	Conn        IRedis
	ErrNil      = errors.New("redis: nil")
	IsMiniredis bool
)

type RedisPool struct {
	RedisConn *redis.Client
	Ctx       *context.Context
}

type IRedis interface {
	/** 取單一值 **/
	Get(key string) *redis.StringCmd
	/** 寫入 key-value 結構 **/
	Set(key string, val interface{}, ttl int64) error
	/** 檢查key是否存在, 不存在則寫入 **/
	SetNX(key string, val interface{}, expire int64) (bool, error)
	/** 刪除key底下資料 **/
	Del(key string) error
	/** 刪除keys底下資料 **/
	DelKeys(keys []string) error
	/** 取單一欄位資料 **/
	HGet(key string, field string) *redis.StringCmd
	/** 寫入一對多(key:fields[value]...)資料 **/
	HSet(key string, values map[string]interface{}, ttl int64) error
	/** 刪除一欄位資料 **/
	HDel(key string, filed string) error
	/** 取得整筆一對多資料內容 **/
	HGetAll(key string) (map[string]string, error)
	/** 列出條件式過濾結果 **/
	Scan(match string) (records []string, err error)
	/** 設定key失效時間 **/
	Expire(key string, expire int64) error
	/** TTL 搜尋該key expire時間 **/
	TTL(key string) (int, error)
	/** 清空資料 **/
	Flushall() (err error)
	/** 訂閱頻道的信息 **/
	Subscribe(callable func(channel string, data []byte) error, channel string)
	/** 發送信息到指定的頻道 **/
	Publish(channel, message string) error
}

func ConfigInit() {
	instance := server.GetServerInstance()
	redisCacher := instance.GetRedisCacher()
	ctx := instance.GetGracefulCtx()
	Conn = &RedisPool{
		RedisConn: redisCacher,
		Ctx:       ctx,
	}
}

func SetMiniredis() {
	m, _ := miniredis.Run()

	rdb := redis.NewClient(&redis.Options{
		Addr: m.Addr(),
	})
	ctx := context.Background()
	Conn = &RedisPool{
		RedisConn: rdb,
		Ctx:       &ctx,
	}
}

/** 取單一值 **/
func (r *RedisPool) Get(key string) *redis.StringCmd {
	return r.RedisConn.Get(*r.Ctx, key)
}

/** 寫入 key-value 結構 **/
func (r *RedisPool) Set(key string, val interface{}, ttl int64) error {
	return r.RedisConn.Set(*r.Ctx, key, val, time.Duration(ttl)*time.Second).Err()
}

/** 檢查key是否存在, 不存在則寫入 **/
func (r *RedisPool) SetNX(key string, val interface{}, ttl int64) (bool, error) {
	return r.RedisConn.SetNX(*r.Ctx, key, val, time.Duration(ttl)*time.Second).Result()
}

/** 刪除key底下資料 **/
func (r *RedisPool) Del(key string) error {
	return r.RedisConn.Del(*r.Ctx, key).Err()
}

/** 刪除keys底下資料 **/
func (r *RedisPool) DelKeys(keys []string) error {
	for _, key := range keys {
		err := r.RedisConn.Del(*r.Ctx, key).Err()
		if err != nil {
			return err
		}
	}
	return nil
}

/** 取單一欄位資料 **/
func (r *RedisPool) HGet(key string, field string) *redis.StringCmd {
	return r.RedisConn.HGet(*r.Ctx, key, field)
}

/** 寫入一對多(key:fields...)資料 **/
func (r *RedisPool) HSet(key string, values map[string]interface{}, ttl int64) (err error) {
	var field []interface{}
	for key, v := range values {
		field = append(field, key)
		field = append(field, v)
	}
	err = r.RedisConn.HSet(*r.Ctx, key, field...).Err()
	if ttl > 0 {
		r.RedisConn.Expire(*r.Ctx, key, time.Duration(ttl)*time.Second)
	}

	return err
}

/** 刪除一欄位資料 **/
func (r *RedisPool) HDel(key string, filed string) (err error) {
	return r.RedisConn.HDel(*r.Ctx, key, filed).Err()
}

/** 取得整筆一對多資料內容 **/
func (r *RedisPool) HGetAll(key string) (vals map[string]string, err error) {
	data, err := r.RedisConn.Do(*r.Ctx, "HGETALL", key).Result()
	if err != nil {
		return vals, err
	}
	// 將取得的資料格式轉換成 map[string]string
	result := make(map[string]string)
	for key, value := range data.(map[interface{}]interface{}) {
		result[key.(string)] = value.(string)
	}

	return result, nil
}

/** 列出條件式過濾結果 **/
func (r *RedisPool) Scan(match string) (records []string, err error) {

	iter := r.RedisConn.Scan(*r.Ctx, 0, match, 1000).Iterator()
	for iter.Next(*r.Ctx) {
		records = append(records, iter.Val())
	}
	if err := iter.Err(); err != nil {
		return records, err
	}

	return records, nil
}

/** 設定key失效時間 **/
func (r *RedisPool) Expire(key string, expire int64) error {
	return r.RedisConn.Expire(*r.Ctx, key, time.Duration(expire)*time.Second).Err()
}

/** TTL 搜尋該key expire時間 **/
func (r *RedisPool) TTL(key string) (int, error) {
	return r.RedisConn.Do(*r.Ctx, "TTL", key).Int()
}

/** 清空資料 **/
func (r *RedisPool) Flushall() (err error) {
	return r.RedisConn.Do(*r.Ctx, "flushall").Err()
}

func (r *RedisPool) Subscribe(callable func(channel string, data []byte) error, channel string) {
	pubSub := r.RedisConn.Subscribe(*r.Ctx, channel)
	defer pubSub.Close()

	// 處理消息
	ch := pubSub.Channel()
	go func() {
		for msg := range ch {
			//如果redis斷線 會自己重連
			go callable(msg.Channel, []byte(msg.Payload))
		}
	}()
}

func (r *RedisPool) Publish(channel, message string) error {
	return r.RedisConn.Publish(*r.Ctx, channel, message).Err()
}
