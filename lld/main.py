from abc import ABC, abstractmethod

# --- DECORATOR ---
class Alert(ABC):
    @abstractmethod
    def trigger(self) -> str:
        pass

class BaseAlert(Alert):
    def trigger(self) -> str:
        return "Siren"

class AlertDecorator(Alert):
    def __init__(self, wrapped: Alert):
        self.wrapped = wrapped

class PhoneAlertDecorator(AlertDecorator):
    def trigger(self) -> str:
        return self.wrapped.trigger() + ", Phone"

class PoliceAlertDecorator(AlertDecorator):
    def trigger(self) -> str:
        return self.wrapped.trigger() + ", Police"

# --- STATE MACHINE ---
class SystemState(ABC):
    @abstractmethod
    def arm(self) -> str:
        pass
    @abstractmethod
    def disarm(self) -> str:
        pass
    @abstractmethod
    def trip_sensor(self) -> str:
        pass

class SecuritySystem:
    def __init__(self, alert: Alert):
        self.alert = alert
        self.state = DisarmedState(self)
        
    def set_state(self, state: SystemState) -> None:
        self.state = state

    def arm(self) -> str:
        return self.state.arm()
        
    def disarm(self) -> str:
        return self.state.disarm()
        
    def trip_sensor(self) -> str:
        return self.state.trip_sensor()

class DisarmedState(SystemState):
    def __init__(self, system: SecuritySystem): 
        self.system = system
        
    def arm(self) -> str: 
        self.system.set_state(ArmedState(self.system))
        return "System armed"

    def disarm(self) -> str: 
        return "Already disarmed"
    
    def trip_sensor(self) -> str: 
        return "Ignored"

class ArmedState(SystemState):
    def __init__(self, system: SecuritySystem):
        self.system = system

    def arm(self) -> str:
        return "Already armed"
    
    def disarm(self) -> str:
        self.system.set_state(DisarmedState(self.system))
        return "System disarmed"
        
    def trip_sensor(self) -> str: 
        self.system.set_state(TriggeredState(self.system))
        return self.system.alert.trigger()
        
class TriggeredState(SystemState):
    def __init__(self, system: SecuritySystem): 
        self.system = system
        
    def arm(self) -> str:
        return "Cannot arm, system triggered"
    
    def disarm(self) -> str: 
        self.system.set_state(DisarmedState(self.system))
        return "System disarmed"
    
    def trip_sensor(self) -> str:
        return "Already triggered"

# --- BUILDER ---
class SecuritySystemBuilder:
    def __init__(self):
        self.alert = None
        
    def set_base_alert(self) -> 'SecuritySystemBuilder':
        self.alert = BaseAlert()
        return self
        
    def add_phone_alert(self) -> 'SecuritySystemBuilder':
        if self.alert is None:
            raise ValueError("Please add base alert")
        self.alert = PhoneAlertDecorator(self.alert)
        return self
        
    def add_police_alert(self) -> 'SecuritySystemBuilder':
        if self.alert is None:
            raise ValueError("Please add base alert")
        self.alert = PoliceAlertDecorator(self.alert)
        return self
        
    def build(self) -> SecuritySystem:
        if self.alert is None:
            raise ValueError("Cannot build system without a base alert")
        return SecuritySystem(self.alert)