package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

type untrackedRepo struct {
	Path string
	Rid  string
}

// local mirrors whose repo id is gone from GitHub (deleted or transferred)
func untrackedRepos(reposPath string, known map[int]bool) ([]untrackedRepo, error) {
	owners, err := os.ReadDir(reposPath)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var res []untrackedRepo
	for _, owner := range owners {
		if !owner.IsDir() {
			continue
		}
		repos, err := os.ReadDir(filepath.Join(reposPath, owner.Name()))
		if err != nil {
			return nil, err
		}
		for _, repo := range repos {
			if !repo.IsDir() {
				continue
			}
			id, err := strconv.Atoi(repo.Name())
			if err != nil || known[id] {
				continue
			}
			path := filepath.Join(reposPath, owner.Name(), repo.Name())
			res = append(res, untrackedRepo{Path: path, Rid: readRid(path + ".rid")})
		}
	}
	return res, nil
}
