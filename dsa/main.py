import multiprocessing
import time

def sumOfSquare(nums):
    time.sleep(5)
    total = sum(n * n for n in nums)
    print(f"The total is {total}")
    
if __name__ == "__main__":
    nums = list(range(10_000_000))
    m1 = multiprocessing.Process(target=sumOfSquare, args = (nums,))
    m2 = multiprocessing.Process(target=sumOfSquare, args = (nums,))
   
    m1.start()
    m2.start()
   
    print("Completed")
   
    m1.join()
    m2.join()
    
    