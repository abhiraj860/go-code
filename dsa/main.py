def min_window(s1, s2):
    n1 = len(s1)
    n2 = len(s2)
    result = []
    def helper(i, j):
        if i == n1:
            return
        if j == n2:
            return "".join(result)
        if s1[i] ==s2[j]:
            result.append(s1[i])
            helper(i + 1, j + 1)
        result.pop()
        helper(i + 1, j)
    helper(0, 0)
    
    
s1 = "hello"
s2 = "he"
print(min_window(s1, s2))