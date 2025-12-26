from __future__ import annotations

import asyncio
import logging
import random
import time
from dataclasses import dataclass
from typing import Final, TypedDict
from venv import logger

import httpx

API_URL = "http://localhost:5000/add-order"

USER_COUNT = 500
PAIR_IDS: Final[list[str]] = [
    "btcusdt",
    "ethusdt",
    "bnbusdt",
    "solusdt",
    "adausdt",
    "xrpusdt",
    "dogeusdt",
    "avaxusdt",
    "dotusdt",
    "linkusdt",
]


MIN_OPS: Final[int] = 2
MAX_OPS: Final[int] = 4

PAIR_PRICE_MODEL: Final[dict[str, tuple[float, float]]] = {
    "btcusdt": (65_000, 1_200),
    "ethusdt": (3_200, 120),
    "bnbusdt": (600, 25),
    "solusdt": (150, 10),
    "adausdt": (0.6, 0.05),
    "xrpusdt": (0.7, 0.06),
    "dogeusdt": (0.15, 0.02),
    "avaxusdt": (45, 4),
    "dotusdt": (7, 0.7),
    "linkusdt": (18, 1.5),
}


AMOUNT_MEAN: Final[float] = 2.0
AMOUNT_STDDEV: Final[float] = 1.2
MIN_AMOUNT: Final[float] = 0.01

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s | %(levelname)s | %(message)s",
)


class OrderPayload(TypedDict):
    price: float
    amount: float
    pair_id: str
    account_id: int
    type: int


@dataclass(frozen=True)
class Account:
    id: int


def normal_bounded(mean: float, stddev: float, min_value: float) -> float:
    value = random.gauss(mean, stddev)
    return max(value, min_value)


def generate_order(accounts: list[Account]) -> OrderPayload:
    account = random.choice(accounts)
    pair = random.choice(PAIR_IDS)

    price_mean, price_stddev = PAIR_PRICE_MODEL[pair]
    price = normal_bounded(price_mean, price_stddev, min_value=0.0001)
    amount = normal_bounded(AMOUNT_MEAN, AMOUNT_STDDEV, MIN_AMOUNT)

    order_type = 0 if random.random() < 0.52 else 1

    return {
        "price": round(price, 4),
        "amount": round(amount, 4),
        "pair_id": pair,
        "account_id": account.id,
        "type": order_type,
    }


async def send_order(
    client: httpx.AsyncClient,
    payload: OrderPayload,
) -> None:
    try:
        response = await client.post(API_URL, json=payload, timeout=2.0)
        response.raise_for_status()

        logger.info(
            "order placed | account=%s pair=%s type=%s price=%.4f amount=%.4f",
            payload["account_id"],
            payload["pair_id"],
            "BUY" if payload["type"] == 0 else "SELL",
            payload["price"],
            payload["amount"],
        )

    except Exception as exc:
        logger.error(
            "order failed | account=%s pair=%s error=%s",
            payload["account_id"],
            payload["pair_id"],
            exc,
        )


async def main() -> None:
    accounts = [Account(id=i) for i in range(1, USER_COUNT + 1)]

    async with httpx.AsyncClient() as client:
        logger.info("Market simulator started")

        while True:
            start = time.perf_counter()

            ops = random.uniform(MIN_OPS, MAX_OPS)
            orders_this_tick = max(1, int(ops))

            tasks = [
                send_order(client, generate_order(accounts))
                for _ in range(orders_this_tick)
            ]

            await asyncio.gather(*tasks)

            elapsed = time.perf_counter() - start
            sleep_time = max(0.0, 1.0 - elapsed)

            await asyncio.sleep(sleep_time)


asyncio.run(main())
