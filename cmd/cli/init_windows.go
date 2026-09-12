//go:build windows
// +build windows

package main

import (
	"os"
	"golang.org/x/sys/windows"
)

func init() {
	// Принудительно включаем кодировку UTF-8 (чтобы эмодзи не превращались в кракозябры/вопросики)
	windows.SetConsoleOutputCP(windows.CP_UTF8)
	windows.SetConsoleCP(windows.CP_UTF8)

	// Включаем поддержку ANSI (виртуальный терминал) для корректного отображения цветов и прогресс-баров
	stdout := windows.Handle(os.Stdout.Fd())
	var originalMode uint32
	if err := windows.GetConsoleMode(stdout, &originalMode); err == nil {
		windows.SetConsoleMode(stdout, originalMode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
	}
}
