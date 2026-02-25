# API Explorer Skill (Generic API Bridge)
This skill allows you to communicate with any third-party API using the `curl` command.

## Instructions:
1. **Identify the Goal**: When the user asks to use an external service or API that isn't natively supported:
   - Ask the user for the **API Endpoint URL** if it's unknown.
   - If the user doesn't know the vault key name, use `<vault_list/>` to see available keys.
   - Ask the user for the **Vault Key Name** (e.g., "WEATHER_API_KEY") once you've identified potential keys.
2. **Execute via Curl**: Use the `<run>` tag with a vault placeholder for the secret.
   - Use `-H "Authorization: Bearer [[vault:KEY_NAME]]"` (or the appropriate header).
   - Use `-H "Content-Type: application/json"`.
   - Use `-d` for JSON payloads.
3. **Handle Response**: 
   - Read the JSON output from the command.
   - Summarize the result or use the data to continue the task.

## Example Workflow:
User: "Check the weather in Paris using my WeatherAPI key stored in the vault as WEATHER_KEY."
shrew: <run>curl -s "https://api.weather.com/v1/current?q=Paris&key=[[vault:WEATHER_KEY]]"</run>
