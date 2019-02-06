package handlers

import (
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type VisitorKeyFunc func(*http.Request) string

type visitor struct {
	limiter *rate.Limiter
	lastSeen time.Time
}

type RateLimiter struct {
	limit rate.Limit
	burst int
	visitors map[string]*visitor
	lock sync.Mutex
	keyFunc VisitorKeyFunc
	done chan bool
}

func ipVisitorKeyFunc(r *http.Request) string {
	return r.RemoteAddr
}

func NewIpVisitor(n, b int) *RateLimiter {
	rl := &RateLimiter{
		limit: rate.Every(time.Minute * time.Duration(n)),
		burst: b,
		keyFunc: ipVisitorKeyFunc,
		done: make(chan bool, 1),
	}

	go rl.cleanup()

	return rl
}

func (rl *RateLimiter) Close() {
	rl.done <- true
}

func (rl *RateLimiter) Get(r *http.Request) *rate.Limiter {
	rl.lock.Lock()
	defer rl.lock.Unlock()

	if rl.visitors == nil {
		rl.visitors = make(map[string]*visitor)
	}
	key := rl.keyFunc(r)
	v, ok := rl.visitors[key]
	if !ok {
		return rl.add(key)
	}

	v.lastSeen = time.Now()
	return v.limiter
}

func (rl *RateLimiter) add(key string) *rate.Limiter {
	limiter := rate.NewLimiter(rl.limit, rl.burst)
	rl.lock.Lock()
	defer rl.lock.Unlock()

	rl.visitors[key] = &visitor{limiter, time.Now()}

	return limiter
}

func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1*time.Minute)
	defer func() {
		ticker.Stop()
	}()
	for {
		select {
		case <- rl.done:
			rl.lock.Lock()
			if rl.visitors == nil {
				rl.visitors = make(map[string]*visitor)
			}
			for key, _ := range rl.visitors {
				delete(rl.visitors, key)
			}
			rl.lock.Unlock()
			return
		case <- ticker.C:
			rl.lock.Lock()
			if rl.visitors == nil {
				rl.visitors = make(map[string]*visitor)
			}
			for key, v := range rl.visitors {
				if time.Now().Sub(v.lastSeen) > 3*time.Minute {
					delete(rl.visitors, key)
				}
			}
			rl.lock.Unlock()
			return
		}
	}
}
