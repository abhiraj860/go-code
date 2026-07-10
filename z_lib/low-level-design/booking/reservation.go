package booking


type Reservation struct {
	confirmationID string
	showtime *Showtime
	seatIDs []string
}

func NewReservation(confirmationID string, showtime *Showtime, seatIDs []string) *Reservation {
	copied := make([]string, len(seatIDs))
	copy(copied, seatIDs)
	return &Reservation{
		confirmationID: confirmationID,
		showtime: showtime,
		seatIDs: copied,
	}
}

func (r *Reservation) GetConfirmationID() string {
	return r.confirmationID
}

func (r *Reservation) GetShowTime() *Showtime {
	return r.showtime
}

func (r *Reservation) GetSeatIDs() []string {
	copied := make([]string, len(r.seatIDs))
	copy(copied, r.seatIDs)
	return copied
}

