package serve

import (
	"io/fs"
	"os"
	"sort"
)

// NewFS combines the top level directories of multiple fs.FS.
func NewFS(vfs ...fs.FS) fs.FS {
	return stackedFS(vfs)
}

// NewFSFromDirs combines the top level directories of multiple directories.
func NewFSFromDirs(dirs ...string) fs.FS {
	result := make([]fs.FS, len(dirs))
	for i, dir := range dirs {
		result[i] = os.DirFS(dir)
	}
	return stackedFS(result)
}

type stackedFS []fs.FS

// Open opens the the first occurrence of named file.
func (s stackedFS) Open(name string) (f fs.File, err error) {
	for _, vfs := range s {
		if f, err = vfs.Open(name); err == nil {
			return f, nil
		}
	}
	return nil, err
}

// ReadDir combines all files on the stack, sorted by stack order first
// and alphabetically within the stack second. Directories are not merged.
func (s stackedFS) ReadDir(name string) (result []fs.DirEntry, err error) {
	seen := map[string]bool{}
	for _, vfs := range s {
		entries, err := fs.ReadDir(vfs, name)
		if err != nil {
			return nil, err
		}
		byName := func(i, j int) bool { return entries[i].Name() < entries[j].Name() }
		sort.Slice(entries, byName)
		for _, entry := range entries {
			if !seen[entry.Name()] {
				seen[entry.Name()] = true
				result = append(result, entry)
			}
		}
	}
	return result, nil
}
