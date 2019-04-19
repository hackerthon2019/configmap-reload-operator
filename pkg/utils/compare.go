package utils

import (
	"crypto/md5"
	"fmt"
	"io"
)

// IsSameMD5 comparing MD5 string if same
func IsSameMD5(a string, b string) bool {
	h := md5.New()
	o := md5.New()
	io.WriteString(h, a)
	io.WriteString(o, b)
	return fmt.Sprintf("%x", h.Sum(nil)) == fmt.Sprintf("%x", o.Sum(nil))
}
