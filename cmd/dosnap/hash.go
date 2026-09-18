package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

func checksumFile(path string) (string, uint64, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer f.Close()
	h := sha256.New()
	n, err := io.Copy(h, f)
	if err != nil {
		return "", 0, err
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), uint64(n), nil
}

func checksumTree(root string) (string, uint64, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return "", 0, err
	}
	sort.Strings(files)
	h := sha256.New()
	var total uint64
	for _, rel := range files {
		p := filepath.Join(root, rel)
		info, err := os.Stat(p)
		if err != nil {
			return "", 0, err
		}
		content, err := os.ReadFile(p)
		if err != nil {
			return "", 0, err
		}
		fmt.Fprintf(h, "%s\x00%d\x00%d\x00", filepath.ToSlash(rel), info.Mode(), len(content))
		h.Write(content)
		total += uint64(len(content))
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), total, nil
}
