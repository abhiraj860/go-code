package booking

type Movie struct {
	id string
	title string
}

func NewMovie(id string, title string) *Movie {
	return &Movie{id: id, title: title}	
}

func (m *Movie) GetID() string {
	return m.id
}

func (m *Movie) GetTitle() string {
	return m.title
}

