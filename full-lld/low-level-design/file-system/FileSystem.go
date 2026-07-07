package filesystem

import (
	"errors"
	"strings"
)

type FileSystem struct {
	root *Folder
}

func NewFileSystem() *FileSystem {
	return &FileSystem{root: NewFolder("/")}
}

func (fs *FileSystem) CreateFile(path string, content string) (*File, error) {
	if path == "/" {
		return nil, errors.New("cannot create file at root")
	}

	parent, err := fs.resolveParent(path)
	if err != nil {
		return nil, err
	}

	fileName := fs.extractName(path)

	if parent.HasChild(fileName) {
		return nil, errors.New("entry already exists: " + fileName)
	}

	file := NewFile(fileName, content)
	parent.AddChild(file)
	return file, nil
}

func (fs *FileSystem) CreateFolder(path string) (*Folder, error) {
	if path == "/" {
		return nil, errors.New("root already exists")
	}

	parent, err := fs.resolveParent(path)
	if err != nil {
		return nil, err
	}

	folderName := fs.extractName(path)

	if parent.HasChild(folderName) {
		return nil, errors.New("entry already exists: " + folderName)
	}

	folder := NewFolder(folderName)
	parent.AddChild(folder)
	return folder, nil
}

func (fs *FileSystem) Delete(path string) error {
	if path == "/" {
		return errors.New("cannot delete root")
	}

	parent, err := fs.resolveParent(path)
	if err != nil {
		return err
	}

	name := fs.extractName(path)

	removed := parent.RemoveChild(name)
	if removed == nil {
		return errors.New("entry not found: " + path)
	}

	return nil
}

func (fs *FileSystem) List(path string) ([]FileSystemEntry, error) {
	entry, err := fs.resolvePath(path)
	if err != nil {
		return nil, err
	}

	if !entry.IsDirectory() {
		return nil, errors.New("cannot list a file")
	}

	return entry.(*Folder).GetChildren(), nil
}

func (fs *FileSystem) Get(path string) (FileSystemEntry, error) {
	return fs.resolvePath(path)
}

func (fs *FileSystem) Rename(path string, newName string) error {
	if path == "/" {
		return errors.New("cannot rename root")
	}

	if newName == "" || strings.Contains(newName, "/") {
		return errors.New("invalid name")
	}

	parent, err := fs.resolveParent(path)
	if err != nil {
		return err
	}

	oldName := fs.extractName(path)

	if !parent.HasChild(oldName) {
		return errors.New("entry not found")
	}

	if parent.HasChild(newName) {
		return errors.New("entry already exists: " + newName)
	}

	entry := parent.RemoveChild(oldName)
	entry.SetName(newName)
	parent.AddChild(entry)

	return nil
}

func (fs *FileSystem) Move(srcPath string, destPath string) error {
	if srcPath == "/" {
		return errors.New("cannot move root")
	}

	srcParent, err := fs.resolveParent(srcPath)
	if err != nil {
		return err
	}

	srcName := fs.extractName(srcPath)
	entry := srcParent.GetChild(srcName)

	if entry == nil {
		return errors.New("source not found: " + srcPath)
	}

	destParent, err := fs.resolveParent(destPath)
	if err != nil {
		return err
	}

	destName := fs.extractName(destPath)

	if entry.IsDirectory() {
		current := destParent
		for current != nil {
			if current == entry {
				return errors.New("cannot move folder into itself")
			}
			current = current.GetParent()
		}
	}

	if destParent.HasChild(destName) {
		return errors.New("destination already exists: " + destPath)
	}

	srcParent.RemoveChild(srcName)
	entry.SetName(destName)
	destParent.AddChild(entry)

	return nil
}

func (fs *FileSystem) resolvePath(path string) (FileSystemEntry, error) {
	if path == "" {
		return nil, errors.New("path cannot be empty")
	}

	if !strings.HasPrefix(path, "/") {
		return nil, errors.New("path must be absolute")
	}

	if path == "/" {
		return fs.root, nil
	}

	parts := strings.Split(path[1:], "/")
	var current FileSystemEntry = fs.root

	for _, part := range parts {
		if part == "" {
			return nil, errors.New("invalid path: consecutive slashes")
		}

		if !current.IsDirectory() {
			return nil, errors.New("not a directory")
		}

		child := current.(*Folder).GetChild(part)
		if child == nil {
			return nil, errors.New("path not found: " + path)
		}

		current = child
	}

	return current, nil
}

func (fs *FileSystem) resolveParent(path string) (*Folder, error) {
	if path == "/" {
		return nil, errors.New("root has no parent")
	}

	lastSlash := strings.LastIndex(path, "/")
	parentPath := "/"
	if lastSlash > 0 {
		parentPath = path[:lastSlash]
	}

	parent, err := fs.resolvePath(parentPath)
	if err != nil {
		return nil, err
	}

	if !parent.IsDirectory() {
		return nil, errors.New("parent is not a directory")
	}

	return parent.(*Folder), nil
}

func (fs *FileSystem) extractName(path string) string {
	lastSlash := strings.LastIndex(path, "/")
	return path[lastSlash+1:]
}
