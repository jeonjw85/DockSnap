package copy

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func SaveBind(ctx context.Context, src, dest string) error {
	if err := copyBind(ctx, src, dest); err != nil {
		return fmt.Errorf("save bind %s: %w", src, err)
	}
	return nil
}

func RestoreBind(ctx context.Context, src, dest string) error {
	if err := copyBind(ctx, src, dest); err != nil {
		return fmt.Errorf("restore bind %s: %w", src, err)
	}
	return nil
}

func copyBind(ctx context.Context, src, dest string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	if _, err := exec.LookPath("rsync"); err != nil {
		return goCopy(ctx, src, dest)
	}
	return rsyncCopy(ctx, src, dest)
}

func rsyncCopy(ctx context.Context, src, dest string) error {
	cmd := exec.CommandContext(ctx, "rsync", "-a", "--delete", "--exclude", ".dosnap", withSlash(src), withSlash(dest))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rsync: %w", err)
	}
	return nil
}

func goCopy(ctx context.Context, src, dest string) error {
	src = filepath.Clean(src)
	dest = filepath.Clean(dest)
	if _, err := os.Stat(src); err != nil {
		return err
	}
	keep := map[string]struct{}{}
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skipDosnap(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		keep[rel] = struct{}{}
		info, err := d.Info()
		if err != nil {
			return err
		}
		target := filepath.Join(dest, rel)
		if d.IsDir() {
			if err := os.MkdirAll(target, info.Mode().Perm()); err != nil {
				return err
			}
			return os.Chmod(target, info.Mode().Perm())
		}
		return copyFile(path, target, info)
	})
	if err != nil {
		return err
	}
	return filepath.WalkDir(dest, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dest, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if skipDosnap(rel) {
			if d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if _, ok := keep[rel]; ok {
			return nil
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		if d.IsDir() {
			return fs.SkipDir
		}
		return nil
	})
}

func copyFile(src, dest string, info os.FileInfo) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if err := os.Chmod(dest, info.Mode().Perm()); err != nil {
		return err
	}
	return os.Chtimes(dest, info.ModTime(), info.ModTime())
}

func skipDosnap(rel string) bool {
	rel = filepath.ToSlash(rel)
	if rel == ".dosnap" || strings.HasPrefix(rel, ".dosnap/") {
		return true
	}
	for _, p := range strings.Split(rel, "/") {
		if p == ".dosnap" {
			return true
		}
	}
	return false
}

func withSlash(p string) string {
	if strings.HasSuffix(p, string(os.PathSeparator)) {
		return p
	}
	return p + string(os.PathSeparator)
}
