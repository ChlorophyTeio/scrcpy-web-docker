package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

type coreBackupManifest struct {
	Kind    string `json:"kind"`
	Version string `json:"version"`
	Created string `json:"created"`
}

func coreCopyTree(ctx context.Context, src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err = ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if entry.Type()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			return os.Symlink(link, target)
		}
		if !info.Mode().IsRegular() {
			return errors.New("备份目录含不支持的特殊文件")
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}

func coreBackupInfo(data string) (coreBackupManifest, error) {
	var m coreBackupManifest
	b, err := os.ReadFile(filepath.Join(data, "web-core-backup", "manifest.json"))
	if err != nil {
		return m, err
	}
	if json.Unmarshal(b, &m) != nil || m.Kind != "maa-web-core-backup" || m.Version == "" {
		return m, errors.New("备份标识无效")
	}
	return m, nil
}

func coreCreateBackup(ctx context.Context, data, version string) error {
	stage, err := os.MkdirTemp(data, ".web-core-backup-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stage)
	for _, name := range []string{"lib", "resource"} {
		if err := coreCopyTree(ctx, filepath.Join(data, name), filepath.Join(stage, name)); err != nil {
			return err
		}
	}
	b, _ := json.Marshal(coreBackupManifest{Kind: "maa-web-core-backup", Version: version, Created: time.Now().Format(time.RFC3339)})
	if err := os.WriteFile(filepath.Join(stage, "manifest.json"), b, 0600); err != nil {
		return err
	}
	target := filepath.Join(data, "web-core-backup")
	old := stage + ".previous"
	if _, err := os.Lstat(target); err == nil {
		if _, err := coreBackupInfo(data); err != nil {
			return err
		}
		if err := os.Rename(target, old); err != nil {
			return err
		}
	}
	if err := os.Rename(stage, target); err != nil {
		_ = os.Rename(old, target)
		return err
	}
	return os.RemoveAll(old)
}

// Prepare both trees before swapping, then undo completed swaps if a rename fails.
func coreRestoreBackup(ctx context.Context, data string) (string, error) {
	m, err := coreBackupInfo(data)
	if err != nil {
		return "", err
	}
	stage, err := os.MkdirTemp(data, ".web-core-restore-")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(stage)
	for _, name := range []string{"lib", "resource"} {
		if err := coreCopyTree(ctx, filepath.Join(data, "web-core-backup", name), filepath.Join(stage, name)); err != nil {
			return "", err
		}
	}
	if err := coreSwapTrees(data, stage); err != nil {
		return "", err
	}
	return m.Version, nil
}

// The staging directory must be on the same filesystem as the active trees.
// Rename old trees away instead of unlinking loaded libraries on Unraid FUSE.
func coreSwapTrees(data, stage string) error {
	old, err := os.MkdirTemp(data, ".web-core-replaced-")
	if err != nil {
		return err
	}
	cleanupOld := false
	defer func() {
		if cleanupOld {
			_ = os.RemoveAll(old)
		}
	}()
	moved := []string{}
	installed := map[string]bool{}
	undo := func() error {
		var errs []error
		for i := len(moved) - 1; i >= 0; i-- {
			name := moved[i]
			if installed[name] {
				if e := os.RemoveAll(filepath.Join(data, name)); e != nil {
					errs = append(errs, e)
					continue
				}
			}
			if e := os.Rename(filepath.Join(old, name), filepath.Join(data, name)); e != nil {
				errs = append(errs, e)
			}
		}
		cleanupOld = len(errs) == 0
		return errors.Join(errs...)
	}
	for _, name := range []string{"lib", "resource"} {
		if err := os.Rename(filepath.Join(data, name), filepath.Join(old, name)); err != nil {
			return errors.Join(err, undo())
		}
		moved = append(moved, name)
		if err := os.Rename(filepath.Join(stage, name), filepath.Join(data, name)); err != nil {
			return errors.Join(err, undo())
		}
		installed[name] = true
	}
	cleanupOld = true
	return nil
}
