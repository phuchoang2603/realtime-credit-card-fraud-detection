from __future__ import annotations

import pickle
import warnings
from pathlib import Path
from typing import Any, Protocol

from sklearn.exceptions import InconsistentVersionWarning


class PredictionModel(Protocol):
    def predict_proba(self, features: Any) -> Any: ...


def load_model(path: Path) -> PredictionModel:
    """Load a trusted, deployment-owned artifact built with the serving runtime.

    Version drift is a deployment error, not a reason to patch sklearn internals.
    """
    with warnings.catch_warnings():
        warnings.simplefilter("error", InconsistentVersionWarning)
        with path.open("rb") as model_file:
            model = pickle.load(model_file)
    if not callable(getattr(model, "predict_proba", None)):
        raise ValueError("Model must implement predict_proba")
    return model
