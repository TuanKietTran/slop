"""FastAPI server exposing review results to the UI and VSCode extension."""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware

from main import reviews

app = FastAPI(title="Big Review Agent API")

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


@app.get("/reviews")
async def list_reviews() -> list[dict]:
    return list(reviews.values())


@app.get("/reviews/ready")
async def reviews_ready() -> list[dict]:
    return [r for r in reviews.values() if r.get("status") == "big-review-done"]


@app.get("/reviews/{review_id}")
async def get_review(review_id: str) -> dict:
    if review_id not in reviews:
        raise HTTPException(status_code=404, detail="Review not found")
    return reviews[review_id]
