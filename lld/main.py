from abc import ABC, abstractmethod
import json

class AnalyticsLogger(ABC):
    @abstractmethod
    def log_event(self, event_name: str, metadata: dict) -> None:
        pass

# The third-party Adaptee (Do not modify the internal logic of this class)
class CloudWatchService:
    def __init__(self):
        self.metrics_store = []
        
    def push_metric(self, metric_name: str, dimensions: str) -> None:
        self.metrics_store.append((metric_name, dimensions))

# Your Adapter
class CloudWatchAdapter(AnalyticsLogger):
    def __init__(self, cloud_service: CloudWatchService):
        self.service:CloudWatchService = cloud_service
        
    def log_event(self, event_name: str, metadata: dict) -> None:
        event_name = event_name.upper()
        metadata = json.dumps(metadata)
        self.service.push_metric(event_name, metadata)