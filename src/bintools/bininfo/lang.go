package bininfo

import (
	"debug/buildinfo"
	"os"
	"path/filepath"
	"strings"
)

type SourceLanguage string

const (
	GoLang SourceLanguage = "golang"
	Python SourceLanguage = "python"
	Binary SourceLanguage = "binary"
)

func GetSourceLanguage(path string, f *os.File) SourceLanguage {
	if isGoBinary(f) {
		return GoLang
	}
	if isPythonCommand(path) {
		return Python

	}

	return Binary
}

func isGoBinary(f *os.File) bool {
	_, err := buildinfo.Read(f)
	return err == nil
}

func isPythonCommand(path string) bool {
	// Resolve the symlink to the actual executable path
	actualPath, err := os.Readlink(path)
	if err != nil {
		return false
	}

	// Check if the actual path is a python binary - path will be something along /usr/bin/python3
	_, file := filepath.Split(actualPath)
	return strings.Contains(file, "python")
}
