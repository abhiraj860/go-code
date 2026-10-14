from abc import ABC, abstractmethod

class State(ABC):
    @abstractmethod
    def play(self) -> str:
        pass
        
    @abstractmethod
    def pause(self) -> str:
        pass
        
    @abstractmethod
    def stop(self) -> str:
        pass

class MediaPlayer:
    def __init__(self):
        self.state = StoppedState(self)
        
    def set_state(self, state: State) -> None:
        self.state = state
              
    def play(self) -> str:
        return self.state.play()
        
    def pause(self) -> str:
        return self.state.pause()
        
    def stop(self) -> str:
        return self.state.stop()

class StoppedState(State):
    def __init__(self, player: MediaPlayer):
        self.player = player    
        
    def play(self) -> str:
        self.player.set_state(PlayingState(self.player))
        return "Starting playback"
    
    def pause(self) -> str: 
        return "Can't pause, player is stopped"
    
    def stop(self) -> str:
        return "Already stopped"

class PlayingState(State):
    def __init__(self, player: MediaPlayer):
        self.player = player
        
    def play(self) -> str: 
        return "Already playing"
    
    def pause(self) -> str: 
        self.player.set_state(PausedState(self.player))
        return "Playback paused"
    
    def stop(self) -> str: 
        self.player.set_state(StoppedState(self.player))
        return "Playback stopped"

class PausedState(State):
    def __init__(self, player: MediaPlayer):
        self.player = player
        
    def play(self) -> str: 
        self.player.set_state(PlayingState(self.player))
        return "Resuming playback"
    
    def pause(self) -> str:
        return "Already paused"
    
    def stop(self) -> str:
        self.player.set_state(StoppedState(self.player))
        return "Playback stopped"
    