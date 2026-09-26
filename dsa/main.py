s1 = "hellointerview"
s2 = "her"

def min_window(s1: str, s2: str) -> str:
    
    def bottomUp():
        n1 = len(s1)
        n2 = len(s2)
        dp = [
            [[(float("inf"), "") for _ in range(n1 + 1)] for _ in range(n2 + 1)] for _ in range(n1 + 1)
        ]
        for i in range(n1, -1, -1):
            for j in range(n2, -1, -1):
                for start in range(-1, n1):
                    start_idx = start + 1
                    if j == n2:
                        dp[i][j][start_idx] = (i - start, s1[start: i])
                        continue
                    if i == n1:
                        dp[i][j][start_idx] = (float("inf"), "")
                        continue
                    best_len, best_str = float("inf"), ""
                    if s1[i] == s2[j]:
                        best_len, best_str = dp[i + 1][j + 1][(i if start == -1 else start) + 1]
                    curr_len, curr_str = dp[i + 1][j][start_idx]
                    if curr_len < best_len:
                        best_len = curr_len 
                        best_str = curr_str
                    dp[i][j][start_idx] = (best_len, best_str) 
        
        
        (_, text) = dp[0][0][0]
        return text
    return bottomUp()
print(min_window(s1, s2))