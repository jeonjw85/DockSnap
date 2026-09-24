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
	info, err := os.Lstat(root)
	if err != nil {
		return "", 0, err
	}
	var entries []string
	if info.IsDir() {
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if path == root {
				return nil
			}
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			entries = append(entries, rel)
			return nil
		})
	} else {
		entries = []string{"."}
	}
	if err != nil {
		return "", 0, err
	}
	sort.Strings(entries)
	h := sha256.New()
	var total uint64
	for _, rel := range entries {
		path := root
		if rel != "." {
			path = filepath.Join(root, rel)
		}
		info, err := os.Lstat(path)
		if err != nil {
			return "", 0, err
		}
		fmt.Fprintf(h, "%s\x00%d\x00", filepath.ToSlash(rel), info.Mode())
		switch {
		case info.Mode()&os.ModeSymlink != 0:
			target, err := os.Readlink(path)
			if err != nil {
				return "", 0, err
			}
			fmt.Fprintf(h, "link\x00%d\x00%s", len(target), target)
			total += uint64(len(target))
		case info.IsDir():
			fmt.Fprint(h, "dir\x00")
		case info.Mode().IsRegular():
			fmt.Fprintf(h, "file\x00%d\x00", info.Size())
			f, err := os.Open(path)
			if err != nil {
				return "", 0, err
			}
			n, copyErr := io.Copy(h, f)
			closeErr := f.Close()
			if copyErr != nil {
				return "", 0, copyErr
			}
			if closeErr != nil {
				return "", 0, closeErr
			}
			total += uint64(n)
		default:
			return "", 0, fmt.Errorf("unsupported file type %s", path)
		}
	}
	return fmt.Sprintf("sha256-tree-v2:%x", h.Sum(nil)), total, nil
}

func checksumTreeLegacy(root string) (string, uint64, error) {
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
		fmt.Fprintf(h, "%s\x00%d\x00%d\x00", filepath.ToSlash(rel), info.Mode(), info.Size())
		f, err := os.Open(p)
		if err != nil {
			return "", 0, err
		}
		n, copyErr := io.Copy(h, f)
		closeErr := f.Close()
		if copyErr != nil {
			return "", 0, copyErr
		}
		if closeErr != nil {
			return "", 0, closeErr
		}
		total += uint64(n)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), total, nil
}
