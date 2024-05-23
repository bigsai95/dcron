package redisCacher

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func setRedisTest() {
	SetMiniredis()

	// 清空 Redis 中的資料
	// Conn.Flushall()
}

func TestBasic(t *testing.T) {
	setRedisTest()
	t.Run("case 1", func(t *testing.T) {
		// 设置测试数据
		key := "test_key"
		expectedValue := "test_value"
		err := Conn.Set(key, expectedValue, 0)
		assert.Equal(t, err, nil)

		value, err := Conn.Get(key).Result()
		assert.Equal(t, err, nil)
		assert.Equal(t, value, expectedValue)

		err = Conn.Del(key)
		assert.Equal(t, err, nil)
	})
	t.Run("case 2", func(t *testing.T) {
		// GET
		key := "Test_Get:TEAM_NAME:test:Test:test"
		err := Conn.Get(key).Err()
		assert.Equal(t, err.Error(), ErrNil.Error())

		// SET
		key = "Test:Set:GROUP:1234567890"
		err = Conn.Set(key, "here", 12)
		assert.Equal(t, err, nil)

		res, err := Conn.Get(key).Result()
		assert.Equal(t, res, "here")
		assert.Equal(t, err, nil)

		err = Conn.Del(key)
		assert.Equal(t, err, nil)

		err = Conn.Get(key).Err()
		assert.Equal(t, err.Error(), ErrNil.Error())

		err = Conn.Del(key)
		assert.Equal(t, err, nil)
	})

	t.Run("case 3", func(t *testing.T) {
		key := "test_key"
		val := "test_value"
		ttl := int64(3600)

		// key 不存在
		result, err := Conn.SetNX(key, val, ttl)
		assert.Nil(t, err)
		assert.True(t, result, "SetNX should have returned true")

		// key 存在
		result, err = Conn.SetNX(key, val, ttl)
		assert.Nil(t, err)
		assert.False(t, result, "SetNX should have returned false")

		err = Conn.Del(key)
		assert.Equal(t, err, nil)
	})
}

func TestHBasic(t *testing.T) {
	setRedisTest()
	t.Run("case 1", func(t *testing.T) {
		key := "test_hash"
		field := "test_field"
		expectedValue := "test_value"
		values := map[string]interface{}{field: expectedValue}

		err := Conn.HSet(key, values, 100)
		assert.Equal(t, err, nil)

		redisCmd := Conn.HGet(key, field)
		value := redisCmd.Val()
		err = redisCmd.Err()
		assert.Equal(t, err, nil)
		assert.Equal(t, value, expectedValue)

		err = Conn.HDel(key, field)
		assert.Equal(t, err, nil)

		redisCmd = Conn.HGet(key, field)
		value = redisCmd.Val()
		err = redisCmd.Err()
		assert.Equal(t, value, "")
		assert.Equal(t, err.Error(), ErrNil.Error())

		err = Conn.Del(key)
		assert.Equal(t, err, nil)
	})
	t.Run("case 2", func(t *testing.T) {
		key := "Test:HGet:GROUP:1234567890"

		redisCmd := Conn.HGet(key, "here")
		err := redisCmd.Err()
		assert.Equal(t, err.Error(), ErrNil.Error())

		// HSET
		key = "Test:HSet:GROUP:1234567890"
		fields := make(map[string]interface{})
		fields["ff1"] = "ff1"
		fields["ff2"] = 20000
		fields["ff3"] = true
		fields["ff4"] = "ff4"
		err = Conn.HSet(key, fields, 12)
		assert.Equal(t, err, nil)

		redisCmd = Conn.HGet(key, "ff1")
		ff1 := redisCmd.Val()
		err = redisCmd.Err()
		assert.Equal(t, ff1, "ff1")
		assert.Equal(t, err, nil)

		redisCmd = Conn.HGet(key, "ff2")
		ff2 := redisCmd.Val()
		err = redisCmd.Err()
		assert.Equal(t, ff2, "20000")
		assert.Equal(t, err, nil)

		ff3, err := Conn.HGet(key, "ff3").Bool()
		assert.Equal(t, ff3, true)
		assert.Equal(t, err, nil)

		redisCmd = Conn.HGet(key, "ff4")
		ff4 := redisCmd.Val()
		err = redisCmd.Err()
		assert.Equal(t, ff4, "ff4")
		assert.Equal(t, err, nil)

		err = Conn.HDel(key, "ff4")
		assert.Equal(t, err, nil)

		redisCmd = Conn.HGet(key, "ff4")
		ff4 = redisCmd.Val()
		err = redisCmd.Err()
		assert.Equal(t, ff4, "")
		assert.Equal(t, err.Error(), ErrNil.Error())

		err = Conn.Del(key)
		assert.Equal(t, err, nil)
	})
}

func TestHGetAll(t *testing.T) {
	setRedisTest()
	t.Run("case 1", func(t *testing.T) {
		key := "Test:HGetAll:GROUP:1234567890"
		fields := make(map[string]interface{})
		fields["ff1"] = "ff1"
		fields["ff2"] = 20000
		fields["ff3"] = true
		fields["ff4"] = "ff4"
		err := Conn.HSet(key, fields, 12)
		assert.Equal(t, err, nil)

		maps, err := Conn.HGetAll(key)
		assert.Equal(t, err, nil)
		for key, v := range maps {
			if key == "ff1" {
				assert.Equal(t, v, "ff1")
			}
			if key == "ff2" {
				assert.Equal(t, v, "20000")
			}
			if key == "ff3" {
				assert.Equal(t, v, "1")
			}
		}

		err = Conn.Del(key)
		assert.Equal(t, err, nil)
	})
}

func TestScan(t *testing.T) {
	setRedisTest()
	t.Run("case 1", func(t *testing.T) {
		fields := []string{
			"Test:Scan:GROUP:1",
			"Test:Scan:GROUP:2",
			"Test:Scan:GROUP:3",
			"Test:Scan:GROUP:4",
		}
		for _, key := range fields {
			err := Conn.Set(key, 1, 0)
			assert.Equal(t, err, nil)
		}

		fields2, err := Conn.Scan("Test:Scan:GROUP:*")
		assert.Equal(t, err, nil)
		assert.Equal(t, len(fields2), 4)
		assert.Equal(t, fields, fields2)

		err = Conn.DelKeys(fields)
		assert.Equal(t, err, nil)
	})

	t.Run("case 2", func(t *testing.T) {
		fields := map[string]interface{}{
			"job_id":           "100001",
			"group_name":       "mytest",
			"name":             "AA01",
			"request_url":      "http://aa.test.net",
			"interval_pattern": "* * * 1 * 30",
			"type":             "http",
		}
		fields2 := map[string]interface{}{
			"job_id":           "100002",
			"group_name":       "myjob",
			"name":             "Job01",
			"request_url":      "http://my.test",
			"interval_pattern": "*/5 * * * * *",
			"type":             "http",
		}
		key1 := "Test:Scan:GROUP:Job_100001"
		key2 := "Test:Scan:GROUP:Job_100002"

		err := Conn.HSet(key1, fields, 0)
		assert.Equal(t, err, nil)
		err = Conn.HSet(key2, fields2, 0)
		assert.Equal(t, err, nil)

		job_id, err := Conn.HGet(key1, "job_id").Result()
		assert.Equal(t, job_id, fields["job_id"])
		assert.Equal(t, err, nil)

		maps, err := Conn.HGetAll(key1)
		for key, value := range maps {
			val, ok := fields[key]
			assert.Equal(t, ok, true)
			assert.Equal(t, val, value, key)
		}
		assert.Equal(t, err, nil)

		maps2, err := Conn.HGetAll(key2)
		for key, value := range maps2 {
			val, ok := fields2[key]
			assert.Equal(t, ok, true)
			assert.Equal(t, val, value, key)
		}
		assert.Equal(t, err, nil)

		err = Conn.Del(key1)
		assert.Equal(t, err, nil)

		err = Conn.Del(key2)
		assert.Equal(t, err, nil)

		job_id2, err := Conn.HGet(key2, "job_id").Result()
		assert.Equal(t, job_id2, "")
		assert.Equal(t, err.Error(), ErrNil.Error())
	})
}

func TestDelKeys(t *testing.T) {
	setRedisTest()

	// 在 Redis 中新增測試資料
	testKeys := []string{"key1", "key2", "key3"}
	for _, key := range testKeys {
		err := Conn.Set(key, "test", 5)
		if err != nil {
			t.Errorf("Failed to set test data in Redis: %v", err)
			return
		}
	}

	// 呼叫 DelKeys 函數進行測試
	err := Conn.DelKeys(testKeys)
	if err != nil {
		t.Errorf("Failed to delete keys: %v", err)
		return
	}

	// 確認 Redis 中的資料已被刪除
	for _, key := range testKeys {
		value, err := Conn.Get(key).Result()
		if err.Error() != ErrNil.Error() {
			t.Errorf("Key %s still exists in Redis after deletion: %v", key, value)
		}
	}
}

func TestExpire(t *testing.T) {
	setRedisTest()

	// 在 Redis 中新增測試資料
	testKey := "key1"
	err := Conn.Set(testKey, "test", 0)
	if err != nil {
		t.Errorf("Failed to set test data in Redis: %v", err)
		return
	}

	// 設定過期時間
	expire := int64(60)
	err = Conn.Expire(testKey, expire)
	if err != nil {
		t.Errorf("Failed to set expiration for key: %v", err)
		return
	}

	// 確認過期時間是否正確
	ttl, err := Conn.TTL(testKey)
	if err != nil {
		t.Errorf("Failed to get TTL for key: %v", err)
		return
	}
	if ttl != int(expire) {
		t.Errorf("Incorrect TTL for key. Expected: %d, Actual: %d", expire, ttl)
	}
}

func TestTTL(t *testing.T) {
	setRedisTest()

	key := "GROUP:Job_100001"
	expectedValue := "test_value"

	err := Conn.Set(key, expectedValue, 0)
	assert.Equal(t, err, nil)

	value, err := Conn.Get(key).Result()
	assert.Equal(t, err, nil)
	assert.Equal(t, value, expectedValue)

	tval, err := Conn.TTL(key)
	assert.Nil(t, err)
	assert.Equal(t, tval, 0)
}

func TestFlushall(t *testing.T) {
	setRedisTest()

	key := "GROUP:Job_100001"
	expectedValue := "test_value"

	err := Conn.Set(key, expectedValue, 0)
	assert.Equal(t, err, nil)

	err = Conn.Flushall()
	assert.Nil(t, err)

	value, err := Conn.Get(key).Result()
	assert.Equal(t, err.Error(), ErrNil.Error())
	assert.Equal(t, value, "")
}
