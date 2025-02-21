package model

import (
	"errors"

	"github.com/go-redis/redis"
)

var (
	Gclient0       *redis.Client //
	Gclient1       *redis.Client //
	GRedisIsactive bool
	rediserr       = errors.New("redis no run error")
)

type RedisParams struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func RedisInit(host string, port string, password string, db int) error {
	var (
		err error
	)
	GRedisIsactive = false

	Gclient0 = redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password, // no password set
		DB:       db,       // use default DB
	})
	_, err = Gclient0.Ping().Result()
	if err != nil {
		return err
	}

	Gclient1 = redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password, // no password set
		DB:       1,        // use default DB
	})
	_, err = Gclient1.Ping().Result()
	if err != nil {
		return err
	}

	GRedisIsactive = true
	//go Deal_MintCount()

	return nil
}

func (rp *RedisParams) Setdb() error {
	err := rediserr
	if GRedisIsactive {
		err = Gclient0.Set(rp.Key, rp.Value, 0).Err()
	}

	return err
}

func (rp *RedisParams) Getdb() error {
	err := rediserr
	if GRedisIsactive {
		val, err1 := Gclient0.Get(rp.Key).Result()
		rp.Value = val
		return err1
	}
	return err
}

func (rp *RedisParams) Getdbsize() (int64, error) {
	err := rediserr
	if GRedisIsactive {
		return Gclient0.DBSize().Result()
	}
	return 0, err
}

func (rp *RedisParams) Cleardb() error {
	err := rediserr
	var cursor uint64
	var n int
	var keys []string

	if GRedisIsactive {
				break
			}
		}

		for _, k := range keys {
			err = Gclient0.Del(k).Err()
			if err != nil {
				return err
			}
		}

	}
	return err
}

func (rp *RedisParams) Getdb1() error {
	err := rediserr
	if GRedisIsactive {
		val, err1 := Gclient1.Get(rp.Key).Result()
		rp.Value = val
		return err1
	}
	return err
}
