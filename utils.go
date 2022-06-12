// MIT License Copyright (C) 2022 Hiroshi Shimamoto
package argit

import (
	"os"
	"path/filepath"

	"github.com/go-git/go-billy/v5"
)

// walk billy 5.3.1 doesn't have walk
func walk(fs billy.Filesystem, path string, info os.FileInfo, fn filepath.WalkFunc) error {
	if !info.IsDir() {
		return fn(path, info, nil)
	}
	dirnames := func(dirname string) ([]string, error) {
		entries, err := fs.ReadDir(dirname)
		if err != nil {
			return nil, err
		}
		names := []string{}
		for _, e := range entries {
			names = append(names, e.Name())
		}
		return names, nil
	}
	names, err := dirnames(path)
	if err != nil {
		return err
	}
	for _, name := range names {
		filename := filepath.Join(path, name)
		fileinfo, err := fs.Lstat(filename)
		if err != nil {
			return err
		}
		err = walk(fs, filename, fileinfo, fn)
		if err != nil {
			return err
		}
	}
	return nil
}

func Billywalk(fs billy.Filesystem, root string, fn filepath.WalkFunc) error {
	info, err := fs.Lstat(root)
	if err != nil {
		return err
	}
	return walk(fs, root, info, fn)
}
