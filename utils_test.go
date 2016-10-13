package main

import "testing"

type (
	TestValuesD struct {
		tested, expected string
	}

	TestValuesF struct {
		tested, expectedD, expectedF string
	}
)

func TestFormatD(t *testing.T) {
	tests := []TestValuesD{
		TestValuesD{"/folderA/folderB/fic.txt/", "/folderA/folderB/fic.txt"},
		TestValuesD{"/folderB/fic.txt", "/folderB/fic.txt"},
		TestValuesD{"/fic.txt     ", "/fic.txt"},
		TestValuesD{"", "/"},
	}

	for _, test := range tests {
		if got := formatD(test.tested); got != test.expected {
			t.Errorf(`formatD of "%s": expected "%s", got "%s"`, test.tested, test.expected, got)
		}
	}
}

func TestFormatF(t *testing.T) {
	tests := []TestValuesF{
		TestValuesF{"/folderA/folderB/fic.txt/", "/folderA/folderB", "fic.txt"},
		TestValuesF{"/folderB/fic.txt", "/folderB", "fic.txt"},
		TestValuesF{"/fic.txt     ", "/", "fic.txt"},
		TestValuesF{"/", "/", ""},
		TestValuesF{"", "/", ""},
	}

	for _, test := range tests {
		if gotD, gotF := formatF(test.tested); gotD != test.expectedD || gotF != test.expectedF {
			t.Errorf(`formatF of "%s": expected "%s" and "%s", got "%s" and "%s"`, test.tested, test.expectedD, test.expectedF, gotD, gotF)
		}
	}
}
