class Product:
    def __init__(self, name, size, extra):
        self.name = name
        self.size = size
        self.extra = extra

class ProductBuilder:
    def __init__(self, name):
        self.name = name
        self.size = 0
        self.extra = None

    def set_size(self, size):
        self.size = size
        return self
    
    def set_extra(self, extra):
        self.extra = extra
        return self
    
    def build(self):
        return Product(self.name, self.size, self.extra)
    
    
p = ProductBuilder("Pizza").set_size(10).set_extra("cheese").build()
print(p.name, p.size, p.extra)