// MIT License Copyright (C) 2022 Hiroshi Shimamoto
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/hshimamoto/go-argit"

	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	"github.com/go-git/go-git/v5/plumbing/object"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("argit <tarball> <command> [args...]")
		return
	}
	// do init
	if os.Args[2] == "init" {
		fs := memfs.New()
		f, err := fs.Create("README.md")
		if err != nil {
			log.Printf("Create: %v", err)
			return
		}
		f.Write([]byte("README"))
		f.Close()
		err = argit.Init(os.Args[1], fs)
		if err != nil {
			log.Printf("%v", err)
		}
		return
	}
	// clone
	if os.Args[2] == "clone" {
		err := argit.Clone(os.Args[1], os.Args[3])
		if err != nil {
			log.Printf("CloneRepository: %v", err)
			return
		}
		return
	}
	r, err := argit.OpenRepository(os.Args[1])
	if err != nil {
		log.Printf("OpenRepository: %v", err)
		return
	}
	switch os.Args[2] {
	case "logs":
		fmt.Println("logs")
		iter, err := r.Logs()
		if err != nil {
			log.Printf("Commits: %v", err)
			return
		}
		iter.ForEach(func(c *object.Commit) error {
			fmt.Printf("%v\n", c)
			return nil
		})
	case "files", "ls":
		fmt.Println("files")
		files, err := r.Files()
		if err != nil {
			log.Printf("Files: %v", err)
			return
		}
		for _, file := range files {
			dir := file.Dir()
			if dir == "/" {
				dir = ""
			}
			fmt.Printf("%s/%s %d %s\n", dir, file.Name(), file.Size(), file.ModTime())
		}
	case "get":
		fmt.Println("get")
		requests := map[string]bool{}
		for _, req := range os.Args[3:] {
			requests[req] = false
		}
		files, err := r.Files()
		if err != nil {
			log.Printf("Files: %v", err)
			return
		}
		for _, file := range files {
			requests[file.Name()] = true
		}
		bad := false
		for k, v := range requests {
			if !v {
				log.Printf("file %s is not found\n", k)
				bad = true
			}
		}
		if bad {
			return
		}
		localfs := osfs.New(".")
		for _, req := range os.Args[3:] {
			err := r.Get(localfs, req)
			if err != nil {
				log.Printf("Get: %v", err)
				return
			}
		}
	case "put":
		fmt.Println("put")
		for _, req := range os.Args[3:] {
			stat, err := os.Stat(req)
			if err != nil {
				log.Printf("Stat: %v", err)
				return
			}
			if stat.IsDir() {
				log.Printf("%s is directory", req)
				return
			}
		}
		localfs := osfs.New(".")
		for _, req := range os.Args[3:] {
			err := r.Put(localfs, req)
			if err != nil {
				log.Printf("Put: %v", err)
				return
			}
			err = r.Add(req)
			if err != nil {
				log.Printf("Add: %v", err)
				return
			}
		}
		err = r.Commit("put files")
		if err != nil {
			log.Printf("Commit: %v", err)
			return
		}
		err = r.Save(os.Args[1])
		if err != nil {
			log.Printf("Save: %v", err)
			return
		}
	}
}
