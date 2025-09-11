#!/bin/bash

echo "Testing Maestro with your APIs"
echo "=============================="

# 1. Start FastAPI mock
echo "Starting FastAPI mock on port 8000..."
python3 -m http.server 8000 &
FASTAPI_PID=$!

# 2. Start Next.js mock
echo "Starting Next.js mock on port 3000..."
python3 -m http.server 3000 &
NEXTJS_PID=$!

sleep 2

# 3. Run workflow with mock responses
echo "\nExecuting user onboarding workflow..."
./bin/maestro execute examples/workflows/user_onboarding.yaml \
  --input '{"email":"test@example.com","name":"Test User","plan":"premium","password":"secret123"}' \
  --debug

# Cleanup
kill $FASTAPI_PID $NEXTJS_PID 2>/dev/null

echo "\nTest completed!"