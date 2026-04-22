package booking

type Theater struct{
	id string
	name string
	showtimes []*Showtime
}

func NewTheater(id string, name string) *Theater{
	return &Theater{id: id, name:name, showtimes: make([]*Showtime, 0)}
}

func (t *Theater) GetId() string {
	return t.id
}

func (t *Theater) GetName() string {
	return t.name
}

func (t *Theater) GetShowTimes() []*Showtime {
	return t.showtimes
}

func (t *Theater) GetShowtimesForMovie(movie *Movie) []*Showtime {
	var results []*Showtime
	for _, showtime := range t.showtimes {
		if showtime.GetMovie().GetID() == movie.GetID() {
			results = append(results, showtime)
		}
	}
	return results
}