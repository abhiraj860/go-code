# PROTOTYPE: make new objects by cloning an existing one.
import copy

# 1. Class with a clone() method
class Product:
    def __init__(self, name, tags):
        self.name = name
        self.tags = tags

    # 2. deepcopy so lists/dicts are copied too
    def clone(self):
        return copy.deepcopy(self)


if __name__ == "__main__":
    original = Product("basic", ["x"])
    p = original.clone()
    p.tags.append("y")
    print(original.tags, p.tags)   # ['x'] ['x', 'y']
