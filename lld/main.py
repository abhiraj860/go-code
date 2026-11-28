# P03 - Splitwise - Strategy (split type)
from abc import ABC, abstractmethod
from collections import defaultdict

class SplitStrategy(ABC):
    @abstractmethod
    def split(self):...
    
    @abstractmethod
    def get_user_ids(self):...
    
class EqualSplit(SplitStrategy):
    def __init__(self, user_ids):
        self.user_ids:list[str] = user_ids
         
    def split(self, amount):
        n = len(self.user_ids)
        equal_amount = amount // n
        left_amount = amount % n
        split_amount = {}
        for users in self.user_ids:
            split_amount[users] = equal_amount
        split_amount[self.user_ids[0]] += left_amount
        return split_amount
        
    def get_user_ids(self):
        return self.user_ids
    
class ExactSplit(SplitStrategy):
    def __init__(self, user_amount):
        self.user_amount:dict[str, int] = user_amount
    
    def split(self, amount):
        if sum(list(self.user_amount.values())) != amount:
            return None
        return self.user_amount
    
    def get_user_ids(self):
        return list(self.user_amount.keys())
    
class PercentSplit(SplitStrategy):
    def __init__(self, user_percent):
        self.user_percent:dict[str, int] = user_percent
    
    def split(self, amount):
        if sum(list(self.user_percent.values())) != 100:
            return None
        total = 0
        for k, v in self.user_percent.items():
            curr_val = (amount * v) // 100
            self.user_percent[k] =  curr_val
            total += curr_val
        remaining = amount - total
        keys = self.get_user_ids()
        i = 0
        while remaining > 0:
            i = i % len(keys)
            self.user_percent[keys[i]] += 1
            remaining -= 1
            i += 1
        return self.user_percent
        
   
    def get_user_ids(self):
        return list(self.user_percent.keys())

class Expense:
    def __init__(self, split_strategy):
        self.shares = split_strategy
        
    def get_shares(self) -> dict | None:
        return self.shares
    
class Group:
    def __init__(self, group_id, member_ids):
        self.group_id = group_id
        self.member_ids = member_ids
        self.expense:dict[str, list[Expense]] = defaultdict(list)
    
    def _check_members(self, ids):
        for id in ids:
            if id not in self.member_ids:
                return False
        return True
    
        
    def add_expense(self, payer_id, amount, strategy: SplitStrategy) -> Expense | None:
        involved = [payer_id] + strategy.get_user_ids()
        if not self._check_members(involved):
            return None
        split_strategy = strategy.split(amount)
        if not split_strategy:
            return None
        e1 = Expense(split_strategy)
        if e1 is not None:
            self.expense[payer_id].append(e1)
        return e1

        
    def settle(self, from_id, to_id, amount):
        if from_id == to_id:
            return False
        involved = [from_id, to_id]
        if not self._check_members(involved):
            return False
        if amount <= 0:
            return False
        self.expense[from_id].append(Expense({to_id: amount}))
        return True
        
        
        
    def get_balance_between(self, a_id, b_id):
        involved = [a_id, b_id]
        if not self._check_members(involved):
            return -1
        a_paid = 0 
        for exp in self.expense[a_id]:
            a_paid += exp.get_shares().get(b_id, 0)
        b_paid = 0
        for exp in self.expense[b_id]:
            b_paid += exp.get_shares().get(a_id, 0)
        return a_paid - b_paid
                    
        
    def get_net_balance(self):
        net_balance = defaultdict(int)
        for user, expense in self.expense.items():
            for exp in expense:
                shares = exp.get_shares()
                for us, amt in shares.items():
                   if us != user:
                       net_balance[user] += amt
                       net_balance[us] -= amt
        return dict(net_balance) 
                   
                       
         
        

g = Group("trip", ["A", "B", "C"])
e1 = g.add_expense("A", 100, EqualSplit(["A", "B", "C"]))
print(e1.get_shares())
print(g.get_balance_between("A", "B"))
print(g.add_expense("B", 90, ExactSplit({"A": 50, "C": 30})))
e2 = g.add_expense("B", 90, ExactSplit({"A":60, "C": 30}))
print(e2.get_shares())
print(g.get_balance_between("A", "B"))
print(g.add_expense("C", 50, PercentSplit({"A": 50, "B": 40})))
e3 = g.add_expense("C", 99, PercentSplit({"A":50, "B": 25, "C": 25}))
print(e3.get_shares())
print(g.add_expense("D", 10, EqualSplit(["A", "B"])))
print(g.get_net_balance())
print(g.settle("A", "B", 32))
print(g.get_net_balance())
print(g.get_balance_between("C", "A"))
print(g.settle("A", "A", 5))