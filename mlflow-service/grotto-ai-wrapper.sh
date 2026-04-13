#!/usr/bin/env bash
# grotto-ai-wrapper.sh
#
# Bridges grotto's AI panel (which expects a CLI tool) to the
# MLflow-backed code completion REST service.
#
# Usage:
#   grotto --ai "./grotto-ai-wrapper.sh"
#
# The script reads stdin (what you type in the AI panel),
# sends it to the completion service, and prints the result.
#
# Setup:
#   chmod +x grotto-ai-wrapper.sh
#   grotto --ai "./grotto-ai-wrapper.sh"

SERVICE_URL="${GROTTO_AI_SERVICE:-http://localhost:8000}"
MAX_TOKENS="${GROTTO_AI_MAX_TOKENS:-80}"
TEMPERATURE="${GROTTO_AI_TEMPERATURE:-0.7}"

# Check service is up
if ! curl -sf "${SERVICE_URL}/health" > /dev/null 2>&1; then
    echo "ERROR: Code completion service not reachable at ${SERVICE_URL}"
    echo "Start it with: python serve.py"
    exit 1
fi

# Read prompt from stdin (what the user typed in grotto's AI panel)
PROMPT=$(cat)

if [ -z "$PROMPT" ]; then
    echo "No prompt provided."
    exit 0
fi

# Send to completion service
RESPONSE=$(curl -sf \
    -X POST "${SERVICE_URL}/complete" \
    -H "Content-Type: application/json" \
    -d "{\"prompt\": $(echo "$PROMPT" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))'), \"max_new_tokens\": ${MAX_TOKENS}, \"temperature\": ${TEMPERATURE}}" \
    2>&1)

if [ $? -ne 0 ]; then
    echo "ERROR: Request to completion service failed."
    echo "$RESPONSE"
    exit 1
fi

# Extract and print just the completion text
echo "$RESPONSE" | python3 -c "
import json, sys
data = json.load(sys.stdin)
print(data.get('completion', ''))
"