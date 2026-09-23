import pickle
from pathlib import Path
from threading import Event

import numpy as np
import pytest
from opentelemetry.sdk.trace.export.in_memory_span_exporter import InMemorySpanExporter
from sklearn.dummy import DummyClassifier

from app.model import load_model
from app.utils.tracing_config import TracingRuntime


def test_trace_cleanup_is_bounded(caplog):
    release = Event()

    class StuckExporter(InMemorySpanExporter):
        def shutdown(self):
            release.wait(2)

    runtime = TracingRuntime(True, "bounded", StuckExporter)
    runtime.start()
    try:
        runtime.stop(timeout=0.01)
        assert "cleanup budget" in caplog.text
        assert runtime.provider is None
    finally:
        release.set()


def test_model_loader_accepts_current_artifacts_and_rejects_non_models(tmp_path):
    model = DummyClassifier().fit([[0], [1]], [0, 1])
    path = tmp_path / "model.pkl"
    path.write_bytes(pickle.dumps(model))
    np.testing.assert_allclose(load_model(path).predict_proba([[0]]), [[0.5, 0.5]])
    path.write_bytes(pickle.dumps({"not": "a model"}))
    with pytest.raises(ValueError, match="predict_proba"):
        load_model(path)
    with pytest.raises(FileNotFoundError):
        load_model(Path("/nonexistent/model.pkl"))


def test_model_loader_rejects_runtime_version_drift(tmp_path):
    model = DummyClassifier().fit([[0], [1]], [0, 1])
    import sklearn
    from sklearn.exceptions import InconsistentVersionWarning

    content = pickle.dumps(model, protocol=0).replace(sklearn.__version__.encode(), b"0.0.0")
    path = tmp_path / "outdated.pkl"
    path.write_bytes(content)
    with pytest.raises(InconsistentVersionWarning):
        load_model(path)
