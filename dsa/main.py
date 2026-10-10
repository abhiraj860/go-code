def vowelStrings(word, queries):
    n = len(word)
    prefix = [0] * (n + 1)
    vowels = ["a", "e", "i", "o", "u"]
    cnt_till_now = 0
    result = []
    for i, ch in enumerate(word):
        if ch in vowels:
            cnt_till_now += 1
        prefix[i + 1] = cnt_till_now
    print(prefix)
    for v in queries:
        result.append(prefix[v[1] + 1] - prefix[v[0]])
    return result
    

word = "prefixsum" 
queries = [[0, 2], [1, 4], [3, 5]]
print(vowelStrings(word, queries))