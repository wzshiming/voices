package voices

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	cacheDir = ".voices"
	space    = regexp.MustCompile(`\s+`)
)

func init() {
	home, _ := os.UserHomeDir()
	if home != "" {
		cacheDir = filepath.Join(home, cacheDir)
	}
}

func clean(text string) string {
	text = strings.TrimSpace(text)
	text = space.ReplaceAllString(text, " ")
	return text
}

func hashName(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
