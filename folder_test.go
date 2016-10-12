package main

import "testing"

func TestFormatD(t *testing.T) {
	path := "/folderA/folderB/fic.txt/"
	path = formatD(path)
	if path != "/folderA/folderB/fic.txt" {
		t.Error("formatD doesn't process last /")
	}
	path = ""
	path = formatD(path)
	if path != "/" {
		t.Error("formatD doesn't process empty string correctly")
	}
}

func TestFormatF(t *testing.T) {
	path := "/folderA/folderB/fic.txt"
	path, file := formatF(path)
	if path != "/folderA/folderB" || file != "fic.txt" {
		t.Error("formatF broken")
	}
}
