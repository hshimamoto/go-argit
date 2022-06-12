// MIT License Copyright (C) 2022 Hiroshi Shimamoto
package argit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Gitconfig struct {
	Name  string
	Email string
}

var config *Gitconfig

func LoadConfig() (*Gitconfig, error) {
	if config != nil {
		return config, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(home, ".gitconfig"))
	if err != nil {
		return nil, err
	}
	// parse
	cfg := &Gitconfig{}
	user := false
	for _, line := range strings.Split(string(b), "\n") {
		// remove spaces
		l := strings.TrimSpace(line)
		if l == "" {
			continue
		}
		if l[0] == '[' {
			user = (l == "[user]")
			continue
		}
		if user {
			a := strings.SplitN(l, "=", 2)
			if len(a) != 2 {
				continue
			}
			k := strings.ToLower(strings.TrimSpace(a[0]))
			v := strings.TrimSpace(a[1])
			if k == "name" {
				cfg.Name = v
			}
			if k == "email" {
				cfg.Email = v
			}
		}
	}
	if cfg.Name == "" || cfg.Email == "" {
		return nil, fmt.Errorf("Name or Email is empty")
	}
	config = cfg
	return config, nil
}
