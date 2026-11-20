import os
import re
import asyncio
import urllib.parse
from aiohttp import web
from dotenv import load_dotenv
from pyrogram import Client, filters
from pyrogram.types import Message

# Load .env file if present
load_dotenv()

API_ID = int(os.environ.get("API_ID", 0))
API_HASH = os.environ.get("API_HASH", "")
BOT_TOKEN = os.environ.get("BOT_TOKEN", "")
BIN_CHANNEL = int(os.environ.get("BIN_CHANNEL", 0))
OWNER_ID = int(os.environ.get("OWNER_ID", 0))
FQDN = os.environ.get("FQDN", "").rstrip("/")
PORT = int(os.environ.get("PORT", 10000))

CHUNK_SIZE = 1024 * 1024  # 1 MiB chunk size for streaming

bot = Client(
    "stream_bot",
    api_id=API_ID,
    api_hash=API_HASH,
    bot_token=BOT_TOKEN,
    in_memory=True
)

def get_media_info(message: Message):
    media = (
        message.document
        or message.video
        or message.audio
        or message.voice
        or message.video_note
        or message.photo
    )
    if not media:
        return None, None, None
    file_name = getattr(media, "file_name", f"file_{message.id}")
    file_size = getattr(media, "file_size", 0)
    mime_type = getattr(media, "mime_type", "application/octet-stream")
    return file_name, file_size, mime_type

async def chunk_streamer(client: Client, message: Message, from_bytes: int, until_bytes: int):
    offset_chunks = from_bytes // CHUNK_SIZE
    skip_initial_bytes = from_bytes % CHUNK_SIZE
    total_to_send = until_bytes - from_bytes + 1
    sent_bytes = 0

    async for chunk in client.stream_media(message, offset=offset_chunks):
        if skip_initial_bytes > 0:
            chunk = chunk[skip_initial_bytes:]
            skip_initial_bytes = 0

        if sent_bytes + len(chunk) > total_to_send:
            chunk = chunk[:total_to_send - sent_bytes]

        yield chunk
        sent_bytes += len(chunk)

        if sent_bytes >= total_to_send:
            break

routes = web.RouteTableDef()

@routes.get("/")
async def root_handler(request):
    return web.Response(text="Bot is running! Ready to stream files.")

@routes.get("/dl/{msg_id:\\d+}")
@routes.get("/stream/{msg_id:\\d+}")
async def stream_handler(request):
    msg_id = int(request.match_info["msg_id"])
    is_download = request.path.startswith("/dl/")

    try:
        msg = await bot.get_messages(chat_id=BIN_CHANNEL, message_ids=msg_id)
        if not msg or msg.empty:
            return web.Response(status=404, text="File not found or deleted from storage channel.")
    except Exception as e:
        return web.Response(status=500, text=f"Error retrieving message: {e}")

    file_name, file_size, mime_type = get_media_info(msg)
    if not file_name or file_size == 0:
        return web.Response(status=404, text="No media found in message.")

    quoted_filename = urllib.parse.quote(file_name)
    disposition = "attachment" if is_download else "inline"

    headers = {
        "Accept-Ranges": "bytes",
        "Content-Type": mime_type,
        "Content-Disposition": f'{disposition}; filename="{file_name}"; filename*=UTF-8\'\'{quoted_filename}',
    }

    range_header = request.headers.get("Range")
    if range_header:
        # HTTP 206 Partial Content (Supports video seeking & IDM/download resume)
        match = re.search(r"bytes=(\d+)-(\d*)", range_header)
        if match:
            from_bytes = int(match.group(1))
            until_bytes = int(match.group(2)) if match.group(2) else file_size - 1

            if from_bytes >= file_size or until_bytes >= file_size or from_bytes > until_bytes:
                return web.Response(status=416, headers={"Content-Range": f"bytes */{file_size}"})

            headers["Content-Range"] = f"bytes {from_bytes}-{until_bytes}/{file_size}"
            headers["Content-Length"] = str(until_bytes - from_bytes + 1)

            response = web.StreamResponse(status=206, headers=headers)
            await response.prepare(request)

            async for chunk in chunk_streamer(bot, msg, from_bytes, until_bytes):
                await response.write(chunk)
            await response.write_eof()
            return response

    # HTTP 200 Full Content
    headers["Content-Length"] = str(file_size)
    response = web.StreamResponse(status=200, headers=headers)
    await response.prepare(request)

    async for chunk in chunk_streamer(bot, msg, 0, file_size - 1):
        await response.write(chunk)
    await response.write_eof()
    return response

@bot.on_message(filters.private & (filters.document | filters.video | filters.audio | filters.photo | filters.voice))
async def handle_incoming_file(client: Client, message: Message):
    if OWNER_ID and message.from_user.id != OWNER_ID:
        await message.reply("Access denied. This bot is private.")
        return

    status = await message.reply("⏳ Processing file, please wait...")
    try:
        # Forward/copy to private bin channel
        stored_msg = await message.copy(chat_id=BIN_CHANNEL)
        msg_id = stored_msg.id

        file_name, file_size, _ = get_media_info(message)
        size_mb = f"{file_size / (1024 * 1024):.2f} MB"

        base_url = FQDN if FQDN else f"http://localhost:{PORT}"
        dl_link = f"{base_url}/dl/{msg_id}"
        stream_link = f"{base_url}/stream/{msg_id}"

        response_text = (
            f"**File Ready!**\n\n"
            f"📁 **Name:** `{file_name}`\n"
            f"📦 **Size:** `{size_mb}`\n\n"
            f"🚀 **Fast Download Link:**\n`{dl_link}`\n\n"
            f"▶️ **Stream / Play Online:**\n`{stream_link}`"
        )
        await status.edit(response_text, disable_web_page_preview=True)
    except Exception as e:
        await status.edit(f"❌ Error saving file: {e}")

@bot.on_message(filters.command("start") & filters.private)
async def start_command(client: Client, message: Message):
    await message.reply(
        "👋 **Welcome!**\n\n"
        "Send or forward any document, video, or audio file up to **2 GB** to get instant direct download and streaming links."
    )

async def main():
    app = web.Application()
    app.add_routes(routes)
    runner = web.AppRunner(app)
    await runner.setup()
    site = web.TCPSite(runner, "0.0.0.0", PORT)
    await site.start()
    print(f"Web server started on port {PORT}")

    await bot.start()
    print("Telegram bot started.")

    await asyncio.Event().wait()

if __name__ == "__main__":
    asyncio.run(main())
