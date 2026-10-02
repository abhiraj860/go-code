def calc(s):
    state = {"}" : "{", ")" : "(", "]": "["}
    stack = []
    for ch in s:
        match ch:
            case "[" | "(" | "{":
                stack.append(ch)
            case "]" | ")" | "}":
                if not stack or state[ch] != stack[-1]:
                    return False 
                stack.pop()
            case _:
                ""
    return len(stack) == 0
                
    
print(calc("(){({}})"))