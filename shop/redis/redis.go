package redis

import (
	"time"

	"github.com/garyburd/redigo/redis"

	"github.com/syaiful6/thatique/configuration"
)

func DialWithConf(conf configuration.Redis) (redis.Conn, error) {
	conn, err := redis.DialTimeout("tcp",
		conf.Addr,
		conf.DialTimeout,
		conf.ReadTimeout,
		conf.WriteTimeout)

	if err != nil {
		return nil, err
	}

	if conf.Password != "" {
		// do auth
		if _, err := conn.Do("AUTH", conf.Password); err != nil {
			conn.Close()
			return nil, err
		}
	}

	// select DB if asked
	if conf.DB != 0 {
		if _, err = conn.Do("SELECT", conf.DB); err != nil {
			conn.Close()
			return nil, err
		}
	}

	return conn, nil
}

// Create redis Pool
func NewRedisPool(conf configuration.Redis) (*redis.Pool, error) {
	pool := &redis.Pool{
		MaxIdle:     conf.MaxIdle,
		MaxActive:   conf.MaxActive,
		IdleTimeout: conf.IdleTimeout,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		Dial: func() (redis.Conn, error) {
			return DialWithConf(conf)
		},
	}

	// test the connection
	_, err := Ping(pool)
	return pool, err
}

// Ping against a server to check if it is alive.
func Ping(pool *redis.Pool) (bool, error) {
	conn := pool.Get()
	defer conn.Close()
	data, err := conn.Do("PING")
	return (data == "PONG"), err
}
