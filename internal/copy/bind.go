package copy

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	_, rsyncErr := exec.LookPath("rsync")
	if info.IsDir() {
		if err := ensureDirectory(dest); err != nil {
			return err
		}
		if rsyncErr != nil {
			return goCopy(ctx, src, dest)
		}
		return rsyncCopy(ctx, src, dest, true)
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if rsyncErr != nil {
		return goCopyEntry(ctx, src, dest, info)
	}
	if err := ensureEntryTarget(dest, info); err != nil {
		return err
	}
	return rsyncCopy(ctx, src, dest, false)
}

func rsyncCopy(ctx context.Context, src, dest string, directory bool) error {
	if directory {
		src = withSlash(src)
		dest = withSlash(dest)
	}
	cmd := exec.CommandContext(ctx, "rsync", "-a", "--delete", "--exclude", ".dosnap", "--", src, dest)
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
	dirModes := map[string]os.FileMode{}
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
		if d.Type()&fs.ModeSymlink != 0 {
			return copySymlink(path, target)
		}
		if d.IsDir() {
			if err := ensureDirectory(target); err != nil {
				return err
			}
			dirModes[target] = info.Mode().Perm()
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type %s", path)
		}
		return copyFile(path, target, info)
	})
	if err != nil {
		return err
	}
	if err := filepath.WalkDir(dest, func(path string, d fs.DirEntry, err error) error {
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
	}); err != nil {
		return err
	}
	dirs := make([]string, 0, len(dirModes))
	for path := range dirModes {
		dirs = append(dirs, path)
	}
	sort.Slice(dirs, func(i, j int) bool { return len(dirs[i]) > len(dirs[j]) })
	for _, path := range dirs {
		if err := os.Chmod(path, dirModes[path]); err != nil {
			return err
		}
	}
	return nil
}

func goCopyEntry(ctx context.Context, src, dest string, info os.FileInfo) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if info.Mode()&fs.ModeSymlink != 0 {
		return copySymlink(src, dest)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("unsupported file type %s", src)
	}
	return copyFile(src, dest, info)
}

func copySymlink(src, dest string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(dest); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.Symlink(target, dest)
}

func ensureDirectory(path string) error {
	info, err := os.Lstat(path)
	if err == nil && (!info.IsDir() || info.Mode()&fs.ModeSymlink != 0) {
		if err := os.RemoveAll(path); err != nil {
			return err
		}
	} else if err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.MkdirAll(path, 0o755)
}

func ensureEntryTarget(path string, source os.FileInfo) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if source.Mode()&fs.ModeSymlink != 0 || info.IsDir() || info.Mode()&fs.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return os.RemoveAll(path)
	}
	return nil
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
	if err := ensureEntryTarget(dest, info); err != nil {
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
