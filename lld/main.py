# Problem - 02 - BookMyShow
from enum import Enum, auto
from abc import ABC, abstractmethod
from collections import defaultdict
import copy

class SeatType(Enum):
    REGULAR = auto()
    PREMIUM = auto()

class SeatStatus(Enum):
    AVAILABLE = auto()
    HELD = auto()
    BOOKED = auto()
    PENDING = auto()
    CONFIRMED = auto()
    CANCELLED = auto()    
    
class PaymentMethod(ABC):
    @abstractmethod
    def pay(self, amount):...
    
    @abstractmethod
    def refund(self, amount):...
    
     
class CardPayment(PaymentMethod):
    def __init__(self, limit):
        self.limit = limit
        
    def pay(self, amount):
        if amount <= self.limit:
            self.limit -= amount
            return True
        return False
    
    def refund(self, amount):
        self.limit += amount

class UpiPayment(PaymentMethod):
    def __init__(self, balance):
        self.balance = balance
    
    def pay(self, amount):
        if amount <= self.balance:
            self.balance -= amount
            return True
        return False
    
    def refund(self, amount):
        self.balance += amount
    
    def get_balance(self):
        return self.balance

class Seat:
    def __init__(self, seat_number, seat_type):
        self.seat_number = seat_number
        self.seat_type = seat_type
    
    def get_seat_number(self):
        return self.seat_number
    
    def get_seat_type(self):
        return self.seat_type
    
class ShowSeat:
    def __init__(self,  seat: Seat):
        self.seat = seat
        self.status = SeatStatus.AVAILABLE
    
    def get_seat_id(self):
        return self.seat.get_seat_number()
            
    def get_seat(self):
        return self.seat
    
    def get_status(self):
        return self.status
    
    def held(self):
        if self.status == SeatStatus.AVAILABLE:
            self.status = SeatStatus.HELD
            return True
        return False
    
    def confirm(self):
        if self.status == SeatStatus.PENDING or self.status == SeatStatus.HELD:
            self.status = SeatStatus.CONFIRMED
            return True
        return False
    
    def book(self):
        if self.status == SeatStatus.CONFIRMED:
            self.status = SeatStatus.BOOKED
            return True
        return False
    
    def pending(self):
        if self.status == SeatStatus.HELD or self.status == SeatStatus.CONFIRMED:
            self.status = SeatStatus.PENDING
            return True
        return False
    
    def release(self): 
        self.status == SeatStatus.AVAILABLE
        return True
    
    def cancel(self):
        if self.status == SeatStatus.BOOKED or self.status == SeatStatus.HELD:
            self.status = SeatStatus.AVAILABLE
            return True
        return False

class Show:
    def __init__(self, show_id, prices_by_type):
        self.show_id = show_id
        self.prices_by_type = prices_by_type
        self.seat_show_seat: dict[str, ShowSeat] = {}
                
    def get_show_id(self):
        return self.show_id
         
    def add_seat(self, seat_number, seat_type):
        seat = Seat(seat_number, seat_type)
        show_seat = ShowSeat(seat)
        self.seat_show_seat[show_seat.get_seat_id()] = show_seat 
        
    def get_show_seat(self, seat_number):
        return self.seat_show_seat.get(seat_number, None)
    
    def get_price_by_type(self):
        return self.prices_by_type
        
    
class Booking:
    def __init__(self, booking_id, show, seats):
        self.booking_id = booking_id
        self.show:Show = show
        self.seats_show_seats: list[ShowSeat] = seats
    
    def get_booking_id(self):
        return self.booking_id
    
    def get_show_seat_list(self):
        return self.seats_show_seats
    
    def hold(self):
        for seat in self.seats_show_seats:
            if not seat.held():
                return False
        return True
    
    def confirm(self):
        for seat in self.seats_show_seats:
            if not seat.confirm():
                return False
        return True
    
    def pending(self):
        for seat in self.seats_show_seats:
            if not seat.pending():
                return False
        return True
    
    def cancelled(self):
        for seat in self.seats_show_seats:
            if not seat.cancel():
                return False
        return True
    
    def booked(self):
        for seat in self.seats_show_seats:
            if not seat.book():
                return False
        return True

    def available(self):
        for seat in self.seats_show_seats:
            if not seat.release():
                return False
        return True
    def get_total(self):
        balance = 0
        for seat in self.seats_show_seats:
            seat_type = seat.get_seat().get_seat_type()
            balance += self.show.get_price_by_type()[seat_type] 
        return balance
     
    def get_status(self):
        return self.seats_show_seats[0].get_status()
    

class BookingService:
    def __init__(self):
        self.shows: dict[str,Show] = {}
        self.booking: dict[str, Booking] = {}
        self.user_booking: dict[str, str] = {}
        self.cnt = 0
        self.payment = None
        
    def add_show(self, show: Show):
        self.shows[show.get_show_id()] = show 
        
    def hold_seats(self, show_id, seat_numbers, user_id):
        show = self.shows.get(show_id, None)
        if show:
            seats:list[ShowSeat] = []
            for seat_num in seat_numbers:
                seat = show.get_show_seat(seat_num)
                if seat is None:
                    return None
                seats.append(seat)
                if show.get_show_seat(seat_num).get_status() != SeatStatus.AVAILABLE:
                    seats.clear()
                    return None
            self.cnt += 1
            b1 = Booking("B" + str(self.cnt), show, seats)
            b1.hold()
            self.user_booking[user_id] = b1.get_booking_id()
            self.booking[b1.get_booking_id()] = b1
            return b1
        return None
         
        
    def confirm_booking(self, booking_id, payment: PaymentMethod):
        booking = self.booking.get(booking_id, None)
        if booking is None:
            return False
        total_amount = booking.get_total()
        self.payment = payment
        booking.confirm()
        if payment.pay(total_amount):
            booking.booked()
            return True
        booking.pending()
        return False 
    
    def cancel_booking(self, booking_id):
        booking = self.booking.get(booking_id, None)
        if booking is None:
            return False
        total = booking.get_total()
        if booking.cancelled():
            self.payment.refund(total)
            return True
        return False
        
    
show = Show("S1", {SeatType.REGULAR: 200, SeatType.PREMIUM: 350})
show.add_seat("A1", SeatType.REGULAR)
show.add_seat("A2", SeatType.REGULAR)
show.add_seat("P1", SeatType.PREMIUM)
svc = BookingService()
svc.add_show(show)
b1 = svc.hold_seats("S1", ["A1", "P1"], "u1")
print(b1.get_booking_id(), b1.get_total())
print(svc.hold_seats("S1", ["A2", "P1"], "u2"))
print(show.get_show_seat("A2").get_status().name)
print(svc.hold_seats("S1", ["A2", "Z9"], "u2"))
upi = UpiPayment(500)
print(svc.confirm_booking("B1", upi))
print(b1.get_status().name, upi.get_balance())
card = CardPayment(1000)
print(svc.confirm_booking("B1", card))
print(show.get_show_seat("P1").get_status().name)
print(svc.confirm_booking("B1", card))
b2 = svc.hold_seats("S1", ["A2"], "u2")
upi2 = UpiPayment(300)
print(svc.confirm_booking(b2.get_booking_id(), upi2))
print(upi2.get_balance())
print(svc.cancel_booking("B2"))
print(upi2.get_balance())
print(show.get_show_seat("A2").get_status().name)
print(svc.cancel_booking("B2"))
b3 = svc.hold_seats("S1", ["A2"], "u3")
print(svc.cancel_booking(b3.get_booking_id()))
print(svc.cancel_booking("B9"))