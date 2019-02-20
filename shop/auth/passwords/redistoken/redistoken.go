package redistoken

import (
	"github.com/gomodule/redigo/redis"
	"github.com/syaiful6/thatique/pkg/text"
	"github.com/syaiful6/thatique/shop/auth"
)

type RedisToken struct {
	Token     string
	Email     string
	Pass      []byte
	CreatedAt int64
}

type RedisTokenGenerator struct {
	Pool      *redis.Pool
	Expire    int
	keyPrefix string
}

const tokenAllowedChars = text.ASCII_LOWERCASE + text.ASCII_UPPERCASE + text.DIGITS + "-_~"

var insertScript = redis.NewScript(2, `
	local tokens = {}
	for i = 2, #ARGS, 1 do
		tokens[#tokens + 1] = ARGV[i]
	end

	redis.call('HMSET', KEYS[1], unpack(tokens))
	redis.call('SET', KEYS[2], KEYS[1])

	if(ARGV[1] ~= '') then
		redis.call('EXPIRE', KEYS[1], ARGV[1])
	end

	return true
`)

func NewRedisTokenGenerator(pool *redis.Pool) *RedisTokenGenerator {
	return &RedisTokenGenerator{
		Pool:      pool,
		Expire:    86400, // one day
		keyPrefix: "token:generator:",
	}
}

func (t *RedisTokenGenerator) SetKeyPrefix(prefix string) {
	t.keyPrefix = prefix
}

func (t *RedisTokenGenerator) Generate(user *auth.User) (token string, err error) {
	conn := t.Pool.Get()
	defer conn.Close()

	if err := conn.Err(); err != nil {
		return "", err
	}

	token, err = text.RandomString(32, tokenAllowedChars)
	if err != nil {
		return "", err
	}

	tok := &RedisToken{
		Token:     token,
		Email:     user.Email,
		Pass:      user.Password,
		CreatedAt: time.Now().UTC().Unix(),
	}

	args := redis.Args{}.Add(t.keyPrefix + user.Id.Hex()).Add(t.keyPrefix + token)
	args = args.Add(t.Expire).AddFlat(tok)
	_, err = insertScript.Do(conn, args...)
	if err != nil {
		return "", err
	}

	return token, nil
}

func (t *RedisTokenGenerator) Delete(token string) error {
	conn := t.Pool.Get()
	defer conn.Close()

	if err := conn.Err(); err != nil {
		return err
	}

	tok := t.keyPrefix + token
	key, err := redis.String(conn.Do("GET", tok))
	if err != nil {
		return err
	}
	conn.Send("MULTI")
	conn.Send("DEL", tok)
	conn.Send("DEL", key)
	_, err = conn.Do("EXEC")

	return err
}

func (t *RedisTokenGenerator) IsValid(user *auth.User, token string) bool {
	conn := t.Pool.Get()
	defer conn.Close()

	if err := conn.Err(); err != nil {
		return false
	}

	data, err := redis.Values(conn.Do("HGETALL", t.keyPrefix+user.Id.Hex()))
	if err != nil {
		return false
	}

	if len(data) == 0 {
		return false
	}

	var stored = new(RedisToken)
	if err = redis.ScanStruct(data, stored); err != nil {
		return false
	}

	if stored.Email != user.Email {
		return false
	}

	if stored.Token != token {
		return false
	}

	return subtle.ConstantTimeCompare(stored.Pass, user.Password) == 1
}
