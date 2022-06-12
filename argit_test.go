// MIT License Copyright (C) 2022 Hiroshi Shimamoto
package argit

import "testing"

import (
	"os"

	"github.com/go-git/go-billy/v5/memfs"
)

const path = ".test.tar"

func TestInitAndOpen(t *testing.T) {
	os.Remove(path)
	config = &Gitconfig{
		Name: "go test",
		Email: "go test",
	}
	fs := memfs.New()
	fs.Create("README.md")
	err := Init(path, fs)
	if err != nil {
		t.Errorf("Init: %v", err)
		return
	}
	r, err := OpenRepository(path)
	if err != nil {
		t.Errorf("OpenRepository: %v", err)
		return
	}
	files, err := r.Files()
	if err != nil {
		t.Errorf("Files: %v", err)
		return
	}
	if len(files) != 1 {
		t.Errorf("Files: len(files)=%d", len(files))
		return
	}
	if files[0].Name() != "README.md" {
		t.Errorf("Files: name=%s", files[0].Name())
		return
	}
}

func TestAddCommit(t *testing.T) {
	TestInitAndOpen(t)
	// there should be .test.tar
	// open again
	r, err := OpenRepository(path)
	if err != nil {
		t.Errorf("OpenRepository: %v", err)
		return
	}
	localfs := memfs.New()
	// prepare
	f, err := localfs.Create("newfile")
	if err != nil {
		t.Errorf("Create: %v", err)
		return
	}
	f.Write([]byte("dummy!"))
	f.Close()
	// store newfile
	err = r.Put(localfs, "newfile")
	if err != nil {
		t.Errorf("Put: %v", err)
		return
	}
	err = r.Add("newfile")
	if err != nil {
		t.Errorf("Add: %v", err)
		return
	}
	err = r.Commit("go test")
	if err != nil {
		t.Errorf("Commit: %v", err)
		return
	}
	err = r.Save(path)
	if err != nil {
		t.Errorf("Save: %v", err)
		return
	}
	// reopen
	r2, err := OpenRepository(path)
	if err != nil {
		t.Errorf("OpenRepository: %v", err)
		return
	}
	files, err := r2.Files()
	if err != nil {
		t.Errorf("Files: %v", err)
		return
	}
	if len(files) != 2 {
		t.Errorf("Files: len(files)=%d", len(files))
		return
	}
	readme := false
	newfile := false
	for _, file := range files {
		switch file.Name() {
		case "README.md":
			readme = true
		case "newfile":
			newfile = true
		}
	}
	if (!readme) || (!newfile) {
		t.Errorf("missing file")
		return
	}
}
