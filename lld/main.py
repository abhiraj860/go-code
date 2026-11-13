import copy 

class Product:
    def __init__(self, name, tags):
        self.name = name
        self.tags = tags

    def clone(self):
        return copy.deepcopy(self)
    
    
    
original = Product("basic", ["X"])
p = original.clone()
p.tags.append("y")
print(original is p)
print(original.tags, p.tags)