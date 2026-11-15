# STATE: behaviour depends on current state; each state picks the next one.
from abc import ABC, abstractmethod

# 1. State interface: one method per action
class XState(ABC):
    @abstractmethod
    def action1(self, ctx): ...
    @abstractmethod
    def action2(self, ctx): ...

# 2. Concrete states: one per status
class StateA(XState):
    def action1(self, ctx):
        # do work
        ctx.state = StateB()          # legal: move to next state
    def action2(self, ctx):
        raise Exception("Illegal in StateA")

class StateB(XState):
    def action1(self, ctx):
        raise Exception("Illegal in StateB")
    def action2(self, ctx):
        # do work
        ctx.state = StateA()

# 3. Context: just forwards to the current state
class X:
    def __init__(self):
        self.state = StateA()         # initial state
    def action1(self):
        self.state.action1(self)
    def action2(self):
        self.state.action2(self)


if __name__ == "__main__":
    x = X()
    x.action1()   # A -> B
    x.action2()   # B -> A
