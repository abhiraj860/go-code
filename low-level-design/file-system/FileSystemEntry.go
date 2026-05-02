package filesystem

type FileSystemEntry interface {
	GetName() string
	SetName(name string)
	GetParent() *Folder
	SetParent(parent *Folder)
	GetPath() string
	IsDirectory() bool
}

type BaseEntry struct {
	name   string
	parent *Folder
}

func NewBaseEntry(name string) *BaseEntry {
	return &BaseEntry{name: name, parent: nil}
}

func (e *BaseEntry) GetName() string {
	return e.name
}

func (e *BaseEntry) SetName(name string) {
	e.name = name
}

func (e *BaseEntry) GetParent() *Folder {
	return e.parent
}

func (e *BaseEntry) SetParent(parent *Folder) {
	e.parent = parent
}

func (e *BaseEntry) GetPath() string {
	if e.parent == nil {
		return e.name
	}

	parentPath := e.parent.GetPath()
	if parentPath == "/" {
		return "/" + e.name
	}
	return parentPath + "/" + e.name
}
