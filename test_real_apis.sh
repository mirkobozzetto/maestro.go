#!/bin/bash

echo "Test avec tes vraies APIs"
echo "========================"
echo ""
echo "Assure-toi que:"
echo "1. FastAPI tourne sur http://localhost:8000"
echo "2. Next.js tourne sur http://localhost:3000"
echo ""
read -p "Presse Enter quand tes APIs sont prêtes..."

echo "Test 1: Validation du workflow"
./bin/maestro validate examples/workflows/local_apis.yaml

echo ""
echo "Test 2: Exécution avec tes APIs"
./bin/maestro execute examples/workflows/local_apis.yaml \
  --input '{"email":"test@example.com","name":"John Doe"}' \
  --debug

echo ""
echo "Test terminé!"