#!/bin/bash
# -------------------------------------------------------------
# GreenTech Hackathon: Test Script for Sats & Lightning Queries
# -------------------------------------------------------------

PORT="${PORT:-8080}"
BASE_URL="http://localhost:$PORT"

echo "==========================================================="
echo " 🌿 GreenTech Bitcoin/Lightning & M-Pesa Test Suite"
echo " Target Server: $BASE_URL"
echo "==========================================================="

echo ""
echo "1️⃣  Checking Blink Wallet Balances..."
curl -s "$BASE_URL/api/blink/wallet" | jq .

echo ""
echo "2️⃣  Querying Real-Time KES -> Sats Conversion (KES 500)..."
curl -s "$BASE_URL/api/blink/price?amount=500" | jq .

echo ""
echo "3️⃣  Generating Lightning Invoice for 50 Sats..."
curl -s -X POST "$BASE_URL/api/lightning/create-invoice" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50, "memo": "GreenTech Hotel Order #101"}' | jq .

echo ""
echo "4️⃣  Generating Lightning Invoice from Fiat (KES 250 -> Sats)..."
curl -s -X POST "$BASE_URL/api/lightning/create-invoice" \
  -H "Content-Type: application/json" \
  -d '{"kesAmount": 250, "memo": "Organic Waste Batch 25kg"}' | jq .

echo ""
echo "5️⃣  Simulating Incoming M-Pesa C2B Payment (KES 1,000)..."
curl -s -X POST "$BASE_URL/api/mpesa/c2b-confirmation" \
  -H "Content-Type: application/json" \
  -d '{"TransactionType":"Pay Bill","TransID":"MPESA98765","TransAmount":"1000","MSISDN":"254712345678"}' | jq .

echo ""
echo "✅ Done! Check your server terminal output to view the asynchronous Sats pipeline log."
