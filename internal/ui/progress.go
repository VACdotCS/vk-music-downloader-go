package ui

import (
	"fmt"
	"math"
	"strings"
)

// RenderDownloaderProgress полностью повторяет логику из Node.js версии.
// Выравнивает прогресс-бар по длине самого большого названия трека в батче.
func RenderDownloaderProgress(percentage float64, currentTitleLength, maxTitleLength, marginCorrection int) string {
	copy := int(percentage * 100)
	if copy < 0 {
		copy = 0
	}
	if copy > 100 {
		copy = 100
	}

	dots := strings.Repeat("#", copy)
	spaces := strings.Repeat(" ", 100-copy)

	diff := float64(maxTitleLength - currentTitleLength - marginCorrection)
	paddingSpaces := int(math.Abs(diff))
	padding := strings.Repeat(" ", paddingSpaces)

	return fmt.Sprintf("%s[%s%s] %d/100%%", padding, dots, spaces, copy)
}
