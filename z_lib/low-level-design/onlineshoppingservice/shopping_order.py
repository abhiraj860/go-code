from __future__ import annotations
from enum import Enum, auto
from abc import ABC, abstractmethod
from cart_and_users import Subject, Customer, Address
from typing import List
from datetime import datetime

class OrderLineItem:
    def __init__(self, productId: str, quantity: int, productName: str, priceAtPurchase: float):
        self.productId = productId
        self.quantity = quantity
        self.productName = productName
        self.priceAtPurchase = priceAtPurchase

class OrderStatus(Enum):
    PLACED = auto()
    SHIPPED = auto()
    PENDING_PAYMENT = auto()
    DELIVERED = auto()
    CANCELLED = auto()
    RETURNED = auto()
    
class OrderState(ABC):
    @abstractmethod
    def shipOrder(self, order: Order):
        pass
    
    @abstractmethod
    def deliverOrder(self, order: Order):
        pass

    @abstractmethod
    def cancelOrder(self, order: Order):
        pass

class PlacedState(OrderState):
    def shipOrder(self, order: Order):
        order.setCurrentState(ShippedState())
        order.status = OrderStatus.SHIPPED
        order.notifyObservers()     
    
    def deliverOrder(self, order: Order):
        raise Exception("Cannot deliver an order that has not been shipped yet")
    
    def cancelOrder(self, order: Order):
        order.setCurrentState(CancelledState())
        order.status = OrderStatus.CANCELLED
        order.notifyObservers()
     
class ShippedState(OrderState):
    def shipOrder(self, order: Order):
        raise Exception("Order already shipped")
    
    def deliverOrder(self, order: Order):
        order.setCurrentState(DeliveredState())
        order.status = OrderStatus.DELIVERED
        order.notifyObservers()
    
    def cancelOrder(self, order: Order):
        raise Exception("Cannot cancel an order that has been already been shipped.")

class DeliveredState(OrderState):
    def shipOrder(self, order: Order):
        raise Exception("Cannot ship an order that is already delivered")
    
    def deliverOrder(self, order: Order):
        raise Exception("Order is already delivered")
    
    def cancelOrder(self, order: Order):
        raise Exception("Cannot cancel an order that has already been delivered")
    
class CancelledState(OrderState):
    def shipOrder(self, order: Order):
        raise Exception("Cannot shipped cancelled order")
    def deliverOrder(self, order: Order):
        raise Exception("Cannot delivered cancelled order")
    def cancelOrder(self, order: Order):
        raise Exception("Order is already cancelled")
    
class Order(Subject):
    def __init__(self, id: str, customer: Customer, items: List[OrderLineItem], totalAmount: float, shippingAddress: Address):
        super().__init__()
        self.id = id
        self.customer = customer
        self.items = items
        self.totalAmount = totalAmount
        self.shippingAddress = shippingAddress
        self.status = OrderStatus.PLACED
        self.currentState = PlacedState()
        self.orderDate = datetime.now()
        self.observers = []
        self.addObserver(customer)

    def addObserver(self, observer) -> None:
        if observer not in self.observers:
            self.observers.append(observer)

    def removeObserver(self, observer) -> None:
        if observer in self.observers:
            self.observers.remove(observer)

    def notifyObservers(self) -> None:
        for observer in self.observers:
            observer.update(self)
    
    def shipOrder(self) -> None:
        self.currentState.shipOrder(self)

    def deliverOrder(self) -> None:
        self.currentState.deliverOrder(self)

    def cancelOrder(self) -> None:
        self.currentState.cancelOrder(self)

    def setCurrentState(self, state: OrderState) -> None:
        self.currentState = state