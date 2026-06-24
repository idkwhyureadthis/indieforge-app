package idgen

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

var enc = base32.NewEncoding("abcdefghijklmnopqrstuvwxyz234567").WithPadding(base32.NoPadding)

// New returns a random identifier with the given prefix, e.g. New("usr") -> "usr_4x7k2p9q".
func New(prefix string) string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return prefix + "_" + strings.ToLower(enc.EncodeToString(b))
}
