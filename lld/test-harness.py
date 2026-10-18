import sys
import traceback

try:
    from main import (
        MarketDataFeed, TradingBot, 
        DipBuyerStrategy, MomentumStrategy
    )
except ImportError as e:
    print(f"FAIL: Could not import required classes: {e}")
    sys.exit(1)

def setup_function():
    MarketDataFeed.reset_instance()

def test_singleton_property():
    setup_function()
    feed1 = MarketDataFeed()
    feed2 = MarketDataFeed()
    assert feed1 is feed2, "MarketDataFeed must be a Singleton"

def test_strategy_logic():
    dip = DipBuyerStrategy(buy_threshold=50.0)
    assert dip.analyze(49.9) == "BUY"
    assert dip.analyze(50.0) == "HOLD"
    
    mom = MomentumStrategy(sell_threshold=100.0)
    assert mom.analyze(100.1) == "SELL"
    assert mom.analyze(100.0) == "HOLD"

def test_observer_subscription_management():
    setup_function()
    feed = MarketDataFeed()
    bot = TradingBot("TestBot", DipBuyerStrategy(50.0))
    
    feed.subscribe(bot, "AAPL")
    
    try:
        feed.subscribe(bot, "AAPL")
        assert False, "Should raise ValueError on duplicate subscription to same symbol"
    except ValueError:
        pass
        
    feed.unsubscribe(bot, "AAPL")
    
    try:
        feed.unsubscribe(bot, "AAPL")
        assert False, "Should raise KeyError when unsubscribing from symbol without subscription"
    except KeyError:
        pass

def test_end_to_end_integration():
    setup_function()
    feed = MarketDataFeed()
    
    bot1 = TradingBot("Alice", DipBuyerStrategy(50.0))
    bot2 = TradingBot("Bob", MomentumStrategy(100.0))
    
    feed.subscribe(bot1, "AAPL")
    feed.subscribe(bot2, "AAPL")
    feed.subscribe(bot1, "TSLA")
    
    # Tick 1: AAPL at 40.0 (Alice buys, Bob holds)
    feed.publish_price("AAPL", 40.0)
    
    # Tick 2: TSLA at 30.0 (Alice buys, Bob not subscribed)
    feed.publish_price("TSLA", 30.0)
    
    # Tick 3: AAPL at 105.0 (Alice holds, Bob sells)
    feed.publish_price("AAPL", 105.0)
    
    # Assert Alice's history
    expected_alice = [
        ("AAPL", "BUY", 40.0),
        ("TSLA", "BUY", 30.0),
        ("AAPL", "HOLD", 105.0)
    ]
    assert bot1.trade_history == expected_alice, f"Alice history mismatch: {bot1.trade_history}"
    
    # Assert Bob's history
    expected_bob = [
        ("AAPL", "HOLD", 40.0),
        ("AAPL", "SELL", 105.0)
    ]
    assert bot2.trade_history == expected_bob, f"Bob history mismatch: {bot2.trade_history}"

def test_dynamic_strategy_swap():
    setup_function()
    feed = MarketDataFeed()
    bot = TradingBot("Charlie", DipBuyerStrategy(50.0))
    
    feed.subscribe(bot, "GOOG")
    feed.publish_price("GOOG", 40.0) # BUY
    
    bot.set_strategy(MomentumStrategy(100.0))
    feed.publish_price("GOOG", 150.0) # SELL
    
    expected = [("GOOG", "BUY", 40.0), ("GOOG", "SELL", 150.0)]
    assert bot.trade_history == expected, f"Dynamic swap mismatch: {bot.trade_history}"


if __name__ == "__main__":
    tests = [
        test_singleton_property,
        test_strategy_logic,
        test_observer_subscription_management,
        test_end_to_end_integration,
        test_dynamic_strategy_swap
    ]
    
    passed = 0
    for test in tests:
        try:
            test()
            print(f"PASS: {test.__name__}")
            passed += 1
        except Exception as e:
            print(f"FAIL: {test.__name__}")
            print(f"      {type(e).__name__}: {e}")
            traceback.print_exc(limit=-1, file=sys.stdout)
            
    print(f"\nSummary: {passed}/{len(tests)} passed.")
    if passed < len(tests):
        sys.exit(1)
    else:
        sys.exit(0)