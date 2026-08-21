RegenFeed
RegenFeed connects hotels and farmers to turn hotel waste into useful agricultural resources instead of sending it to dumpsites.
Stack
Frontend: Svelte
Backend: Go
Database: PostgreSQL
Payments: Bitcoin Lightning
DevOps: Docker, Render
Users
Hotels 🏨
Register/login
Manage profile & location
Upload waste with description/photos
Manage waste availability
View orders and payments
Farmers 🌱
Register/login
Manage profile & location
Browse available waste
Place orders
Pay with Bitcoin Lightning
Track orders and collection
Structure
FarmCycle/
├── frontend/       # Svelte frontend
├── backend/
│   └── smd/
│       └── app/
│           └── main.go
├── Dockerfile
├── docker-compose.yml
└── README.md
Run Frontend
npm i
npm run dev
Run Backend
cd backend
cd cmd
cd app
go run main.go
Basic Flow
Hotel → Upload Waste
          ↓
       Marketplace
          ↓
Farmer → Order → Lightning Payment
          ↓
       Collection
          ↓
       Completed
Goal: Reduce waste pollution while helping farmers access useful agricultural resources. 🌍
