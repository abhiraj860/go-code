# dic = {
#     344: "Harry",
#     56: "Shubham",
#     678: "zakir",
#     567: "Neha",
# }

# print(dic[344])
# print(dic.get(34))
# print(dic.keys())
# print(dic.values())

# for key in dic.keys():
#     print(f"The value corresponding to the key {key} is {dic[key]}")
    
    
# print(dic.items())
# for key, val in dic.items():
#     print(key)

ep = {122: 45, 344:56, 122:90, 5566: 89}
ep2 = {122:56, 5566:900, "hhhe": "opop"}

ep.update(ep2)
print(ep)
ep.popitem()
del ep[122]
print(ep)