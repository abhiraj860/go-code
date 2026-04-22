package booking

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"time"
)

type BookingSystem struct {
	theaters []*Theater
	moviesById map[string]*Movie
	showtimesByMovieId map[string][]*Showtime
	showtimesById map[string]*Showtime
	reservationsById map[string]*Reservation
}

func NewBookingSystem(theaters []*Theater) *BookingSystem {
	bs := &BookingSystem{
		theaters: theaters,
		moviesById: make(map[string]*Movie),
		showtimesByMovieId: make(map[string][]*Showtime),
		showtimesById: make(map[string]*Showtime),
		reservationsById: make(map[string] *Reservation),
	}	

	for _, theater := range theaters {
		for _, showtime := range theater.GetShowTimes() {
			movie := showtime.GetMovie()
			bs.moviesById[movie.GetID()] = movie
			bs.showtimesById[movie.GetID()] = showtime
			bs.showtimesByMovieId[movie.GetID()] = append(bs.showtimesByMovieId[movie.GetID()], showtime)
		}
	}
	return bs
}


func (bs *BookingSystem) SearchMovies(title string) []*Showtime {
	if title == "" {
		return nil
	}
	var results []*Showtime
	searchLower := strings.ToLower(title)
	now := time.Now()
	for _, movie := range bs.moviesById {
		if strings.Contains(strings.ToLower(movie.GetTitle()), searchLower) {
			movieShowtimes := bs.showtimesByMovieId[movie.GetID()]
			for _, showtime := range movieShowtimes {
				if showtime.GetDateTime().After(now) {
					results = append(results, showtime)
				}
			}
		}
	}
	return results
}


func (bs *BookingSystem) GetShowtimesAtTheater(theater *Theater) [] *Showtime {
	if theater == nil {
		return nil
	}

	var results []*Showtime
	now := time.Now()

	for _, showtime := range theater.GetShowTimes() {
		if showtime.GetDateTime().After(now) {
			results = append(results, showtime)
		}
	}
	return results
}


func (bs *BookingSystem) Book(showtimeID string, seatIDs []string) (*Reservation, error) {
	if showtimeID == "" || len(seatIDs) == 0 {
		return nil, errors.New("invalid booking request")
	}

	showtime, ok := bs.showtimesById[showtimeID]
	if !ok {
		return nil, fmt.Errorf("showtime not found: %s", showtimeID)
	}
	reservation := NewReservation(
		generateConfirmationID(),
		showtime,
		seatIDs,
	)
	if err := showtime.Book(reservation); err != nil {
		return nil, err
	}
	bs.reservationsById[reservation.GetConfirmationID()] = reservation
	return reservation, nil
}



func (bs *BookingSystem) CancelReservation(confirmationID string) error {
	if confirmationID == "" {
		return errors.New("invalid confirmation ID")
	}
	reservation, ok := bs.reservationsById[confirmationID]
	if !ok {
		return fmt.Errorf("reservation not found: %s", confirmationID);
	}
	showtime := reservation.GetShowTime()
	showtime.Cancel(reservation)

	delete(bs.reservationsById, confirmationID)

	return nil
}

func generateConfirmationID() string {
	b := make([] byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}
