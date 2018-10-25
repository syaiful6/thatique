package redis

import (
	"strconv"
	"time"

	"github.com/gomodule/redigo/redis"
)


// Dial connect to redis server, do AUTH if passed non empty password
func Dial(network, address, password string) (redis.Conn, error) {
	c, err := redis.Dial(network, address)
	if err != nil {
		return nil, err
	}

	if password != "" {
		if _, err := c.Do("AUTH", password); err != nil {
			c.Close()
			return nil, err
		}
	}

	return c, err
}

// DialWithDB connect to redis server, do AUTH if passed non empty password
// and then select the given database
func DialWithDB(network, address, password string, db int) (redis.Conn, error) {
	c, err := Dial(network, address, password)
	if err != nil {
		return nil, err
	}
	
	if _, err := c.Do("SELECT", strconv.Itoa(db)); err != nil {
		c.Close()
		return nil, err
	}

	return c, err
}

// Create redis Pool
func NewRedisPool(size int, network, address, password string, db int) (*redis.Pool, error) {
	pool := &redis.Pool{
		MaxIdle:     size,
		IdleTimeout: 240 * time.Second,
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
		},
		Dial: func() (redis.Conn, error) {
			return DialWithDB(network, address, password, db)
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