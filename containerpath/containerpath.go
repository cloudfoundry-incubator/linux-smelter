// +build !windows2012R2

package containerpath

import (
	"path/filepath"
)

func New(_ string) *cpath {
	return &cpath{
		root: "/",
	}
}

func (c *cpath) For(path ...string) string {
	path = append([]string{c.root}, path...)
	return filepath.Clean(filepath.Join(path...))
}
