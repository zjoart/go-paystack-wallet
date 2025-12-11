# Flux Wallet

A secure digital wallet service built with Go, Redis, and PostgreSQL. Features Google OAuth, atomic transactions, and Paystack integration.

## 🛠️ Local Installation

1.  **Configure Environment**
    Copy the example env file and update with your credentials:
    ```bash
    cp .env.example .env
    ```

2.  **Start Application**
    Docker Compose handles the database, redis, migrations, and app:
    ```bash
    make start-app
    ```

3.  **View Documentation**
    Access the interactive API docs at: [http://localhost:8080/swagger/index.html](http://localhost:8080/swagger/index.html)



### 1. Migrations
This repository uses **GitHub Actions** to handle migrations via ci automatically.
*   **Action**: Add `DATABASE_URL` to your GitHub Repository Secrets.

