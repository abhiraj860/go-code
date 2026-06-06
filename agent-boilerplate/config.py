import os 
from dotenv import load_dotenv

load_dotenv(override=True)

# LLM Provider
LLM_PROVIDER = os.getenv("LLM_PROVIDER", "ollama")

# OLLAMA Models
OLLAMA_MODEL = os.getenv("OLLAMA_MODEL", "llama3.2:3b")
OLLAMA_EMBED_MODEL = os.getenv("OLLAMA_EMBED_MODEL", "nomic-embed-text")

# Oracle AI26
ORACLE_USER = os.getenv("ORACLE_USER", "system")
ORACLE_PASSWORD = os.getenv("ORACLE_PASSWORD", "YourPassword123")
ORACLE_DSN = os.getenv("ORACLE_DSN", "localhost:1521/FREE")

print(f"LLM PROVIDER : {LLM_PROVIDER}")

if LLM_PROVIDER == "ollama":
    print(f"Ollama model       : {OLLAMA_MODEL}")
    print(f"Ollama embed model : {OLLAMA_EMBED_MODEL}")
else:
    print("LLM provider not set. Please setup LLM provider")

print(f"Oracle DSN : {ORACLE_DSN}")
print(f"Oracle User : {ORACLE_USER}")


