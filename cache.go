package voices

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
)

var cacheDir = ".voices"

func init() {
	home, _ := os.UserHomeDir()
	if home != "" {
		cacheDir = filepath.Join(home, cacheDir)
	}
}

func clean(text string) string {
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "(", " ")
	text = strings.ReplaceAll(text, ")", " ")
	return text
}

func hashName(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}
