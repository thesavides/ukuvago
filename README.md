# UkuvaGo - Angel Investment Platform

A comprehensive platform connecting startup developers with angel investors, featuring secure project showcasing, digital NDA signing, payment processing, and SAFE note term sheet generation.

## Features

- **User Registration**: Separate flows for investors, developers, and admins
- **Digital NDA Signing**: Electronic signature capture with legal compliance
- **Payment Processing**: Stripe integration for viewing fees ($500 for 4 project views)
- **Project Management**: Developers can submit projects for admin approval
- **Investment Offers**: Investors can make offers on approved projects
- **SAFE Note Generation**: Automated term sheet creation with dual signatures

## Quick Start

### Prerequisites

- Go 1.21+
- Node.js (optional, for frontend development)

### Installation

1. Clone the repository
2. Install dependencies:

```bash
cd UkuvaGo
go mod tidy
```

3. Run the server:

```bash
go run cmd/server/main.go
```

4. Open http://localhost:8080 in your browser

### Default Admin Account

- Email: admin@ukuvago.com
- Password: admin123 (change this in production!)

## Configuration

Set environment variables or use defaults:

| Variable | Default | Description |
|----------|---------|-------------|
| SERVER_PORT | 8080 | Server port |
| DATABASE_TYPE | sqlite | "sqlite" or "postgres" |
| DATABASE_URL | ukuvago.db | Database connection string |
| JWT_SECRET | (random) | JWT signing key |
| STRIPE_SECRET_KEY | | Stripe API key (optional) |
| VIEW_FEE_AMOUNT | 50000 | View fee in cents ($500) |
| MAX_PROJECT_VIEWS | 4 | Projects viewable per payment |

## API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login
- `GET /api/auth/me` - Get current user

### Projects
- `GET /api/projects` - List approved projects (public)
- `GET /api/projects/:id` - View project (requires NDA + payment)
- `POST /api/projects` - Create project (developer)

### NDA
- `GET /api/nda/template` - Get NDA content
- `POST /api/nda/sign` - Sign NDA

### Payments
- `POST /api/payments/create-intent` - Create payment
- `POST /api/payments/confirm` - Confirm payment

### Offers
- `POST /api/offers` - Submit investment offer
- `POST /api/offers/:id/respond` - Accept/reject offer

### Admin
- `GET /api/admin/stats` - Dashboard statistics
- `POST /api/admin/projects/:id/approve` - Approve project

## Project Structure

```
UkuvaGo/
├── cmd/server/           # Application entry point
├── internal/
│   ├── config/           # Configuration
│   ├── database/         # Database layer
│   ├── handlers/         # API handlers
│   ├── middleware/       # Auth, NDA, payment middleware
│   ├── models/           # Data models
│   ├── routes/           # Route definitions
│   └── services/         # Business logic
├── web/                  # Frontend assets
└── uploads/              # Uploaded files
```

## License

Proprietary - All rights reserved
