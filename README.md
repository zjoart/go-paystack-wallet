# Flux

A golang digital wallet system featuring **Google OAuth**, **Paystack Integrated Payments**, and **Atomic Transactions**. Built for reliability with a **Redis Event-Driven Architecture**.

## key Features
- 🔐 **Secure Authentication**: Seamless Google OAuth & JWT support.
- 💳 **Smart Wallet**: Credit/Debit ledger, atomic transfers, and Paystack funding.
- ⚡ **Event Driven**: Asynchronous webhook processing with Redis & Dead Letter Queues.
- 🔑 **API Security**: One-time view API keys with granular permissions.

## Quick Start

1. **Setup Env**:
   ```bash
   cp .env.example .env
   # Add your Google & Paystack keys
   ```

2. **Launch**:
   ```bash
   make start-app
   ```
   *Boots Database, Redis, runs migrations, and starts the server.*


> For all other commands, run `make help`.

## API Docs
Explore the interactive Swagger UI at `https://flux-wallet.up.railway.app/swagger/index.html` 
