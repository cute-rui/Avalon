package utils

import (
	"math/rand"
	"reflect"
	"strings"
	"time"
	"unsafe"
)

func StringBuilder(p ...string) string {
	var (
		b strings.Builder
		c int
	)
	l := len(p)
	for i := 0; i < l; i++ {
		c += len(p[i])
	}
	b.Grow(c)
	for i := 0; i < l; i++ {
		b.WriteString(p[i])
	}
	return b.String()
}

var src = rand.NewSource(time.Now().UnixNano())

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandString(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func ByteToString(p []byte) string {
	var b strings.Builder
	l := len(p)
	b.Grow(l)
	for i := 0; i < l; i++ {
		b.WriteByte(p[i])
	}
	return b.String()
}

func StringToByte(s string) []byte {
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	bh := reflect.SliceHeader{
		Data: sh.Data,
		Len:  sh.Len,
		Cap:  sh.Len,
	}
	return *(*[]byte)(unsafe.Pointer(&bh))
}
