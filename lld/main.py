from abc import ABC, abstractmethod

class Notification(ABC):
    @abstractmethod
    def send(self, message: str, recipient: str) -> str:
        pass

class EmailNotification(Notification):
    def send(self, message: str, recipient: str) -> str:
        return f"Email sent to {recipient}: {message}"

class SMSNotification(Notification):
    def send(self, message: str, recipient: str) -> str:
        return f"SMS sent to {recipient}: {message}"

class NotificationService(ABC):
    @abstractmethod
    def create_notification(self) -> Notification:
        pass
    
    def send_notification(self, message: str, recipient: str) -> str:
        if not message or not message.strip():
            raise ValueError("Message cannot be empty") 
        if not recipient or not recipient.strip():
            raise ValueError("Recipiect cannot be empty")
        notification = self.create_notification()
        return notification.send(message, recipient)
            

class EmailNotificationService(NotificationService):
    def create_notification(self) -> Notification:
        return EmailNotification()

class SMSNotificationService(NotificationService):
    def create_notification(self) -> Notification:
        return SMSNotification()