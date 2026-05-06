package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	ex "github.com/joomcode/errorx"
	"github.com/servercurio/go-echo-starter/internal/errors"
	"github.com/servercurio/go-echo-starter/internal/logging"
	"gopkg.in/yaml.v3"
)

// FromPaths searches each path in paths for a config file matching name (with
// the extensions returned by FileNameVariants) and unmarshals every match
// into cfg in the order found, so later paths override earlier ones. Missing
// directories or files are skipped silently; an unmarshal error is returned
// with a stack trace.
func FromPaths(cfg interface{}, name string, paths ...string) error {
	if name == "" {
		return ex.IllegalArgument.New("name is empty")
	}

	if cfg == nil {
		return ex.IllegalArgument.New("config is nil")
	}

	if len(paths) == 0 {
		return ex.IllegalArgument.New("paths are empty")
	}

	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		p = filepath.Clean(p)

		if ok, err := checkPath(p, directoryCheck); err != nil || !ok {
			continue
		}

		for _, qualifiedName := range FileNameVariants(name) {
			file := filepath.Join(p, qualifiedName)
			if ok, err := checkPath(file, fileCheck); err != nil || !ok {
				continue
			}

			file, err := filepath.Abs(file)
			if err != nil {
				continue
			}

			if err := FromFile(cfg, file); err != nil {
				return ex.EnsureStackTrace(err)
			}

			logging.Daemon.
				Debug().
				Str("path", filepath.Dir(file)).
				Str("name", filepath.Base(file)).
				Msg("loaded config file")
		}
	}

	return nil
}

// FromFile reads the file at the given path and unmarshals it into cfg using
// YAML or JSON based on the file extension. Returns errors.IllegalFileFormat
// for unsupported extensions or malformed content, errors.InvalidFilePath
// when the path resolves to a directory, and a wrapped errorx for I/O
// failures.
func FromFile(cfg interface{}, file string) error {
	file = strings.TrimSpace(file)

	if cfg == nil {
		return ex.IllegalArgument.New("config is nil")
	}

	if file == "" {
		return ex.IllegalArgument.New("file path is empty")
	}

	file = filepath.Clean(file)

	if ok, err := checkPath(file, fileCheck); err != nil {
		return ex.EnsureStackTrace(err)
	} else if !ok {
		return errors.InvalidFilePath.New("file is a directory, expected a regular file: %s", file)
	}

	if !isYamlFile(file) && !isJsonFile(file) {
		return errors.IllegalFileFormat.New("unsupported file format: %s", file)
	}

	bytes, err := os.ReadFile(file)
	if err != nil {
		return ex.ExternalError.Wrap(err, "failed to read file: %s", file)
	}

	if isYamlFile(file) {
		if err := yaml.Unmarshal(bytes, cfg); err != nil {
			return errors.IllegalFileFormat.Wrap(err, "failed to unmarshal YAML file: %s", file)
		}
	} else {
		if err := json.Unmarshal(bytes, cfg); err != nil {
			return errors.IllegalFileFormat.Wrap(err, "failed to unmarshal JSON file: %s", file)
		}
	}

	return nil
}
