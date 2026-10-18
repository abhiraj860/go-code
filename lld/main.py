from abc import ABC, abstractmethod
from typing import List, Tuple, Dict

# --- STRATEGY ---
class TradingStrategy(ABC):
    @abstractmethod
    def analyze(self, price: float) -> str:
        """Returns 'BUY', 'SELL', or 'HOLD'."""
        pass

class DipBuyerStrategy(TradingStrategy):
    def __init__(self, buy_threshold: float):
        self.buy_threshold = buy_threshold

    def analyze(self, price: float) -> str:
        if price < self.buy_threshold:
            return "BUY"
        return "HOLD"

class MomentumStrategy(TradingStrategy):
    def __init__(self, sell_threshold: float):
        self.sell_threshold = sell_threshold
        
    def analyze(self, price: float) -> str:
        if price > self.sell_threshold:
            return "SELL"
        return "HOLD"

# --- OBSERVER ---
class Observer(ABC):
    @abstractmethod
    def update(self, symbol: str, price: float) -> None:
        pass

class TradingBot(Observer):
    def __init__(self, name: str, strategy: TradingStrategy):
        self.name = name
        self.strategy = strategy
        # Must store action history as (symbol, action, price)
        self.trade_history: List[Tuple[str, str, float]] = []
        
    def set_strategy(self, strategy: TradingStrategy) -> None:
        self.strategy = strategy

    def update(self, symbol: str, price: float) -> None:
        self.trade_history.append((symbol, self.strategy.analyze(price), price))

# --- SINGLETON (Subject) ---
class MarketDataFeed:
    _instance = None
    _initialised = False
    
    def __new__(cls):
        if cls._instance is None:
            cls._instance = super().__new__(cls)
        return cls._instance

    def __init__(self):
        if MarketDataFeed._initialised:
            return
        self.observer_list:dict[str, list[Observer]] = {}
        MarketDataFeed._initialised = True

    def subscribe(self, bot: Observer, symbol: str) -> None:
        if symbol in self.observer_list and bot in self.observer_list[symbol]:
            raise ValueError("Bot and Symbol already present")
        if symbol in self.observer_list:
            self.observer_list[symbol].append(bot)
            return
        self.observer_list[symbol] = [bot]
        return 
            
        
    def unsubscribe(self, bot: Observer, symbol: str) -> None:
        if symbol not in self.observer_list or bot not in self.observer_list[symbol]:
            raise KeyError("Bot not subscribed to the symbol")
        self.observer_list[symbol].remove(bot)
        return 
        
    def publish_price(self, symbol: str, price: float) -> None:
        for bots in self.observer_list.get(symbol, []):
            bots.update(symbol, price)

    @classmethod
    def reset_instance(cls) -> None:
        MarketDataFeed._instance = None
        MarketDataFeed._initialised = False