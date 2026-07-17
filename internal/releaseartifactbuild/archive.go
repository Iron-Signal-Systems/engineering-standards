package releaseartifactbuild

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxArchiveInput = 16 * 1024 * 1024

type sourceFile struct {
	Path string
	Mode int64
	Data []byte
}

func buildArchive(ctx context.Context, runner commandRunner, boundary sourceBoundary, listPath, prefix, outputPath string) error {
	listData, _, err := runner.Run(ctx, boundary.Root, nil, "git", "show", boundary.Commit+":"+listPath)
	if err != nil {
		return fmt.Errorf("read archive file list %s", listPath)
	}
	paths, err := parseFileList(listData)
	if err != nil {
		return fmt.Errorf("validate archive file list %s: %w", listPath, err)
	}

	files := make([]sourceFile, 0, len(paths))
	var total int
	for _, filePath := range paths {
		typeOutput, _, err := runner.Run(ctx, boundary.Root, nil, "git", "cat-file", "-t", boundary.Commit+":"+filePath)
		if err != nil || strings.TrimSpace(typeOutput) != "blob" {
			return fmt.Errorf("archive input %s is not a tracked blob", filePath)
		}
		modeOutput, _, err := runner.Run(ctx, boundary.Root, nil, "git", "ls-tree", boundary.Commit, "--", filePath)
		if err != nil {
			return fmt.Errorf("read archive input mode for %s", filePath)
		}
		mode, err := parseTreeMode(modeOutput, filePath)
		if err != nil {
			return err
		}
		content, _, err := runner.Run(ctx, boundary.Root, nil, "git", "show", boundary.Commit+":"+filePath)
		if err != nil {
			return fmt.Errorf("read archive input %s", filePath)
		}
		total += len(content)
		if total > maxArchiveInput {
			return errors.New("archive source content exceeds the bounded input limit")
		}
		files = append(files, sourceFile{Path: filePath, Mode: mode, Data: []byte(content)})
	}
	return writeDeterministicTarGzip(outputPath, prefix, files)
}

func parseFileList(value string) ([]string, error) {
	if strings.Contains(value, "\r") || !strings.HasSuffix(value, "\n") {
		return nil, errors.New("file list must use LF endings and end with a newline")
	}
	lines := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
	if len(lines) == 0 {
		return nil, errors.New("file list is empty")
	}
	previous := ""
	for _, line := range lines {
		if line == "" || strings.TrimSpace(line) != line || strings.Contains(line, "\\") {
			return nil, errors.New("file list contains an invalid path")
		}
		cleaned := path.Clean(line)
		if cleaned != line || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") || cleaned == ".git" || strings.HasPrefix(cleaned, ".git/") {
			return nil, errors.New("file list contains an unsafe path")
		}
		if previous != "" && line <= previous {
			return nil, errors.New("file list entries must be unique and sorted")
		}
		previous = line
	}
	return lines, nil
}

func parseTreeMode(value, expectedPath string) (int64, error) {
	line := strings.TrimSpace(value)
	metadata, filePath, found := strings.Cut(line, "\t")
	if !found || filePath != expectedPath {
		return 0, fmt.Errorf("unexpected Git tree record for %s", expectedPath)
	}
	fields := strings.Fields(metadata)
	if len(fields) != 3 || fields[1] != "blob" {
		return 0, fmt.Errorf("unexpected Git tree metadata for %s", expectedPath)
	}
	mode, err := strconv.ParseInt(fields[0], 8, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid Git tree mode for %s", expectedPath)
	}
	switch mode {
	case 0o100644:
		return 0o644, nil
	case 0o100755:
		return 0o755, nil
	default:
		return 0, fmt.Errorf("unsupported Git tree mode for %s", expectedPath)
	}
}

func writeDeterministicTarGzip(outputPath, prefix string, files []sourceFile) (err error) {
	if path.Clean(prefix) != prefix || prefix == "." || strings.HasPrefix(prefix, "/") || strings.Contains(prefix, "\\") {
		return errors.New("archive prefix is invalid")
	}
	if len(files) == 0 {
		return errors.New("archive requires at least one file")
	}
	copyFiles := append([]sourceFile(nil), files...)
	sort.Slice(copyFiles, func(i, j int) bool { return copyFiles[i].Path < copyFiles[j].Path })

	file, err := os.OpenFile(outputPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := file.Close()
		if err == nil && closeErr != nil {
			err = closeErr
		}
		if err != nil {
			_ = os.Remove(outputPath)
		}
	}()

	gzipWriter, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		return err
	}
	gzipWriter.Header.ModTime = time.Time{}
	gzipWriter.Header.OS = 255
	tarWriter := tar.NewWriter(gzipWriter)

	for _, source := range copyFiles {
		name := path.Join(prefix, source.Path)
		header := &tar.Header{
			Name:       name,
			Mode:       source.Mode,
			Size:       int64(len(source.Data)),
			ModTime:    time.Unix(0, 0).UTC(),
			AccessTime: time.Time{},
			ChangeTime: time.Time{},
			Uid:        0,
			Gid:        0,
			Uname:      "",
			Gname:      "",
			Typeflag:   tar.TypeReg,
			Format:     tar.FormatPAX,
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(tarWriter, bytes.NewReader(source.Data)); err != nil {
			return err
		}
	}
	if err := tarWriter.Close(); err != nil {
		return err
	}
	if err := gzipWriter.Close(); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Chmod(0o644); err != nil {
		return err
	}
	return nil
}

func archiveOutputPath(directory, name string) string {
	return filepath.Join(directory, filepath.FromSlash(name))
}
