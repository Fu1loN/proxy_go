import asyncio
import logging

from aiogram import Bot, Dispatcher, types
from aiogram.filters import Command


with open('token.txt') as f:
    TOKEN = f.read().strip()

# Включаем логирование, чтобы видеть ошибки и сообщения в консоли
logging.basicConfig(level=logging.INFO)

# Создаём объекты бота и диспетчера
bot = Bot(token=TOKEN)
dp = Dispatcher()

# Хэндлер на команду /start
@dp.message(Command("start"))
async def cmd_start(message: types.Message):
    await message.answer(
        "Привет! Я эхо-бот на aiogram.\n"
        "Отправь мне любое сообщение, и я повторю его.\n"
        "Команда /help — для справки."
    )

# Хэндлер на команду /help
@dp.message(Command("help"))
async def cmd_help(message: types.Message):
    await message.answer(
        "Просто напиши мне что угодно — я отвечу тем же.\n"
        "Команды: /start, /help"
    )

# Хэндлер на все текстовые сообщения (эхо)
@dp.message()
async def echo(message: types.Message):
    # Отправляем обратно текст пользователя с префиксом "Эхо: "
    # Можно также использовать message.answer или bot.send_message
    await message.reply(f"Эхо: {message.text}")

# Запуск бота
async def main():
    print("Бот запущен...")
    await dp.start_polling(bot)

if __name__ == "__main__":
    asyncio.run(main())