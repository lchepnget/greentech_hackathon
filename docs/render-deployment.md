# Deploy RegenFeed on Render

The repository includes a Render Blueprint in `render.yaml`. It creates:

- `regenfeed-db`: PostgreSQL
- `regenfeed-api`: Go/Buffalo backend
- `regenfeed-web`: SvelteKit frontend

## Deploy

1. Push this repository to GitHub or GitLab. Do not commit `.env` files.
2. In Render, select **New → Blueprint** and connect the repository.
3. Render reads `render.yaml`. Enter every environment value marked `sync: false`.
4. After Render assigns the service URLs, set:
   - Backend `FRONTEND_ORIGIN` to `https://regenfeed-web.onrender.com` (use the actual generated frontend URL).
   - Frontend `PUBLIC_API_BASE_URL` to `https://regenfeed-api.onrender.com` (use the actual generated backend URL; `/api` is added automatically).
5. Redeploy both services after saving those two URLs.

If you created the services manually, set the API service **Root Directory** to
`backend`, or use these commands exactly:

```text
Build:  cd backend && go build -o bin/regenfeed ./cmd/app && go build -o bin/migrate ./cmd/migrate
Start: cd backend && ./bin/migrate && ./bin/regenfeed
```

## Required backend secrets

- `BLINK_API_KEY`
- `BLINK_WALLET_ID`
- `BLINK_PLATFORM_WALLET_ADDRESS`
- M-Pesa credentials and callback URLs listed in `render.yaml`

Render generates `SESSION_SECRET` and injects `DATABASE_URL`. The backend start command applies the SQL migration once and records it in `regenfeed_migrations` before starting the API. This works on Render free-tier services, which do not support pre-deploy commands.

## Payment callback URLs

Set M-Pesa callback values to the public backend URL, for example:

```text
https://regenfeed-api.onrender.com/api/mpesa/c2b-confirmation
https://regenfeed-api.onrender.com/api/mpesa/c2b-validation
https://regenfeed-api.onrender.com/api/mpesa/b2c-result
https://regenfeed-api.onrender.com/api/mpesa/b2c-timeout
```

Replace the hostname with the one Render assigns. Blink credentials remain backend-only and must never be added to the frontend service.

## Verification

After deployment, check:

```text
https://regenfeed-api.onrender.com/api/listings/summary/
https://regenfeed-web.onrender.com/
```

Then register a test user, create a small Blink deposit invoice, and confirm that its QR opens a real `lightning:` invoice.
