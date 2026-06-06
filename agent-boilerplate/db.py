import oracledb
from config import (
    ORACLE_USER,
    ORACLE_PASSWORD,
    ORACLE_DSN,
)


# ── Connection pool ───────────────────────────────────────────────────────────
# A pool rather than a single connection means later cells can acquire and
# release connections independently without holding one open the whole time.
# OracleVS accepts a ConnectionPool directly — the pool is passed in at
# vector store creation time rather than acquired per operation.
pool = oracledb.create_pool(
    user=ORACLE_USER,
    password=ORACLE_PASSWORD,
    dsn=ORACLE_DSN,
    min=2,
    max=10,
    increment=1,
)

# ── Default tablespace ────────────────────────────────────────────────────────
# The SYSTEM tablespace uses manual segment space management, which blocks
# JSON and VECTOR column types (ORA-43853). Switching to USERS — which uses
# automatic segment space management — allows OracleVS to create its table.
with pool.acquire() as conn:
    with conn.cursor() as cur:
        cur.execute(f"ALTER USER {ORACLE_USER} DEFAULT TABLESPACE USERS")
    conn.commit()
print(f"Default tablespace for '{ORACLE_USER}' set to USERS.")

# ── Smoke test ────────────────────────────────────────────────────────────────
try:
    with pool.acquire() as conn:
        with conn.cursor() as cur:
            cur.execute("SELECT 'Oracle connection OK' FROM dual")
            row = cur.fetchone()
    print(f"Connected to Oracle AI Database 26ai: {row[0]}")
    print("Connection pool ready (min=2, max=10).")
except oracledb.DatabaseError as e:
    error, = e.args
    if error.code == 1017:
        print("ERROR ORA-01017: invalid username/password.")
        print("Check that ORACLE_PASSWORD in alf-05-config matches the")
        print("ORACLE_PWD value used when starting the container.")
    elif error.code == 12541:
        print("ERROR ORA-12541: no listener. Is the container running?")
    else:
        raise