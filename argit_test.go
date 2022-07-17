// MIT License Copyright (C) 2022 Hiroshi Shimamoto
package argit

import "testing"

import (
	"bytes"
	"os"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
)

const path = ".test.tar"

func initTestRepo(fs billy.Filesystem) error {
	os.Remove(path)
	config = &Gitconfig{
		Name:  "go test",
		Email: "go test",
	}
	return Init(path, fs)
}

func TestInitAndOpen(t *testing.T) {
	fs := memfs.New()
	fs.Create("README.md")
	err := initTestRepo(fs)
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
	// remove local newfile
	localfs.Remove("newfile")
	// get newfile
	err = r.Get(localfs, "newfile")
	if err != nil {
		t.Errorf("Get: %v", err)
		return
	}
	// check
	f, err = localfs.Open("newfile")
	if err != nil {
		t.Errorf("Open: %v", err)
		return
	}
	buf := new(bytes.Buffer)
	buf.ReadFrom(f)
	if buf.String() != "dummy!" {
		t.Errorf("content mismatch")
		return
	}
	f.Close()
}

func TestInitTree(t *testing.T) {
	fs := memfs.New()
	fs.MkdirAll("/dir/dir/dir", 0755)
	fs.Create("/dir/dir/dir/file")
	err := initTestRepo(fs)
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
	if files[0].Dir() != "/dir/dir/dir" {
		t.Errorf("Files: dir=%s", files[0].Dir())
		return
	}
	if files[0].Name() != "file" {
		t.Errorf("Files: name=%s", files[0].Name())
		return
	}
	// Get test
	localfs := memfs.New()
	err = r.Get(localfs, "dir/dir/dir/file")
	if err != nil {
		t.Errorf("Get: %v", err)
		return
	}
	f, err := localfs.Open("dir/dir/dir/file")
	if err != nil {
		t.Errorf("Open: %v", err)
		return
	}
	f.Close()
	// Put test
	localfs.MkdirAll("dir1/dir2/dir3/dir4", 0755)
	f, err = localfs.Create("dir1/dir2/dir3/dir4/file5")
	if err != nil {
		t.Errorf("Create: %v", err)
		return
	}
	f.Close()
	err = r.Put(localfs, "dir1/dir2/dir3/dir4/file5")
	if err != nil {
		t.Errorf("Put: %v", err)
		return
	}
	err = r.Add("dir1/dir2/dir3/dir4/file5")
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
	files, err = r2.Files()
	if err != nil {
		t.Errorf("Files: %v", err)
		return
	}
	if len(files) != 2 {
		t.Errorf("Files: len(files)=%d", len(files))
		return
	}
	f1 := false
	f2 := false
	for _, file := range files {
		switch file.Name() {
		case "file":
			f1 = file.Dir() == "/dir/dir/dir"
		case "file5":
			f2 = file.Dir() == "/dir1/dir2/dir3/dir4"
		}
	}
	if (!f1) || (!f2) {
		t.Errorf("missing file")
		return
	}
	// Get test
	localfs2 := memfs.New()
	err = r2.Get(localfs2, "/dir1/dir2/dir3/dir4/file5")
	if err != nil {
		t.Errorf("Get: %v", err)
		return
	}
	f, err = localfs2.Open("dir1/dir2/dir3/dir4/file5")
	if err != nil {
		t.Errorf("Open: %v", err)
		return
	}
	f.Close()
}
