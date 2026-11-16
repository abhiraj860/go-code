from abc import ABC, abstractmethod

class Command(ABC):
    @abstractmethod
    def execute(self): ...
    
    @abstractmethod
    def undo(self): ...
    
class TextEditor:
    def __init__(self):
        self.text = ""
        
class AppendCommand(Command):
    def __init__(self, editor, s):
        self.editor = editor
        self.s = s

    def execute(self):
        self.editor.text += self.s
        
    def undo(self):
        self.editor.text = self.editor.text[:-len(self.s)]

class Invoker:
    def __init__(self):
        self.history = []
        
    def run(self, cmd):
        cmd.execute()
        self.history.append(cmd)
        
    def undo(self):
        if self.history:
            self.history.pop().undo()
            
ed = TextEditor()
inv = Invoker()
inv.run(AppendCommand(ed, "hello"))
inv.run(AppendCommand(ed, " world"))
print(ed.text)
inv.undo()
print(ed.text)
    