// Thin wrapper for ds-scaffold add-endpoint
package main

import (
	"os"
	"os/exec"
	"syscall"
)

func main() {
	args := append([]string{"add-endpoint"}, os.Args[1:]...)
	
	// Find ds-scaffold binary
	binary, err := exec.LookPath("ds-scaffold")
	if err != nil {
		// Try to run via go run if not installed
		cmd := exec.Command("go", append([]string{"run", "github.com/DigiStratum/GoTools/cmd/ds-scaffold@latest", "add-endpoint"}, os.Args[1:]...)...)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			os.Exit(1)
		}
		return
	}
	
	// Exec ds-scaffold directly
	if err := syscall.Exec(binary, append([]string{"ds-scaffold"}, args...), os.Environ()); err != nil {
		os.Exit(1)
	}
}
