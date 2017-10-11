// +build windows,windows2012R2

package containerpath

import (
	"fmt"
	"os"
	"path/filepath"
)

func For(path string) string {
	fmt.Fprintf(os.Stderr, "\nPath is %v", path)
	fmt.Fprintf(os.Stderr, "\nEnv is %v", os.Getenv("USERPROFILE"))
	fmt.Fprintf(os.Stderr, "\nCleaned, Env is %v", filepath.Clean(os.Getenv("USERPROFILE")))
	fmt.Fprintf(os.Stderr, "\nJoined Cleaned, Env is %v", filepath.Join(filepath.Clean(os.Getenv("USERPROFILE")), path))
	return filepath.Join(filepath.Clean(os.Getenv("USERPROFILE")), path)
}
