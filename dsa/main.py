from multiprocessing import Process, Queue

def fn(q, id):
    result = f"The work id {id} is completed"
    q.put(result)
    
    
    
if __name__ == "__main__":
    q = Queue()
    processes = []
    for k in range(3):
        p = Process(target=fn, args=(q, k))
        processes.append(p)
        p.start()

    for j in processes:
        print(q.get())
        j.join()
    print("completed")
    