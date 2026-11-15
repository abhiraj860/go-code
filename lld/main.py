from abc import ABC, abstractmethod

class State(ABC):
    @abstractmethod
    def action1(self): ...
    
    @abstractmethod
    def action2(self): ...
    
class StateA(State):
    def action1(self, ctx):
        print("State A transitioned to State B")
        ctx.state = StateB()
        
    def action2(self, ctx):
        raise ValueError("Cannot perform action A")
    
class StateB(State):
    def action1(self, ctx):
        raise ValueError("Connot perform action B")
    def action2(self, ctx):
        print("State B transitioned to State A")
        ctx.state = StateA()
        
        
class Object:
    def __init__(self):
        self.state = StateA()
        
    def action1(self):
        self.state.action1(self)
    
    def action2(self):
        self.state.action2(self)
        
x = Object()
x.action1()
x.action2()
x.action1()