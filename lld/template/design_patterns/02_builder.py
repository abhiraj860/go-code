# BUILDER: set fields step by step, then build().

# 1. Product: the final object
class Product:
    def __init__(self, name, size, extra):
        self.name = name
        self.size = size
        self.extra = extra

# 2. Builder: each setter returns self (for chaining)
class ProductBuilder:
    def __init__(self, name):
        self.name = name        # required
        self.size = 0           # optional, default
        self.extra = None       # optional, default

    def set_size(self, size):
        self.size = size
        return self

    def set_extra(self, extra):
        self.extra = extra
        return self

    # 3. build(): create the product
    def build(self):
        return Product(self.name, self.size, self.extra)


if __name__ == "__main__":
    p = ProductBuilder("pizza").set_size(10).set_extra("cheese").build()
    print(p.name, p.size, p.extra)
