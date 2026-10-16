from typing import List

class Projector:
    def turn_on(self) -> str:
        return "Projector powered on"
        
    def turn_off(self) -> str:
        return "Projector powered off"
        
    def set_input(self, source: str) -> str:
        return f"Projector input set to {source}"

class SoundSystem:
    def turn_on(self) -> str:
        return "Sound system powered on"
        
    def turn_off(self) -> str:
        return "Sound system powered off"
        
    def set_volume(self, level: int) -> str:
        return f"Sound system volume set to {level}"

class StreamingDevice:
    def turn_on(self) -> str:
        return "Streaming device powered on"
        
    def turn_off(self) -> str:
        return "Streaming device powered off"
        
    def play_movie(self, title: str) -> str:
        return f"Playing '{title}'"

class HomeTheaterFacade:
    def __init__(self, projector: Projector, sound_system: SoundSystem, streaming_device: StreamingDevice):
        self.projector = projector
        self.sound_system = sound_system
        self.streaming_device = streaming_device

    def watch_movie(self, movie_title: str, volume: int) -> List[str]:
        if volume < 0 or volume > 100:
            raise ValueError("Volume outside the range")
        
        steps = []
        steps.append(self.projector.turn_on())
        steps.append(self.projector.set_input("HDMI 1"))
        steps.append(self.sound_system.turn_on())
        steps.append(self.sound_system.set_volume(volume))
        steps.append(self.streaming_device.turn_on())
        steps.append(self.streaming_device.play_movie(movie_title))
        return steps

    def end_movie(self) -> List[str]:
       steps = []
       steps.append(self.streaming_device.turn_off())
       steps.append(self.sound_system.turn_off())
       steps.append(self.projector.turn_off())
       return steps