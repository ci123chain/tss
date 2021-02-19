package tgo

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

func UtilCryptoMD5Lower(str string) string {
	str = strings.ToLower(strings.TrimSpace(str))
	hash := md5.New()
	hash.Write([]byte(str))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func UtilCryptoSha1(s string) string {
	t := sha1.New()
	io.WriteString(t, s)
	return fmt.Sprintf("%x", t.Sum(nil))
}

func UtilCryptoMd5(s string) string {
	md5Ctx := md5.New()
	md5Ctx.Write([]byte(s))
	cipherStr := md5Ctx.Sum(nil)
	return hex.EncodeToString(cipherStr)
}

func UtilCryptoGenerateRandomToken16() (string, error) {
	return UtilCryptoGenerateRandomToken(16)
}

func UtilCryptoGenerateRandomToken32() (string, error) {
	return UtilCryptoGenerateRandomToken(32)
}

func UtilCryptoGenerateRandomToken(n int) (string, error) {
	token := make([]byte, n)
	_, err := rand.Read(token)
	return fmt.Sprintf("%x", token), err
}
