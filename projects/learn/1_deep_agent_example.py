from dotenv import load_dotenv
load_dotenv()  

from deepagents import create_deep_agent
from langchain.messages import HumanMessage

baseline_agent = create_deep_agent(
    model = "openai:gpt-5.5",
    tools = [],
    system_prompt = (
        "You are a helpful LangChain documentation assistant. "
        "Answer questions about LangChain APIs and patterns."
    )
)

result = baseline_agent.invoke(
    {"messages": [HumanMessage(content="How do I stream intermediate tool results from a subagent?")]}    
)

print(result["messages"][-1].text)