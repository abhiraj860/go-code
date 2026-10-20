from abc import ABC, abstractmethod
import copy

# --- PROTOTYPE & DECORATOR BASE ---
class UIComponent(ABC):
    @abstractmethod
    def render(self) -> str:
        pass
        
    @abstractmethod
    def clone(self) -> 'UIComponent':
        pass

# --- CONCRETE PROTOTYPES ---
class ButtonComponent(UIComponent):
    def __init__(self, label: str, metadata: dict):
        # Store label and metadata
        self.label = label
        self.metadata = metadata
        
    def render(self) -> str:
        return f"[Button: {self.label}]"
        
    def clone(self) -> 'ButtonComponent':
        cloned = copy.deepcopy(self)
        return cloned

# --- DECORATORS ---
class ComponentDecorator(UIComponent):
    def __init__(self, wrapped: UIComponent):
        self.wrapped: UIComponent = wrapped

class PaddingDecorator(ComponentDecorator):
    def render(self) -> str:
        return f"padding({self.wrapped.render()})"
        
    def clone(self) -> 'PaddingDecorator':
        cloned = copy.deepcopy(self)
        return cloned

class ShadowDecorator(ComponentDecorator):
    def render(self) -> str:
        return f"shadow({self.wrapped.render()})"
        
    def clone(self) -> 'ShadowDecorator':
        cloned = copy.deepcopy(self)
        return cloned

# --- SINGLETON REGISTRY ---
class ComponentRegistry:
    _instance = None
    _initialized = False    
    
    def __new__(cls):
        if cls._instance is None:
            cls._instance =  super().__new__(cls)
            return cls._instance
        return cls._instance
        
    def __init__(self):
        if ComponentRegistry._initialized:
            return
        self.diction = {}
        ComponentRegistry._initialized = True
        return 

    def register(self, key: str, component: UIComponent) -> None:
        self.diction[key] = component
        return
        
    def create_instance(self, key: str) -> UIComponent:
        if key in self.diction:
            cloned = self.diction[key].clone()
            return cloned
        raise KeyError("The key does not exist")

    @classmethod
    def reset_instance(cls) -> None:
        ComponentRegistry._instance = None
        ComponentRegistry._initialized = False