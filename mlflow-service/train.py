"""
train.py — Fine-tune a code completion model and log everything to MLflow.

Uses microsoft/CodeGPT-small-py (a small GPT-2 model trained on Python code)
so it runs in CI without needing a GPU.

Usage:
    python train.py
    python train.py --epochs 3 --lr 5e-5 --max-length 128
"""

import argparse
import json
import math
import os
import time

import mlflow
import mlflow.pytorch
import torch
from torch.utils.data import DataLoader, Dataset
from transformers import AutoModelForCausalLM, AutoTokenizer

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
BASE_MODEL = "microsoft/CodeGPT-small-py"
MLFLOW_EXPERIMENT = "grotto-code-completion"

# Small curated dataset of Python code snippets for demo fine-tuning
TRAIN_SNIPPETS = [
    "def add(a, b):\n    return a + b\n",
    "def subtract(a, b):\n    return a - b\n",
    "def multiply(a, b):\n    return a * b\n",
    "def divide(a, b):\n    if b == 0:\n        raise ValueError('Cannot divide by zero')\n    return a / b\n",
    "def factorial(n):\n    if n <= 1:\n        return 1\n    return n * factorial(n - 1)\n",
    "def fibonacci(n):\n    if n <= 0:\n        return []\n    elif n == 1:\n        return [0]\n    seq = [0, 1]\n    while len(seq) < n:\n        seq.append(seq[-1] + seq[-2])\n    return seq\n",
    "def is_prime(n):\n    if n < 2:\n        return False\n    for i in range(2, int(n**0.5) + 1):\n        if n % i == 0:\n            return False\n    return True\n",
    "def reverse_string(s):\n    return s[::-1]\n",
    "def flatten(lst):\n    result = []\n    for item in lst:\n        if isinstance(item, list):\n            result.extend(flatten(item))\n        else:\n            result.append(item)\n    return result\n",
    "class Stack:\n    def __init__(self):\n        self.items = []\n    def push(self, item):\n        self.items.append(item)\n    def pop(self):\n        return self.items.pop()\n    def is_empty(self):\n        return len(self.items) == 0\n",
    "def read_file(path):\n    with open(path, 'r') as f:\n        return f.read()\n",
    "def write_file(path, content):\n    with open(path, 'w') as f:\n        f.write(content)\n",
    "import os\ndef list_files(directory):\n    return [f for f in os.listdir(directory) if os.path.isfile(os.path.join(directory, f))]\n",
    "def merge_dicts(d1, d2):\n    return {**d1, **d2}\n",
    "def chunk_list(lst, size):\n    return [lst[i:i+size] for i in range(0, len(lst), size)]\n",
]


# ---------------------------------------------------------------------------
# Dataset
# ---------------------------------------------------------------------------
class CodeDataset(Dataset):
    def __init__(self, snippets, tokenizer, max_length):
        self.encodings = tokenizer(
            snippets,
            truncation=True,
            padding="max_length",
            max_length=max_length,
            return_tensors="pt",
        )

    def __len__(self):
        return self.encodings["input_ids"].shape[0]

    def __getitem__(self, idx):
        input_ids = self.encodings["input_ids"][idx]
        return {"input_ids": input_ids, "labels": input_ids.clone()}


# ---------------------------------------------------------------------------
# Evaluation helpers
# ---------------------------------------------------------------------------
def compute_perplexity(model, dataloader, device):
    """Compute perplexity on the dataset — standard LM quality metric."""
    model.eval()
    total_loss = 0.0
    total_batches = 0
    with torch.no_grad():
        for batch in dataloader:
            input_ids = batch["input_ids"].to(device)
            labels = batch["labels"].to(device)
            outputs = model(input_ids=input_ids, labels=labels)
            total_loss += outputs.loss.item()
            total_batches += 1
    avg_loss = total_loss / max(total_batches, 1)
    return math.exp(avg_loss)


def sample_completion(model, tokenizer, prompt, max_new_tokens=40, device="cpu"):
    """Generate a sample completion for qualitative inspection."""
    model.eval()
    inputs = tokenizer(prompt, return_tensors="pt").to(device)
    with torch.no_grad():
        output = model.generate(
            **inputs,
            max_new_tokens=max_new_tokens,
            do_sample=True,
            temperature=0.7,
            pad_token_id=tokenizer.eos_token_id,
        )
    return tokenizer.decode(output[0], skip_special_tokens=True)


# ---------------------------------------------------------------------------
# Main training loop
# ---------------------------------------------------------------------------
def train(epochs: int, lr: float, max_length: int, batch_size: int):
    device = "cuda" if torch.cuda.is_available() else "cpu"
    print(f"Using device: {device}")

    # ── Load tokenizer + model ──────────────────────────────────────────────
    print(f"Loading base model: {BASE_MODEL}")
    tokenizer = AutoTokenizer.from_pretrained(BASE_MODEL)
    if tokenizer.pad_token is None:
        tokenizer.pad_token = tokenizer.eos_token
    model = AutoModelForCausalLM.from_pretrained(BASE_MODEL).to(device)

    # ── Dataset + DataLoader ────────────────────────────────────────────────
    dataset = CodeDataset(TRAIN_SNIPPETS, tokenizer, max_length)
    dataloader = DataLoader(dataset, batch_size=batch_size, shuffle=True)

    optimizer = torch.optim.AdamW(model.parameters(), lr=lr)

    # ── MLflow run ──────────────────────────────────────────────────────────
    mlflow.set_experiment(MLFLOW_EXPERIMENT)

    with mlflow.start_run() as run:
        run_id = run.info.run_id
        print(f"MLflow run ID: {run_id}")

        # Log hyperparameters
        mlflow.log_params({
            "base_model": BASE_MODEL,
            "epochs": epochs,
            "learning_rate": lr,
            "max_length": max_length,
            "batch_size": batch_size,
            "train_samples": len(TRAIN_SNIPPETS),
            "device": device,
        })

        # Log initial (pre-training) perplexity as baseline
        initial_ppl = compute_perplexity(model, dataloader, device)
        mlflow.log_metric("perplexity_initial", initial_ppl, step=0)
        print(f"Initial perplexity: {initial_ppl:.2f}")

        # ── Training ────────────────────────────────────────────────────────
        model.train()
        global_step = 0
        start_time = time.time()

        for epoch in range(1, epochs + 1):
            epoch_loss = 0.0
            for batch in dataloader:
                input_ids = batch["input_ids"].to(device)
                labels = batch["labels"].to(device)

                optimizer.zero_grad()
                outputs = model(input_ids=input_ids, labels=labels)
                loss = outputs.loss
                loss.backward()
                optimizer.step()

                epoch_loss += loss.item()
                global_step += 1
                mlflow.log_metric("train_loss", loss.item(), step=global_step)

            avg_epoch_loss = epoch_loss / len(dataloader)
            epoch_ppl = math.exp(avg_epoch_loss)
            mlflow.log_metric("epoch_loss", avg_epoch_loss, step=epoch)
            mlflow.log_metric("epoch_perplexity", epoch_ppl, step=epoch)
            print(f"Epoch {epoch}/{epochs} — loss: {avg_epoch_loss:.4f} — perplexity: {epoch_ppl:.2f}")

        training_time = time.time() - start_time
        mlflow.log_metric("training_time_seconds", training_time)

        # ── Final evaluation ─────────────────────────────────────────────────
        final_ppl = compute_perplexity(model, dataloader, device)
        ppl_improvement = initial_ppl - final_ppl
        mlflow.log_metric("perplexity_final", final_ppl)
        mlflow.log_metric("perplexity_improvement", ppl_improvement)
        print(f"Final perplexity: {final_ppl:.2f} (improved by {ppl_improvement:.2f})")

        # ── Sample completion (logged as artifact) ───────────────────────────
        sample_prompt = "def binary_search(arr, target):\n"
        completion = sample_completion(model, tokenizer, sample_prompt, device=device)
        sample_path = "/tmp/sample_completion.txt"
        with open(sample_path, "w") as f:
            f.write(f"Prompt:\n{sample_prompt}\n\nCompletion:\n{completion}\n")
        mlflow.log_artifact(sample_path, "samples")
        print(f"\nSample completion:\n{completion}\n")

        # ── Log model to MLflow Model Registry ───────────────────────────────
        print("Logging model to MLflow...")
        mlflow.pytorch.log_model(
            model,
            artifact_path="model",
            registered_model_name="grotto-code-completion",
        )

        # ── Save metadata for evaluate.py ────────────────────────────────────
        meta = {
            "run_id": run_id,
            "perplexity_final": final_ppl,
            "perplexity_improvement": ppl_improvement,
            "training_time_seconds": training_time,
        }
        with open("run_meta.json", "w") as f:
            json.dump(meta, f, indent=2)
        print(f"Saved run metadata to run_meta.json")

    return run_id, final_ppl


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Train code completion model with MLflow tracking")
    parser.add_argument("--epochs", type=int, default=2, help="Number of training epochs")
    parser.add_argument("--lr", type=float, default=3e-5, help="Learning rate")
    parser.add_argument("--max-length", type=int, default=128, help="Max token sequence length")
    parser.add_argument("--batch-size", type=int, default=4, help="Batch size")
    args = parser.parse_args()

    run_id, final_ppl = train(
        epochs=args.epochs,
        lr=args.lr,
        max_length=args.max_length,
        batch_size=args.batch_size,
    )
    print(f"\nDone. Run ID: {run_id} | Final perplexity: {final_ppl:.2f}")