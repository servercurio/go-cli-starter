package errors

import e "github.com/joomcode/errorx"

var (
	// FileSystemErrors is the errorx namespace for file-system level
	// failures (missing files, permission errors, malformed paths or
	// formats). Wrap upstream os/filepath errors with one of the typed
	// members below so callers can branch on category.
	FileSystemErrors = e.NewNamespace("filesystem")

	// FileAccessDenied marks failure to access a path due to OS permissions.
	// Wrap errors satisfying os.IsPermission with this type.
	FileAccessDenied = FileSystemErrors.NewType("file_access_denied")

	// FileNotFound marks failure to locate a requested path. Carries the
	// errorx NotFound trait so callers can use errorx.IsOfType /
	// errorx.HasTrait checks. Wrap errors satisfying os.IsNotExist with
	// this type.
	FileNotFound = FileSystemErrors.NewType("file_not_found", e.NotFound())

	// InvalidFilePath marks a path that exists but isn't usable in the
	// caller's context (e.g. expected a regular file but got a directory).
	InvalidFilePath = FileSystemErrors.NewType("invalid_file_path")

	// IllegalFileFormat marks a file whose contents could not be parsed in
	// the expected format (YAML, JSON, etc.).
	IllegalFileFormat = FileSystemErrors.NewType("illegal_file_format")
)
