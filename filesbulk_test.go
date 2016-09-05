package main

import (
	"os"
	"testing"
)

var file *os.File

const (
	PATH     = "/"
	FILENAME = "README.md"
	FILE1    = "README1.md"
	FILE2    = "README2.md"
)

func init() {
	file, _ = os.Open(FILENAME)
}

func TestFilesBulkPush(t *testing.T) {
	id1, _ := miogo.CreateGFSFile(FILE1, file)
	id2, _ := miogo.CreateGFSFile(FILE2, file)

	fb := NewFilesBulk(PATH)
	fb.AddFile(id1, FILE1)
	fb.AddFile(id2, FILE2)

	miogo.PushFilesBulk(fb)

	if _, ok := miogo.FetchFile(PATH + FILE1); !ok {
		t.Fatal("Files bulk push failed (cannot fetch file 1)")
	}

	if _, ok := miogo.FetchFile(PATH + FILE2); !ok {
		t.Fatal("Files bulk push failed (cannot fetch file 2)")
	}

	miogo.RemoveFile(PATH + FILE1)
	miogo.RemoveFile(PATH + FILE2)
}

func TestFilesBulkRevert(t *testing.T) {
	id1, _ := miogo.CreateGFSFile(FILE1, file)
	id2, _ := miogo.CreateGFSFile(FILE2, file)

	fb := NewFilesBulk(PATH)
	fb.AddFile(id1, FILE1)
	fb.AddFile(id2, FILE2)
	fb.Revert()

	if _, ok := miogo.FetchFile(PATH + FILE1); ok {
		t.Fatal("Files bulk revert failed (can fetch file 1)")
	}

	if _, ok := miogo.FetchFile(PATH + FILE2); ok {
		t.Fatal("Files bulk revert failed (can fetch file 2)")
	}

	if _, err := db.GridFS("fs").OpenId(id1); err == nil {
		t.Fatal("File 1 is still in GridFS")
	}

	if _, err := db.GridFS("fs").OpenId(id2); err == nil {
		t.Fatal("File 1 is still in GridFS")
	}
}
