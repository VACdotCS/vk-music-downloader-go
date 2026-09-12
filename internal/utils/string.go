package utils

import (
	"regexp"
	"strings"
)

var fileNameRegExp = regexp.MustCompile(`[\/|.?<>":,«»æ*\\;]+`)
var invalidCharsRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]+`)

func GetNormalFileName(artist, title string) string {
	s := artist + " - " + title
	return fileNameRegExp.ReplaceAllString(s, "") + ".mp3"
}

func SanitizeFolderName(name string) string {
	s := invalidCharsRegex.ReplaceAllString(name, "")
	s = strings.TrimLeft(s, ".")
	s = strings.TrimRight(s, ".")
	s = strings.TrimSpace(s)
	return s
}
