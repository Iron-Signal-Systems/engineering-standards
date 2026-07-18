package projectpin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const maxGitErrorBytes = 16 * 1024

// LoadCommitted loads the exact project pin committed at HEAD and proves that
// both the index and working-tree copy are byte-identical to that commit.
func LoadCommitted(ctx context.Context, root string) (Pin, error) {
	path := filepath.Join(root, filepath.FromSlash(MetadataPath))
	if err := requireRegularPinPath(root, path); err != nil {
		return Pin{}, err
	}
	working, err := readBoundedFile(path, MaxFileSize)
	if err != nil {
		return Pin{}, err
	}
	committed, err := readGitBlob(ctx, root, "HEAD:"+MetadataPath)
	if err != nil {
		return Pin{}, errors.New("project pin is not committed at target HEAD")
	}
	indexed, err := readGitBlob(ctx, root, ":"+MetadataPath)
	if err != nil {
		return Pin{}, errors.New("project pin is not present in the target index")
	}
	if !bytes.Equal(committed, indexed) {
		return Pin{}, errors.New("staged project pin differs from target HEAD")
	}
	if !bytes.Equal(committed, working) {
		return Pin{}, errors.New("working-tree project pin differs from target HEAD")
	}
	pin, err := Parse(committed)
	if err != nil {
		return Pin{}, fmt.Errorf("validate committed project pin: %w", err)
	}
	return pin, nil
}

func readGitBlob(ctx context.Context, root, object string) ([]byte, error) {
	command := exec.CommandContext(ctx, "git", "show", object)
	command.Dir = root
	var stdout boundedBuffer
	stdout.limit = MaxFileSize
	var stderr boundedBuffer
	stderr.limit = maxGitErrorBytes
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("read Git object: %s", strings.TrimSpace(stderr.String()))
	}
	if stdout.exceeded {
		return nil, fmt.Errorf("committed project pin exceeds %d-byte limit", MaxFileSize)
	}
	return append([]byte(nil), stdout.Bytes()...), nil
}

func readBoundedFile(path string, limit int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read project pin %s: %w", MetadataPath, err)
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, fmt.Errorf("read project pin %s: %w", MetadataPath, err)
	}
	if len(data) > limit {
		return nil, fmt.Errorf("project pin exceeds %d-byte limit", limit)
	}
	return data, nil
}

func requireRegularPinPath(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("project pin path escapes the target repository")
	}
	current := root
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if err != nil {
			return errors.New("project pin path is unavailable")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return errors.New("project pin path contains a symbolic link")
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return errors.New("project pin is not a regular file")
	}
	return nil
}

type boundedBuffer struct {
	bytes.Buffer
	limit    int
	exceeded bool
}

func (buffer *boundedBuffer) Write(data []byte) (int, error) {
	remaining := buffer.limit - buffer.Len()
	if remaining <= 0 {
		buffer.exceeded = true
		return len(data), nil
	}
	if len(data) > remaining {
		buffer.exceeded = true
		_, _ = buffer.Buffer.Write(data[:remaining])
		return len(data), nil
	}
	return buffer.Buffer.Write(data)
}
