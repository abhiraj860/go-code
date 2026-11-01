from fastapi import FastAPI

app = FastAPI()

@app.get("/users")
async def read_users():
    return ["Rick", "Mortys"]

@app.get("/users")
async def read_users2():
    return ["Bean", "Elfo"]


@app.get("/user/me")
async def read_user_me():
    return {"user_id": "the current user"}

@app.get("/user/{user_id}")
async def read_user(user_id: str):
    return {"user_id": user_id}

