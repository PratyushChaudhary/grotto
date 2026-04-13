"""
serve.py — FastAPI server that loads the Production model from MLflow
and exposes a /complete endpoint for grotto's AI panel.

Usage:
    python serve.py                    # loads Production model from MLflow
    python serve.py --port 8080
    python serve.py --model-version 2  # explicit version

Grotto integration:
    grotto --ai "curl -s -X POST http://localhost:8000/complete ..."
    OR pipe via a small wrapper script (see grotto-ai-wrapper.sh)
"""

import os
from contextlib import asynccontextmanager
from typing import Optional

import mlflow
import mlflow.pytorch
import torch
import uvicorn
from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from mlflow.tracking import MlflowClient
from pydantic import BaseModel, Field
from transformers import AutoTokenizer

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
MODEL_NAME = "grotto-code-completion"
BASE_MODEL = "microsoft/CodeGPT-small-py"
DEFAULT_PORT = 8000

# ---------------------------------------------------------------------------
# Global model state
# ---------------------------------------------------------------------------
model_state: dict = {
    "model": None,
    "tokenizer": None,
    "version": None,
    "run_id": None,
    "device": None,
}


def load_production_model(version: Optional[int] = None):
    """Load the Production (or specified) model version from MLflow."""
    client = MlflowClient()
    device = "cuda" if torch.cuda.is_available() else "cpu"

    if version is not None:
        model_uri = f"models:/{MODEL_NAME}/{version}"
        ver_str = str(version)
    else:
        # Load the Production alias / stage
        try:
            mv = client.get_model_version_by_alias(MODEL_NAME, "champion")
            model_uri = f"models:/{MODEL_NAME}@champion"
            ver_str = mv.version
        except Exception:
            # Fallback: latest Production stage
            versions = client.get_latest_versions(MODEL_NAME, stages=["Production"])
            if not versions:
                raise RuntimeError(
                    f"No Production version found for '{MODEL_NAME}'. "
                    "Run train.py then evaluate.py first."
                )
            ver_str = versions[0].version
            model_uri = f"models:/{MODEL_NAME}/{ver_str}"

    print(f"Loading model: {model_uri} on {device}")
    pytorch_model = mlflow.pytorch.load_model(model_uri, map_location=device)
    pytorch_model.eval()

    tokenizer = AutoTokenizer.from_pretrained(BASE_MODEL)
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token

    model_state["model"] = pytorch_model
    model_state["tokenizer"] = tokenizer
    model_state["version"] = str(ver_str)
    model_state["device"] = device
    print(f"Model v{ver_str} loaded successfully.")


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup: load model
    model_version = os.environ.get("MODEL_VERSION")
    load_production_model(int(model_version) if model_version else None)
    yield
    # Shutdown: cleanup
    model_state["model"] = None


app = FastAPI(
    title="Grotto Code Completion Service",
    description="MLflow-backed code completion endpoint for the grotto TUI editor",
    version="1.0.0",
    lifespan=lifespan,
)

app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_methods=["*"],
    allow_headers=["*"],
)


# ---------------------------------------------------------------------------
# Schemas
# ---------------------------------------------------------------------------
class CompletionRequest(BaseModel):
    prompt: str = Field(..., description="Code prefix to complete", min_length=1)
    max_new_tokens: int = Field(60, ge=1, le=256, description="Max tokens to generate")
    temperature: float = Field(0.7, ge=0.1, le=2.0, description="Sampling temperature")
    top_p: float = Field(0.95, ge=0.1, le=1.0, description="Nucleus sampling p")


class CompletionResponse(BaseModel):
    prompt: str
    completion: str
    full_text: str
    model_version: str
    tokens_generated: int


class HealthResponse(BaseModel):
    status: str
    model_loaded: bool
    model_version: Optional[str]
    device: Optional[str]


# ---------------------------------------------------------------------------
# Endpoints
# ---------------------------------------------------------------------------
@app.get("/health", response_model=HealthResponse)
def health():
    """Health check — used by CI to verify the server started correctly."""
    return HealthResponse(
        status="ok",
        model_loaded=model_state["model"] is not None,
        model_version=model_state.get("version"),
        device=model_state.get("device"),
    )


@app.post("/complete", response_model=CompletionResponse)
def complete(req: CompletionRequest):
    """Generate a code completion for the given prompt."""
    if model_state["model"] is None:
        raise HTTPException(status_code=503, detail="Model not loaded")

    model = model_state["model"]
    tokenizer = model_state["tokenizer"]
    device = model_state["device"]

    inputs = tokenizer(req.prompt, return_tensors="pt").to(device)
    input_len = inputs["input_ids"].shape[1]

    with torch.no_grad():
        output = model.generate(
            **inputs,
            max_new_tokens=req.max_new_tokens,
            do_sample=True,
            temperature=req.temperature,
            top_p=req.top_p,
            pad_token_id=tokenizer.eos_token_id,
        )

    full_text = tokenizer.decode(output[0], skip_special_tokens=True)
    completion = tokenizer.decode(output[0][input_len:], skip_special_tokens=True)
    tokens_generated = output.shape[1] - input_len

    return CompletionResponse(
        prompt=req.prompt,
        completion=completion,
        full_text=full_text,
        model_version=model_state["version"],
        tokens_generated=tokens_generated,
    )


@app.get("/model/info")
def model_info():
    """Return current model metadata from MLflow."""
    if model_state["version"] is None:
        raise HTTPException(status_code=503, detail="Model not loaded")

    client = MlflowClient()
    mv = client.get_model_version(MODEL_NAME, model_state["version"])
    return {
        "name": MODEL_NAME,
        "version": mv.version,
        "stage": mv.current_stage,
        "run_id": mv.run_id,
        "creation_timestamp": mv.creation_timestamp,
        "tags": mv.tags,
    }


@app.post("/model/reload")
def reload_model(version: Optional[int] = None):
    """Hot-reload the model (e.g. after a new Production version is promoted)."""
    load_production_model(version)
    return {"status": "reloaded", "version": model_state["version"]}


# ---------------------------------------------------------------------------
# Entry point
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument("--port", type=int, default=DEFAULT_PORT)
    parser.add_argument("--host", type=str, default="0.0.0.0")
    parser.add_argument("--model-version", type=int, default=None)
    args = parser.parse_args()

    if args.model_version:
        os.environ["MODEL_VERSION"] = str(args.model_version)

    uvicorn.run(app, host=args.host, port=args.port)