from multiprocessing import cpu_count, Pool

def calc(num):
    total = sum([n * n for n in range(num)])
    print(f"The total is {total}")
    

if __name__ == "__main__":
    print(f"The total cpu count is {cpu_count()}")
    arr = [1_000_000, 2_000_000, 10_00_000, 500_000, 900_0000]
    with Pool() as pool:
        pool.map(calc, arr)
        

    
    print("Completed")    