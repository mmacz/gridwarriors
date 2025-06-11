# Running tests

```bash
cd qa
python -m venv .venv
source .venv/bin/activate
pip install -e .[dev]
pytest -v
```

