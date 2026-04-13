"""
evaluate.py — Evaluate a trained model and promote it in the MLflow Model Registry.

This is the "gate" in the CI/CD pipeline. If the model meets the quality
threshold it gets promoted from Staging → Production. Otherwise CI fails.

Usage:
    python evaluate.py                          # uses run_meta.json
    python evaluate.py --run-id <id>            # explicit run ID
    python evaluate.py --perplexity-threshold 150
"""

import argparse
import json
import os
import sys

import mlflow
from mlflow.tracking import MlflowClient

MODEL_NAME = "grotto-code-completion"
DEFAULT_PPL_THRESHOLD = 200.0   # max acceptable perplexity
DEFAULT_MIN_IMPROVEMENT = 0.0   # model must at least not get worse


def get_latest_version(client: MlflowClient, model_name: str) -> str:
    """Return the latest version number of a registered model."""
    versions = client.search_model_versions(f"name='{model_name}'")
    if not versions:
        raise RuntimeError(f"No versions found for model '{model_name}'")
    latest = sorted(versions, key=lambda v: int(v.version), reverse=True)[0]
    return latest.version


def evaluate(run_id: str | None, ppl_threshold: float, min_improvement: float):
    client = MlflowClient()

    # ── Load metrics ──────────────────────────────────────────────────────
    if run_id is None:
        # Read from file written by train.py
        if not os.path.exists("run_meta.json"):
            print("ERROR: run_meta.json not found. Run train.py first or pass --run-id.")
            sys.exit(1)
        with open("run_meta.json") as f:
            meta = json.load(f)
        run_id = meta["run_id"]
        final_ppl = meta["perplexity_final"]
        ppl_improvement = meta["perplexity_improvement"]
    else:
        run = client.get_run(run_id)
        final_ppl = run.data.metrics.get("perplexity_final")
        ppl_improvement = run.data.metrics.get("perplexity_improvement", 0.0)
        if final_ppl is None:
            print(f"ERROR: Run {run_id} has no 'perplexity_final' metric.")
            sys.exit(1)

    print(f"Evaluating run: {run_id}")
    print(f"  Final perplexity : {final_ppl:.2f}  (threshold: <= {ppl_threshold})")
    print(f"  PPL improvement  : {ppl_improvement:.2f}  (min required: >= {min_improvement})")

    # ── Gate checks ───────────────────────────────────────────────────────
    passed = True

    if final_ppl > ppl_threshold:
        print(f"FAIL: Perplexity {final_ppl:.2f} exceeds threshold {ppl_threshold}")
        passed = False

    if ppl_improvement < min_improvement:
        print(f"FAIL: Improvement {ppl_improvement:.2f} below minimum {min_improvement}")
        passed = False

    if not passed:
        # Log the gate decision back to the MLflow run
        with mlflow.start_run(run_id=run_id):
            mlflow.log_metric("gate_passed", 0)
        print("\nModel did NOT pass quality gate. Will not be promoted.")
        sys.exit(1)

    # ── Promote to Production ─────────────────────────────────────────────
    print("\nAll checks passed. Promoting model to Production...")

    version = get_latest_version(client, MODEL_NAME)
    print(f"  Model: {MODEL_NAME}  Version: {version}")

    # Transition to Production
    client.transition_model_version_stage(
        name=MODEL_NAME,
        version=version,
        stage="Production",
        archive_existing_versions=True,   # auto-archive older Production versions
    )
    print(f"  Promoted to Production (previous Production versions archived)")

    # Add descriptive tag and alias
    client.set_model_version_tag(MODEL_NAME, version, "gate_passed", "true")
    client.set_model_version_tag(MODEL_NAME, version, "run_id", run_id)
    client.set_registered_model_alias(MODEL_NAME, "champion", version)

    # Log gate decision back to run
    with mlflow.start_run(run_id=run_id):
        mlflow.log_metric("gate_passed", 1)
        mlflow.set_tag("promoted_to", "Production")
        mlflow.set_tag("model_version", version)

    print(f"\nSuccess! {MODEL_NAME} v{version} is now Production.")
    print(f"  Alias 'champion' → version {version}")

    # Write promotion metadata for downstream steps
    promo_meta = {
        "model_name": MODEL_NAME,
        "version": version,
        "run_id": run_id,
        "perplexity_final": final_ppl,
        "stage": "Production",
    }
    with open("promotion_meta.json", "w") as f:
        json.dump(promo_meta, f, indent=2)
    print(f"Saved promotion metadata to promotion_meta.json")


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------
if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Evaluate model and promote in MLflow registry")
    parser.add_argument("--run-id", type=str, default=None, help="MLflow run ID (default: read from run_meta.json)")
    parser.add_argument("--perplexity-threshold", type=float, default=DEFAULT_PPL_THRESHOLD,
                        help=f"Max acceptable perplexity (default: {DEFAULT_PPL_THRESHOLD})")
    parser.add_argument("--min-improvement", type=float, default=DEFAULT_MIN_IMPROVEMENT,
                        help=f"Min required perplexity improvement (default: {DEFAULT_MIN_IMPROVEMENT})")
    args = parser.parse_args()

    evaluate(
        run_id=args.run_id,
        ppl_threshold=args.perplexity_threshold,
        min_improvement=args.min_improvement,
    )