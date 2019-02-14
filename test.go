package main

import (
	"crypto/rand"
	"fmt"
	"strings"
)

const (
	ASCII_LOWERCASE = "abcdefghijklmnopqrstuvwxyz"
	ASCII_UPPERCASE = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	DIGITS          = "0123456789"
	PUNCTUATIONS    = "!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~"
	ALL_CHARS       = ASCII_LOWERCASE + ASCII_UPPERCASE + DIGITS + PUNCTUATIONS
)

func RandomString(n int, allowedChars string) string {
	if n < 1 {
		return ""
	}
	if allowedChars == "" {
		allowedChars = ALL_CHARS
	}
	var (
		charLen   = len(allowedChars)
		mask      = getMinimalBitMask(charLen - 1)
		buf       strings.Builder
		iterLimit = n * 64
		i         = 0
		randIdx   = 0
		random    = RandomInts(2 * n)
	)
	for i < n {
		if randIdx >= len(random) {
			random = RandomInts(2 * (n - i))
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
			panic(fmt.Errorf("Hit iteration limit when generating random %d string", n))
		}
	}
	return buf.String()
}

func RandomInts(n int) []int {
	var size = n * 4
	var randBytes = make([]byte, size)
	if _, err := rand.Read(randBytes[:]); err != nil {
		panic(fmt.Sprintf("could not generate random bytes for HTTP secret: %v", err))
	}
	var (
		xs = make([]int, n)
		x  int
	)
	for i := 0; i < n; i++ {
		x = 0
		for j := 0; j < 4; j++ {
			x = (x << 8) | (int(randBytes[i*4+j]) & 0xFF)
		}
		x = x & 2147483647
		xs = append(xs, x)
	}
	return xs
}

func getMinimalBitMask(to int) int {
	if to < 1 {
		panic(fmt.Errorf("the argument passed to getMinimalBitMask must be a positive integer, you passed %d", to))
	}
	mask := 1
	for mask < to {
		mask = (mask << 1) | 1
	}
	return mask
}

func main() {
	for i := 0; i < 32; i++ {
		fmt.Printf("random string 32: %s", RandomString(32, ""))
	}
}
