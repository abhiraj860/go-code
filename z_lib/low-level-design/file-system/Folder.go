package filesystem

type Folder struct {
	*BaseEntry
	children map[string]FileSystemEntry
}

func NewFolder(name string) *Folder {
	return &Folder{
		BaseEntry: NewBaseEntry(name),
		children:  make(map[string]FileSystemEntry),
	}
}

func (f *Folder) IsDirectory() bool {
	return true
}

func (f *Folder) AddChild(entry FileSystemEntry) bool {
	if entry == nil {
		return false
	}

	if _, exists := f.children[entry.GetName()]; exists {
		return false
	}

	f.children[entry.GetName()] = entry
	entry.SetParent(f)
	return true
}

func (f *Folder) RemoveChild(name string) FileSystemEntry {
	entry, exists := f.children[name]
	if !exists {
		return nil
	}

	delete(f.children, name)
	entry.SetParent(nil)
	return entry
}

func (f *Folder) GetChild(name string) FileSystemEntry {
	return f.children[name]
}

func (f *Folder) HasChild(name string) bool {
	_, exists := f.children[name]
	return exists
}

func (f *Folder) GetChildren() []FileSystemEntry {
	result := make([]FileSystemEntry, 0, len(f.children))
	for _, child := range f.children {
		result = append(result, child)
	}
	return result
}
