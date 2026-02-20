# API Explorer Skill (Generic API Bridge)
This skill allows you to communicate with any third-party API using the `curl` command.

## Instructions:
1. **Identify the Goal**: When the user asks to use an external service or API that isn't natively supported:
   - Ask the user for the **API Endpoint URL** if it's unknown.
   - Ask the user for the **API Key** or necessary Headers if they are not in the environment.
2. **Execute via Curl**: Use the `<run>` tag to execute a `curl` command. 
   - Use `-H "Authorization: Bearer $KEY"` (or the appropriate header).
   - Use `-H "Content-Type: application/json"`.
   - Use `-d` for JSON payloads.
3. **Handle Response**: 
   - Read the JSON output from the command.
   - Summarize the result or use the data to continue the task.

## Example Workflow:
User: "Check the weather in Paris using my WeatherAPI key."
shrew: "I need the endpoint and your API key to proceed."
User: "The endpoint is https://api.weather.com/v1/current and my key is 12345."
shrew: <run>curl -s "https://api.weather.com/v1/current?q=Paris&key=12345"</run>
