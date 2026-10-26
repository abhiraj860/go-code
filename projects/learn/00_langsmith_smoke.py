from dotenv import load_dotenv
load_dotenv()  # finds projects/.env by walking up from this file

from deepagents import create_deep_agent
from langchain.messages import HumanMessage

EXAMPLE_QUERY = "What can you do?"

baseline_agent = create_deep_agent(
    model="openai:gpt-6-luna",
    tools=[],
    system_prompt=(
        "Which model are you?"
    ),
)

result = baseline_agent.invoke(
    {"messages": [HumanMessage(content=EXAMPLE_QUERY)]}
)

print(result["messages"][-1].text)