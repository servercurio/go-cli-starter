package config

import (
	"os"
	"path/filepath"
	"strings"

	ex "github.com/joomcode/errorx"
	"github.com/servercurio/go-cli-starter/internal/errors"
)

// validFilePathFn returns true if a path's FileInfo satisfies a caller-defined
// predicate (e.g. "is a directory" or "is a regular file").
type validFilePathFn func(stat os.FileInfo) bool

// directoryCheck is a validFilePathFn that accepts directories.
var directoryCheck = func(stat os.FileInfo) bool {
	return stat.IsDir()
}

// fileCheck is a validFilePathFn that accepts non-directory paths.
var fileCheck = func(stat os.FileInfo) bool {
	return !stat.IsDir()
}

// FileNameVariants returns the set of file names a config search should try
// for the given base name: ".json", ".yml", and ".yaml" extensions, in that
// order. An existing extension on name is stripped before the variants are
// appended.
func FileNameVariants(name string) []string {
	name = strings.TrimSpace(name)
	if name == "" {
		return []string{}
	}

	name = strings.TrimSuffix(name, filepath.Ext(name))

	return []string{
		name + ".json",
		name + ".yml",
		name + ".yaml",
	}
}

// checkPath stats file and returns the result of checkFn against the
// resulting FileInfo. Translates os.Stat errors into typed errors from the
// internal/errors package so callers can branch on category.
func checkPath(file string, checkFn validFilePathFn) (bool, error) {
	file = strings.TrimSpace(file)
	if file == "" {
		return false, ex.IllegalArgument.New("file path is empty")
	}

	stat, err := os.Stat(file)
	if err != nil {
		if os.IsNotExist(err) {
			return false, errors.FileNotFound.Wrap(err, "file not found: %s", file)
		} else if os.IsPermission(err) {
			return false, errors.FileAccessDenied.Wrap(err, "permission denied: %s", file)
		}

		return false, ex.ExternalError.Wrap(err, "failed to stat file: %s", file)
	}

	return checkFn(stat), nil
}

// isYamlFile reports whether file has a .yaml or .yml extension.
func isYamlFile(file string) bool {
	ext := filepath.Ext(file)
	return ext == ".yaml" || ext == ".yml"
}

// isJsonFile reports whether file has a .json extension.
func isJsonFile(file string) bool {
	ext := filepath.Ext(file)
	return ext == ".json"
}
