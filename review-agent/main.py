"""Big review agent: consumes runner outputs from the review-queue, aggregates
findings across N runners, dedups, ranks by severity, emits a ReviewReport."""

import asyncio
import json
import os
import uuid
from pathlib import Path

import anthropic
import nats

# In-memory store shared with FastAPI server.
reviews: dict[str, dict] = {}

client = anthropic.Anthropic(api_key=os.environ.get("ANTHROPIC_API_KEY", ""))

BIG_REVIEW_PROMPT = (Path(__file__).parent / "prompts" / "big-review.md").read_text()


def aggregate(runner_outputs: list[dict]) -> dict:
    """Merge test results + diffs + self-reviews from each runner."""
    merged: dict = {"tests": [], "results": [], "self_reviews": []}
    for o in runner_outputs:
        for k in merged:
            items = o.get(k, [])
            if isinstance(items, list):
                merged[k].extend(items)
            elif items:
                merged[k].append(items)
    return merged


def call_big_review(merged: dict) -> dict:
    msg = client.messages.create(
        model="claude-opus-4-5",
        max_tokens=4096,
        system=BIG_REVIEW_PROMPT,
        messages=[{"role": "user", "content": json.dumps(merged)}],
    )
    text = "".join(b.text for b in msg.content if b.type == "text")
    try:
        return json.loads(text)
    except json.JSONDecodeError:
        return {"report": text, "findings": [], "severity": "unknown"}


async def handle_message(msg) -> None:
    data = json.loads(msg.data.decode())
    task_id = data.get("task_id", str(uuid.uuid4()))

    # Aggregate artifacts from the payload.
    runner_outputs = [data] if data else []
    merged = aggregate(runner_outputs)

    print(f"[review-agent] processing task {task_id}")
    review_result = call_big_review(merged)
    review_result["task_id"] = task_id
    review_result["status"] = "big-review-done"
    reviews[task_id] = review_result
    print(f"[review-agent] review complete for {task_id}")


async def main() -> None:
    nats_url = os.environ.get("NATS_URL", "nats://localhost:4222")
    print(f"[review-agent] connecting to NATS at {nats_url}")

    nc = await nats.connect(nats_url)
    sub = await nc.subscribe("review-queue", cb=handle_message)
    print("[review-agent] subscribed to review-queue")

    try:
        while True:
            await asyncio.sleep(1)
    except KeyboardInterrupt:
        pass
    finally:
        await sub.unsubscribe()
        await nc.drain()


if __name__ == "__main__":
    asyncio.run(main())
