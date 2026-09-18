from abc import ABC, abstractmethod

class NotificationChannel(ABC):
    @abstractmethod
    def send_message(self, email, message):
        pass

class EmailChannel(NotificationChannel):
    def send_message(self, email, message):
        return f"Email sent to {email}: {message}"
        
class SMSChannel(NotificationChannel):
    def send_message(self, email, message ):
        return f"SMS sent to {email}: {message}"

class BaseAlert:
    def __init__(self, email, message):
        self.email = email
        self.message = message
    
    def dispatch(self, channel: NotificationChannel):
        return channel.send_message(self.email, self.message)

class ManagerAlert(BaseAlert):
    def __init__(self, email, message, department):
        super().__init__(email, message)
        self.department = department
    
    def dispatch(self, channel: NotificationChannel):
        text = super().dispatch(channel)
        return f"{text}, Dept: {self.department}"

def is_valid_channel(obj: object) -> bool:
    return isinstance(obj, NotificationChannel)

class PayloadAuditor:
    def inspect_payload(self, obj):
       public_attributes = []
       dic = {}
       dic["priority"] = getattr(obj, "priority", "NORMAL")
       for attr in dir(obj):
           if not attr.startswith('__') and not callable(getattr(obj, attr)):
                public_attributes.append(attr)
       dic["public_attributes"] = public_attributes    
       return dic        

# --- INTERVIEW TEST SUITE ---
if __name__ == "__main__":
    # Test 1: Open-Closed Principle & Method Extension
    email = EmailChannel()
    sms = SMSChannel()

    base_alert = BaseAlert("alice@company.com", "Server down")
    mgr_alert = ManagerAlert("bob@company.com", "Budget exceeded", "Finance")

    print("--- Testing Alerts & Method Extension ---")
    print(base_alert.dispatch(email))
    print(mgr_alert.dispatch(sms))

    # Test 2: Dynamic Object Inspection
    class EventPayload:
        def __init__(self, event_id: int, message: str):
            self.event_id = event_id
            self.message = message

    class PriorityPayload:
        def __init__(self, event_id: int, priority: str):
            self.event_id = event_id
            self.priority = priority

    auditor = PayloadAuditor()
    print("\n--- Testing Object Inspection ---")
    print(auditor.inspect_payload(EventPayload(101, "System Reboot")))
    print(auditor.inspect_payload(PriorityPayload(202, "CRITICAL")))

    # Test 3: Type Hierarchy Validation
    print("\n--- Testing Type Hierarchy Verification ---")
    print(f"EmailChannel is valid: {is_valid_channel(email)}")
    print(f"SMSChannel is valid: {is_valid_channel(sms)}")
    print(f"Raw String is valid: {is_valid_channel('Not a channel')}")