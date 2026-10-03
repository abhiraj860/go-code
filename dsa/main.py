class ListNode:
    def __init__(self, val = 0, next = None):
        self.val = val
        self.next = next

l1 = ListNode(1)
l1.next = ListNode(4)
l1.next.next = ListNode(6)


l2 = ListNode(2)
l2.next = ListNode(3)

def printList(head):
    curr = head
    while curr:
        print(curr.val, end=" ")
        curr = curr.next
    print()

def mergelist(l1: ListNode, l2: ListNode):        
    if l1 is None:
        return l2
    if l2 is None:
        return l1
    curr = None
    if l1.val < l2.val:
        curr = l1
        l1 = l1.next
    else:
        curr = l2
        l2 = l2.next
    head = curr
    while l1 and l2:
        if l1.val < l2.val:
            curr.next = l1
            l1 = l1.next
        else:
            curr.next = l2
            l2 = l2.next
        curr = curr.next
    curr.next = l1 or l2
    return head

printList(l1)
printList(l2)
head = mergelist(l1, l2)
printList(head)

