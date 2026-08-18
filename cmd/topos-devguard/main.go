// Command topos-devguard is the dev-run isolation gate (ISOL-01/ISOL-02).
package main

import (
	"fmt"
	"io"
	"os"
)

func run(args []string, stdout, stderr io.Writer) int {
	fmt.Fprintln(stderr, "devguard: not implemented")
	return 1
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}
