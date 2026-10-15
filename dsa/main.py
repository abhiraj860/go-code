def min_window(s1, s2):
    n = len(s1)
    m = len(s2)
    dp = [-1] * m
    min_length = float("inf")
    string = ""
    for i in range(n):
         for j in range(m - 1, -1, -1):
             if s1[i] == s2[j]:
                 if j == 0:
                     dp[j] = i
                 else:
                     dp[j] = dp[j - 1]
         if dp[m - 1] != -1:
            start = dp[m - 1]
            if i - start + 1 < min_length:
                min_length = i - start + 1
                string = s1[start: i + 1]
                
    return string   
    
print(min_window("hellointerview", "her"))
print(min_window("codingisfun", "xyz"))