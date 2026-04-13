# grotto AI Service — MLflow Code Completion

This directory contains the **DevOps for AI** component of grotto.
It trains, tracks, versions, and serves a code completion model using [MLflow](https://mlflow.org).

---

## Architecture

```
mlflow-service/
├── train.py              # Fine-tune CodeGPT, log everything to MLflow
├── evaluate.py           # Gate: promote model to Production if metrics pass
├── serve.py              # FastAPI server — loads Production model from MLflow
├── grotto-ai-wrapper.sh  # Connects grotto's --ai panel to the REST endpoint
├── Dockerfile            # Container for the serve.py service
├── docker-compose.yml    # Runs MLflow server + grotto-ai together
├── requirements.txt      # Python dependencies
└── ai-pipeline.yml       # GitHub Actions workflow (copy to .github/workflows/)
```

---

## Quickstart (Local)

### 1. Install dependencies
```bash
cd mlflow-service
pip install -r requirements.txt
```

### 2. Train the model
```bash
python train.py --epochs 2 --lr 3e-5
```
This will:
- Fine-tune `microsoft/CodeGPT-small-py` on a curated Python snippet dataset
- Log all hyperparameters, metrics, and artifacts to MLflow
- Register the model in the **MLflow Model Registry**
- Write `run_meta.json` with the run ID and final perplexity

### 3. View the MLflow UI
```bash
mlflow ui --port 5000
```
Open http://localhost:5000 to see:
- Experiment runs with params and metrics
- Perplexity curves across epochs
- Registered model versions

### 4. Promote the model to Production
```bash
python evaluate.py --perplexity-threshold 200
```
If perplexity ≤ 200, the model is promoted to **Production** in the registry and tagged `champion`.

### 5. Start the serving API
```bash
python serve.py
```
The API is now live at http://localhost:8000.

Test it:
```bash
curl -X POST http://localhost:8000/complete \
  -H "Content-Type: application/json" \
  -d '{"prompt": "def binary_search(arr, target):\n", "max_new_tokens": 60}'
```

### 6. Connect to grotto
```bash
chmod +x grotto-ai-wrapper.sh
grotto --ai "./grotto-ai-wrapper.sh"
```
The AI panel in grotto will now send prompts to the local MLflow-backed service.

---

## Using Docker Compose

```bash
# Start MLflow server + grotto-ai service
docker-compose up

# MLflow UI:        http://localhost:5000
# Completion API:   http://localhost:8000
# Health check:     http://localhost:8000/health
```

Note: Run `train.py` and `evaluate.py` locally first to populate the MLflow DB
before starting the compose stack.

---

## CI/CD Pipeline

Copy `ai-pipeline.yml` to `.github/workflows/`:
```bash
cp ai-pipeline.yml ../.github/workflows/
```

The pipeline has **3 jobs**:

```
Push to main (mlflow-service/** changed)
        │
        ├──► train      Train model → log to MLflow → upload artifacts
        │
        ├──► evaluate   Download artifacts → check perplexity gate
        │               → promote to Production in MLflow registry
        │               → fail CI if quality threshold not met
        │
        └──► build-and-push   Build Docker image → push to ghcr.io
                              (only on main, not PRs)
```

### Manual trigger with custom hyperparameters
In GitHub → Actions → AI Pipeline → Run workflow:
- `epochs` — number of training epochs
- `learning_rate` — optimizer LR
- `perplexity_threshold` — max acceptable perplexity for promotion gate

---

## API Reference

| Endpoint | Method | Description |
|---|---|---|
| `/health` | GET | Health check, model load status |
| `/complete` | POST | Generate code completion |
| `/model/info` | GET | Current model version metadata |
| `/model/reload` | POST | Hot-reload model (after new promotion) |

### POST /complete
```json
{
  "prompt": "def binary_search(arr, target):\n",
  "max_new_tokens": 60,
  "temperature": 0.7,
  "top_p": 0.95
}
```

---

## MLflow Concepts Used

| Concept | Where |
|---|---|
| Experiments | `grotto-code-completion` experiment groups all runs |
| Runs | Each `train.py` execution = one run with params + metrics |
| Artifacts | Model weights, sample completions stored per run |
| Model Registry | Versioned model store with Staging/Production stages |
| Model Alias | `champion` alias points to current Production version |
| Metric logging | `perplexity_initial`, `epoch_perplexity`, `perplexity_final` |