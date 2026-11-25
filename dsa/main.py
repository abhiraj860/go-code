def encode(arr):
    # write your logic to encode the strings
    string = ""
    for v in arr:
        l = str(len(v)) + "/:"
        string = string + l + v
    return string

def decode(s):
        # write your logic to decode the string
        result = []
        i, n, num = 0, len(s), 0
        while i < n:
            if s[i].isdigit():
                num = num * 10 + int(s[i])
                i += 1
            else:
                result.append(s[i + 2: i + 2 + num])
                i += (num + 2)
                num = 0
        return result


print(decode(encode(["Hello", "World"])))
print(decode(encode(["abc", "!@erwerwrewrwerwerewrwerew"])))