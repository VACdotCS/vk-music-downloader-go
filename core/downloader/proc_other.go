//go:build !windows

package downloader

import "os/exec"

func setHideWindow(cmd *exec.Cmd) {
	// No-op on non-Windows platforms
}
