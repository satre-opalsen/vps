# Telegram File to Link / Stream Bot

A lightweight, high-performance Telegram bot that converts documents, videos, and audio files up to **2 GB** into direct download links and streaming URLs.

## Features
- **100% Free** personal usage.
- Streams files directly from Telegram cloud (no local disk storage needed).
- Supports HTTP Range requests (`HTTP 206 Partial Content`) for video seeking and resume support in download managers (IDM, Aria2, curl).
- Owner-locked: only your Telegram User ID can upload files.
- Safe storage: files are copied to your private channel so links don't break if you delete your local chat.

## Deploying on Render (Free)
1. Fork or push this repository to GitHub.
2. Sign up on [Render.com](https://render.com).
3. Click **New +** -> **Web Service** -> Select this repository.
4. Set Environment Variables:
   - `API_ID`: Your Telegram API ID from my.telegram.org
   - `API_HASH`: Your Telegram API Hash from my.telegram.org
   - `BOT_TOKEN`: Token from @BotFather
   - `BIN_CHANNEL`: Private Channel ID (e.g., `-100...`)
   - `OWNER_ID`: Your numerical Telegram User ID
   - `FQDN`: `https://your-service-name.onrender.com`
5. Click **Deploy Web Service**!
