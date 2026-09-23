import pytest

from tools.check_coverage import passes
from tools.run_mutation import evaluate, mutant_pattern, select_functions


@pytest.mark.parametrize(
    "covered,total,expected",
    [
        (90, 100, False),
        (91, 100, True),
        (89999, 100000, False),
        (90001, 100000, True),
        (0, 0, False),
        (101, 100, False),
    ],
    ids=[
        "equality-fails",
        "above-passes",
        "rounded-90-fails",
        "unrounded-above-passes",
        "empty-fails",
        "invalid-fails",
    ],
)
def test_strict_coverage_threshold(covered, total, expected):
    assert passes({"covered_lines": covered, "num_statements": total}) is expected


def test_changed_function_selection_does_not_include_unchanged_sibling():
    before = "def changed(x):\n    return x > 1\n\ndef unchanged(x):\n    return x + 1\n"
    after = before.replace("x > 1", "x > 2")
    assert select_functions(before, after)["functions"] == ["changed"]
    assert select_functions(after, after)["functions"] == []


def test_new_moved_deleted_and_module_changes_are_visible():
    assert select_functions("", "def new():\n    return 1\n")["functions"] == ["new"]
    assert select_functions("def old():\n    return 1\n", "")["deleted_functions"] == ["old"]
    source = "def moved():\n    return 1\n"
    assert select_functions(source, "\n" + source)["functions"] == ["moved"]
    before = "LIMIT = 1\n" + source
    result = select_functions(before, before.replace("LIMIT = 1", "LIMIT = 2"))
    assert result["functions"] == ["moved"]
    assert result["module_changed"]


def test_method_filter_does_not_match_sibling_function():
    assert (
        mutant_pattern({"file": "app/model.py", "function": "Adapter.predict"})
        == "app.model.xǁAdapterǁpredict__mutmut_*"
    )


@pytest.mark.parametrize(
    "outcomes,expected",
    [
        (["killed"] * 4 + ["survived"], "FAIL"),
        (["killed"] * 5 + ["survived"], "PASS"),
        (["timeout"], "FAIL"),
        (["killed", "suspicious"], "FAIL"),
        ([], "N/A"),
        (["no tests"], "FAIL"),
        (["error"], "FAIL"),
    ],
)
def test_mutation_results_never_certify_unresolved_or_borderline(outcomes, expected):
    assert evaluate(outcomes)["status"] == expected


def test_unsupported_changes_do_not_pass_as_empty():
    assert evaluate([], unsupported=True)["status"] == "FAIL"


def test_manifest_records_untracked_unsupported_deleted_and_explicit_base(tmp_path):
    import subprocess

    from tools.run_mutation import selection_manifest

    def git(*args):
        return subprocess.check_output(["git", *args], cwd=tmp_path, text=True)

    git("init", "-q")
    app = tmp_path / "src/fraud-service/app"
    app.mkdir(parents=True)
    source = app / "rules.py"
    source.write_text("def old():\n    return 1\n")
    git("add", ".")
    git("-c", "user.name=Verification", "-c", "user.email=verification@example.invalid", "commit", "-qm", "fixture")
    base = git("rev-parse", "HEAD").strip()
    empty = selection_manifest(base, tmp_path)
    assert empty["selected"] == empty["unsupported"] == []
    source.unlink()
    (app / "new.py").write_text("def added():\n    return True\n")
    (app / "constants.py").write_text("LIMIT = 1\n")
    manifest = selection_manifest(base, tmp_path)
    assert manifest["deleted"] == ["app/rules.py:old"]
    assert manifest["selected"][0]["function"] == "added"
    assert manifest["unsupported"][0]["file"] == "app/constants.py"
    with pytest.raises(subprocess.CalledProcessError):
        selection_manifest("not-a-revision", tmp_path)
