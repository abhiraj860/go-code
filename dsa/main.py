class IntGraphNode:
    def __init__(self, value = 0, neighbours = None):
        self.value = value
        self.neighbours = neighbours if neighbours is not None else []
        
n1 = IntGraphNode(1)
n2 = IntGraphNode(2)
n3 = IntGraphNode(3)
n4 = IntGraphNode(4)

n1.neighbours = [n2, n4]
n2.neighbours = [n1, n3]
n3.neighbours = [n2, n4]
n4.neighbours = [n1, n3]

node = n1

adj_list = {}

def dfs(node: IntGraphNode):
   if node is None:
       return
   if node.value in adj_list:
       return
   adj_list[node.value] = []
   for nbrs in node.neighbours:
        adj_list[node.value].append(nbrs.value)
        dfs(nbrs)
dfs(node)

print(adj_list)

