class ListNode:
    def __init__(self, val = 0, next = None):
        self.val = val
        self.next = None

first = ListNode(1)
second = ListNode(2)
third = ListNode(3)
fourth = ListNode(4)
fifth = ListNode(5)

head = first
first.next = second
second.next = third
third.next = fourth
fourth.next = fifth

def printNode(head):
    curr = head
    while curr:
        print(curr.val)
        curr = curr.next
    return

def deleteNode(head: ListNode, value):
    if head.val == value:
        return head.next
    prev, curr = None, head
    while curr.val != value:
        prev = curr
        curr = curr.next
    if curr == None:
        return head
    prev.next = curr.next
    return head
printNode(head)
print("After Deleting")
p = deleteNode(head, 5)
printNode(p)
        