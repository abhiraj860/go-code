from config import LLM_PROVIDER
from config import OLLAMA_MODEL


# ── LLM initialisation ────────────────────────────────────────────────────────
# Default is Ollama (local, no API key needed). To switch providers, set
# LLM_PROVIDER in .env to "openai" or "gemini" and supply the matching key.
# temperature=0 keeps the agent deterministic across all providers.

# if LLM_PROVIDER == "openai":
#     from langchain_openai import ChatOpenAI
#     llm = ChatOpenAI(model=LLM_MODEL, api_key=OPENAI_API_KEY, temperature=0)

# elif LLM_PROVIDER == "gemini":
#     from langchain_google_genai import ChatGoogleGenerativeAI
#     llm = ChatGoogleGenerativeAI(model=LLM_MODEL, google_api_key=GOOGLE_API_KEY, temperature=0)

# else:  # ollama — default, runs locally on http://localhost:11434
from langchain_ollama import ChatOllama

llm = ChatOllama(model=OLLAMA_MODEL, temperature=0)

# ── Smoke test ────────────────────────────────────────────────────────────────
# Confirms the chosen provider is reachable and the model is loaded.
# Gemini returns content as a list of parts; other providers return a plain
# string. Normalise to a string before printing.
test = llm.invoke("Reply with exactly three words: agent loop ready")
reply = (
    test.content
    if isinstance(test.content, str)
    else "".join(
        p.get("text", str(p)) if isinstance(p, dict) else str(p)
        for p in test.content
    )
)
print(f"LLM check    : {reply.strip()}")
print(f"Provider     : {LLM_PROVIDER}")