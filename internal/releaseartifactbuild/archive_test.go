package releaseartifactbuild

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestParseFileListRequiresSortedSafePaths(t *testing.T) {
	paths, err := parseFileList("a.txt\nb/c.txt\n")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(paths, []string{"a.txt", "b/c.txt"}) {
		t.Fatalf("paths = %#v", paths)
	}
	for _, value := range []string{
		"b.txt\na.txt\n",
		"a.txt\na.txt\n",
		"../a.txt\n",
		"/a.txt\n",
		"a\\b.txt\n",
		"a.txt",
		"a.txt\r\n",
	} {
		if _, err := parseFileList(value); err == nil {
			t.Fatalf("expected invalid file list: %q", value)
		}
	}
}

func TestDeterministicTarGzipIsByteIdentical(t *testing.T) {
	files := []sourceFile{
		{Path: "b.txt", Mode: 0o644, Data: []byte("bravo\n")},
		{Path: "a/tool", Mode: 0o755, Data: []byte("alpha\n")},
	}
	first := filepath.Join(t.TempDir(), "first.tar.gz")
	second := filepath.Join(t.TempDir(), "second.tar.gz")
	if err := writeDeterministicTarGzip(first, "bundle", files); err != nil {
		t.Fatal(err)
	}
	if err := writeDeterministicTarGzip(second, "bundle", files); err != nil {
		t.Fatal(err)
	}
	firstData, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondData, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstData, secondData) {
		t.Fatal("deterministic archives differ")
	}

	gzipReader, err := gzip.NewReader(bytes.NewReader(firstData))
	if err != nil {
		t.Fatal(err)
	}
	if !gzipReader.ModTime.IsZero() {
		t.Fatalf("gzip timestamp = %s", gzipReader.ModTime)
	}
	tarReader := tar.NewReader(gzipReader)
	var names []string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, header.Name)
		if !header.ModTime.Equal(time.Unix(0, 0).UTC()) || header.Uid != 0 || header.Gid != 0 {
			t.Fatalf("nondeterministic header: %#v", header)
		}
	}
	if !reflect.DeepEqual(names, []string{"bundle/a/tool", "bundle/b.txt"}) {
		t.Fatalf("archive names = %#v", names)
	}
}
