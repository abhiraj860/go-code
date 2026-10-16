def brute_force_search(text: str, pattern: str) -> list[int]:
    matches = []
    for i in range(len(text) - len(pattern) + 1):
        if text[i: i + len(pattern)] == pattern:
            matches.append(i)
    return matches