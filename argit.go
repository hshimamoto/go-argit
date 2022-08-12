// MIT License Copyright (C) 2022 Hiroshi Shimamoto
package argit

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/filesystem"
)

// savetarball create tar archive
func savetarball(tf *TARFile, fs billy.Filesystem) error {
	return Billywalk(fs, "/", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		// remove "/"
		path = path[1:]
		f, err := fs.Open(path)
		if err != nil {
			// fatal
			return err
		}
		tf.WriteRegFile(path, info, f)
		f.Close()
		return nil
	})
}

func archive(tf *TARFile, gitdir string) error {
	writer := func(name string) error {
		info, err := os.Stat(name)
		if err != nil {
			return err
		}
		f, err := os.Open(name)
		if err != nil {
			return err
		}
		defer f.Close()
		return tf.WriteRegFile(name, info, f)
	}

	curr, _ := os.Getwd()
	err := os.Chdir(gitdir)
	if err != nil {
		return err
	}
	defer os.Chdir(curr)

	// use only [HEAD config index objects/ refs/]
	if _, err := os.Stat("packed-refs"); err == nil {
		err = writer("packed-refs")
		if err != nil {
			return err
		}
	}
	err = writer("HEAD")
	if err != nil {
		return err
	}
	err = writer("config")
	if err != nil {
		return err
	}
	err = writer("index")
	if err != nil {
		return err
	}
	err = filepath.Walk("objects/", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		return writer(path)
	})
	if err != nil {
		return err
	}
	err = filepath.Walk("refs/", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		return writer(path)
	})
	if err != nil {
		return err
	}
	return nil
}

func checkout(r *git.Repository, wt *git.Worktree) error {
	bi, err := r.Branches()
	if err != nil {
		return nil
	}
	defer bi.Close()
	first, err := bi.Next()
	if err != nil {
		return nil
	}
	return wt.Checkout(&git.CheckoutOptions{
		Force:  true,
		Branch: first.Name(),
	})
}

// Repository contains go-git Repository and some shortcut params
type Repository struct {
	*git.Repository
	fs       billy.Filesystem
	worktree *git.Worktree
}

// InitTARFile initializes tarball with the new git repository
func InitTARFile(tf *TARFile, files billy.Filesystem) error {
	fs := memfs.New()
	s := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	r, err := git.Init(s, files)
	if err != nil {
		return err
	}
	// add files and commit
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	err = Billywalk(files, "/", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		name := filepath.Base(path)
		if name == ".git" {
			return nil
		}
		wt.Add(path[1:])
		return nil
	})
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	author := &object.Signature{
		Name:  cfg.Name,
		Email: cfg.Email,
		When:  time.Now(),
	}
	commit, err := wt.Commit("first commit", &git.CommitOptions{
		Author: author,
	})
	if err != nil {
		return err
	}
	_, err = r.CommitObject(commit)
	if err != nil {
		return err
	}

	// write to TARFile
	return savetarball(tf, fs)
}

// Init creates new git repository tarball
func Init(path string, files billy.Filesystem) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = InitTARFile(NewTARFile(f), files)
	if err != nil {
		os.Remove(path)
		return err
	}

	return nil
}

// OpenRepositoryFromTARFile setups git repository
func OpenRepositoryFromTARFile(tf *TARFile) (*Repository, error) {
	// setup repository files in memfs from tarball
	fs := memfs.New()
	for {
		hdr, rd, err := tf.ReadRegFile()
		if err != nil {
			break
		}
		f, err := fs.Create(hdr.Name)
		if err != nil {
			// fatal
			return nil, err
		}
		io.Copy(f, rd)
		f.Close()
	}
	s := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	r, err := git.Open(s, memfs.New())
	if err != nil {
		return nil, err
	}
	wt, err := r.Worktree()
	if err != nil {
		return nil, err
	}
	err = checkout(r, wt)
	if err != nil {
		return nil, err
	}
	return &Repository{
		Repository: r,
		fs:         fs,
		worktree:   wt,
	}, nil
}

// OpenRepository opens tarball as git repository
func OpenRepository(path string) (*Repository, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return OpenRepositoryFromTARFile(NewTARFile(f))
}

// CloneTARFile clones url repo to TARFile
func CloneTARFile(tf *TARFile, url string) error {
	remote := strings.HasPrefix(url, "git:") || strings.HasPrefix(url, "https:")
	if !remote {
		// from local filesystem
		err := archive(tf, url)
		if err != nil {
			return err
		}
		return nil
	}
	fs := memfs.New()
	s := filesystem.NewStorage(fs, cache.NewObjectLRUDefault())
	r, err := git.Clone(s, memfs.New(), &git.CloneOptions{
		URL: url,
	})
	if err != nil {
		return err
	}
	wt, err := r.Worktree()
	if err != nil {
		return err
	}
	err = checkout(r, wt)
	if err != nil {
		return err
	}
	repo := &Repository{
		Repository: r,
		fs:         fs,
		worktree:   wt,
	}
	return repo.SaveTARFile(tf)
}

// Clone clones url repo
func Clone(path, url string) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	err = CloneTARFile(NewTARFile(f), url)
	if err != nil {
		os.Remove(path)
		return err
	}

	return nil
}

// SaveTARFile writes git repository to TARFile
func (r *Repository) SaveTARFile(tf *TARFile) error {
	return savetarball(tf, r.fs)
}

// Save creats tarball which contains git repository
func (r *Repository) Save(path string) error {
	f, err := os.OpenFile(path, os.O_TRUNC|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	return savetarball(NewTARFile(f), r.fs)
}

// Logs returns CommitIter of HEAD
func (r *Repository) Logs() (object.CommitIter, error) {
	head, err := r.Head()
	if err != nil {
		return nil, err
	}
	return r.Log(&git.LogOptions{From: head.Hash()})
}

// Files returns array of FileInfo in worktree
func (r *Repository) Files() ([]FileInfo, error) {
	files := []FileInfo{}
	Billywalk(r.worktree.Filesystem, "/", func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		files = append(files, FileInfo{info, filepath.Dir(path)})
		return nil
	})
	return files, nil
}

// Get extracts a file from worktree
func (r *Repository) Get(localfs billy.Filesystem, path string) error {
	fs := r.worktree.Filesystem
	reader, err := fs.Open(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	localfs.MkdirAll(filepath.Dir(path), 0755)
	writer, err := localfs.Create(path)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.Copy(writer, reader)
	return err
}

// Put stores a file into worktree
func (r *Repository) Put(localfs billy.Filesystem, path string) error {
	fs := r.worktree.Filesystem
	reader, err := localfs.Open(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	fs.MkdirAll(filepath.Dir(path), 0755)
	writer, err := fs.Create(path)
	if err != nil {
		return err
	}
	defer writer.Close()
	_, err = io.Copy(writer, reader)
	return err
}

// Add makes a file to stage
func (r *Repository) Add(path string) error {
	_, err := r.worktree.Add(path)
	return err
}

// Commit to git repository
func (r *Repository) Commit(msg string) error {
	status, err := r.worktree.Status()
	if err != nil {
		return err
	}
	if status.IsClean() {
		return fmt.Errorf("no midification")
	}
	cfg, err := LoadConfig()
	if err != nil {
		return err
	}
	author := &object.Signature{
		Name:  cfg.Name,
		Email: cfg.Email,
		When:  time.Now(),
	}
	commit, err := r.worktree.Commit(msg, &git.CommitOptions{
		Author: author,
	})
	if err != nil {
		return err
	}
	_, err = r.CommitObject(commit)
	return err
}
