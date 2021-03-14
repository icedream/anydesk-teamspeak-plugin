//+build windows

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func init() {
	programfilesx86 := os.Getenv("PROGRAMFILES(X86)")
	anydeskBinaryDirectoryPaths = append(anydeskBinaryDirectoryPaths, filepath.Join(programfilesx86, "AnyDesk"))
	anydeskBinaryName += ".exe"
}

func makeProcessInvisible(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
