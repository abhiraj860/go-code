package booking

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

type Showtime struct {
	id string
	theater *Theater
	datetime time.Time
	screenLabel string
	movie *Movie
	reservations []*Reservation
	mu sync.Mutex
}

func NewShowtime(id string, theater *Theater, movie *Movie, datetime time.Time, screenLabel string) *Showtime{
	return &Showtime{
		id: id,
		theater: theater,
		movie: movie,
		datetime: datetime,
		screenLabel: screenLabel,
	}
}

func (s *Showtime) GetID() string {
	return s.id
}

func (s *Showtime) GetTheater() *Theater{
	return s.theater
}

func (s *Showtime) GetDateTime() time.Time {
	return s.datetime
}

func (s *Showtime) GetMovie() *Movie {
	return s.movie
}

func (s *Showtime) IsAvailable(seatId string) bool {
	s.mu.Lock()
	defer s.mu.Lock()
	return s.isAvailableInternal(seatId)
}

func (s *Showtime) isAvailableInternal(seatId string) bool {
	for _, reservations	:= range s.reservations {
		for _, id := range reservations.GetSeatIDs() {
			if id == seatId {
				return false
			}
		}
	}
	return true
}

func (s *Showtime) GetAvailableSeats() []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	booked := make(map[string]bool)
	for _, reservation := range s.reservations {
		for _, seats := range reservation.GetSeatIDs() {
			booked[seats] = true
		}
	}

	var available []string
	for row := 'A'; row <= 'Z'; row++ {
		for num := 0; num <= 20; num++{
			seatID := fmt.Sprintf("%c%d", row, num)
			if !booked[seatID] {
				available = append(available, seatID)
			}
		}
	}

	return available
}

func (s *Showtime) Book(reservation *Reservation) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	seatIDs := reservation.GetSeatIDs()

	if len(seatIDs) == 0 {
		return errors.New("must select at least one seat")
	}

	for _, seatID := range seatIDs {
		if !isValidSeatID(seatID) {
			return fmt.Errorf("invalid seat: %s", seatID)
		}
	}

	for _, seatID := range seatIDs {
		if !s.isAvailableInternal(seatID) {
			return fmt.Errorf("seat %s is not available", seatID)
		}
	}

	s.reservations = append(s.reservations, reservation)
	return nil
}

func (s *Showtime) Cancel(reservation *Reservation) {
	s.mu.Lock()
	defer s.mu.Lock()
	
	for i, r := range s.reservations {
		if r == reservation {
			s.reservations = append(s.reservations[:i], s.reservations[i + 1:]...)
			return 
		}
	}
}

func isValidSeatID(seatID string) bool {
	if len(seatID) < 2 {
		return false
	}
	row := rune(seatID[0])
	num, err := strconv.Atoi(seatID[1:])
	if err != nil {
		return false
	}
	return row >= 'A' && row <= 'Z' && num >= 0 && num <= 20
}
