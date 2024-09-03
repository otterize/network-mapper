package bininfo

import (
	"debug/buildinfo"
	"os"
	"path/filepath"
	"strings"
)

type SourceLanguage string

const (
	SourceLanguageGoLang SourceLanguage = "golang"
	SourceLanguagePython SourceLanguage = "python"
	SourceLanguageNodeJs SourceLanguage = "nodejs"
	SourceLanguageBinary SourceLanguage = "binary"
)

func GetSourceLanguage(path string, f *os.File) SourceLanguage {
	if isGoBinary(f) {
		return SourceLanguageGoLang
	}
	if isPythonCommand(path) {
		return SourceLanguagePython

	}
	if isNodeCommand(path) {
		return SourceLanguageNodeJs
	}

	return SourceLanguageBinary
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

func isNodeCommand(path string) bool {
	// Resolve the symlink to the actual executable path
	actualPath, err := os.Readlink(path)
	if err != nil {
		return false
	}

	// Check if the actual path is a python binary - path will be something along /usr/bin/node
	_, file := filepath.Split(actualPath)
	return strings.Contains(file, "node")
}
