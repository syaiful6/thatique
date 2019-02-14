package text

import (
	"crypto/rand"
	"errors"
	"strings"
)

const (
	ASCII_LOWERCASE = "abcdefghijklmnopqrstuvwxyz"
	ASCII_UPPERCASE = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	DIGITS          = "0123456789"
	PUNCTUATIONS    = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	ALL_CHARS       = ASCII_LOWERCASE + ASCII_UPPERCASE + DIGITS + PUNCTUATIONS
)

var (
	ErrMaxIteration = errors.New("Maximum iteration exceeded during generating string")
	ErrInvalidArgs  = errors.New("Must be non negative integer")
)

// generate unbiased random string.
func RandomString(n int, allowedChars string) (string, error) {
	if n < 1 {
		return "", ErrInvalidArgs
	}
	if allowedChars == "" {
		allowedChars = ALL_CHARS
	}

	var (
		charLen   = len(allowedChars)
		buf       strings.Builder
		iterLimit = n * 64
		i         = 0
		randIdx   = 0
	)

	mask, err := getMinimalBitMask(charLen - 1)
	if err != nil {
		return "", err
	}

	random, err := randomInts(2 * n)
	if err != nil {
		return "", err
	}

	for i < n {
		if randIdx >= len(random) {
			random, err = randomInts(2 * (n - i))
			if err != nil {
				return "", err
			}
			randIdx = 0
		}
		c := random[randIdx] & mask
		randIdx += 1
		if c < charLen {
			buf.WriteByte(allowedChars[c])
			i += 1
		}
		iterLimit -= 1
		if iterLimit <= 0 {
			return "", ErrMaxIteration
		}
	}
	return buf.String(), nil
}

func randomInts(n int) (xs []int, err error) {
	var size = n * 4
	var randBytes = make([]byte, size)
	if _, err = rand.Read(randBytes[:]); err != nil {
		return
	}
	var x int
	for i := 0; i < n; i++ {
		x = 0
		for j := 0; j < 4; j++ {
			x = (x << 8) | (int(randBytes[i*4+j]) & 0xFF)
		}
		x = x & 2147483647
		xs = append(xs, x)
	}
	return
}

func getMinimalBitMask(to int) (mask int, err error) {
	if to < 1 {
		err = ErrInvalidArgs
		return
	}
	mask = 1
	for mask < to {
		mask = (mask << 1) | 1
	}
	return
}
