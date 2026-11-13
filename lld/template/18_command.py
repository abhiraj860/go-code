# COMMAND: wrap an action as an object -> run later, queue it, undo it.
from abc import ABC, abstractmethod

# 1. Command interface
class Command(ABC):
    @abstractmethod
    def execute(self): ...

    @abstractmethod
    def undo(self): ...

# 2. Receiver: the thing that actually does the work
class TextEditor:
    def __init__(self):
        self.text = ""

# 3. Concrete command: remembers what it needs to undo
class AppendCommand(Command):
    def __init__(self, editor, s):
        self.editor = editor
        self.s = s

    def execute(self):
        self.editor.text += self.s

    def undo(self):
        self.editor.text = self.editor.text[: -len(self.s)]

# 4. Invoker: runs commands, keeps history for undo
class Invoker:
    def __init__(self):
        self.history = []

    def run(self, cmd):
        cmd.execute()
        self.history.append(cmd)

    def undo(self):
        if self.history:
            self.history.pop().undo()


if __name__ == "__main__":
    ed = TextEditor()
    inv = Invoker()
    inv.run(AppendCommand(ed, "hello"))
    inv.run(AppendCommand(ed, " world"))
    print(ed.text)   # hello world
    inv.undo()
    print(ed.text)   # hello
