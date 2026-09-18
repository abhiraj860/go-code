import re

sample = "Order #A-102 cost $45.99 on 2026-09-18"

numbers = re.findall(r"\d+\.?\d*", sample)
print(numbers)

date = re.search(r"\d{4}-\d{2}-\d{2}", sample)
print(date.group())


