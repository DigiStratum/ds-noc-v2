// Thin wrapper for ds-scaffold add-feature
package main

import (
	"os"
	"os/exec"
	"syscall"
)

func main() {
	args := append([]string{"add-feature"}, os.Args[1:]...)
	
	binary, err := exec.LookPath("ds-scaffold")
	if err != nil {
		cmd := exec.Command("go", append([]string{"run", "github.com/DigiStratum/GoTools/cmd/ds-scaffold@latest", "add-feature"}, os.Args[1:]...)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		return
	}
	
	if err := syscall.Exec(binary, append([]string{"ds-scaffold"}, args...), os.Environ()); err != nil {
		os.Exit(1)
	}
}
