// MIT License Copyright (C) 2022 Hiroshi Shimamoto
package argit

import (
	"io/fs"
)

type FileInfo struct {
	fs.FileInfo
	dir string
}

func (f FileInfo) Dir() string {
	return f.dir
}
