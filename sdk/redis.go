package wechat

import (
	"github.com/go-redis/redis"
	"time"
)

type ClientType struct {
	RedisCon *redis.Client
}

var RedisClient *ClientType

func InitRedis(ip string, port string, password string, db int) {
	RedisClient = &ClientType{
		RedisCon: redis.NewClient(&redis.Options{
			Addr:     ip + ":" + port,
			Password: password, // no password set
			DB:       db,          // use default DB
		}),
	}
}

func (Client *ClientType) Set(key string, value interface{}, expiration time.Duration) *redis.Client {
	err := (*Client).RedisCon.Set(key, value, expiration).Err()
	if err != nil {
		panic(err)
	}
	return (*Client).RedisCon
}

func (Client *ClientType) Get(key string) (string, *redis.Client) {
	val, err := (*Client).RedisCon.Get(key).Result()

	if err == redis.Nil {
		return "", (*Client).RedisCon
	}

	if err != nil {
		panic(err)
	}

	return val, (*Client).RedisCon
}

func (Client *ClientType) Del(key string) *redis.Client {
	_, err := (*Client).RedisCon.Del(key).Result()
	if err != nil {
		panic(err)
	}
	return (*Client).RedisCon
}
