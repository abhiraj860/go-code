package filesystem

type File struct {
	*BaseEntry
	content string
}

func NewFile(name string, content string) *File {
	return &File{
		BaseEntry: NewBaseEntry(name),
		content:   content,
	}
}

func (f *File) GetContent() string {
	return f.content
}

func (f *File) SetContent(content string) {
	f.content = content
}

func (f *File) IsDirectory() bool {
	return false
}
