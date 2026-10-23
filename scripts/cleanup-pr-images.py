import json
import os
import re
import subprocess
import urllib.parse
from pathlib import Path


def api(path, method="GET", missing_ok=False):
    response = subprocess.run(
        ["gh", "api", "--method", method, path],
        capture_output=True,
        text=True,
        check=False,
    )
    if response.returncode:
        if missing_ok and "HTTP 404" in response.stderr:
            return None
        raise RuntimeError(f"GitHub API {method} failed for {path}: {response.stderr}")
    return json.loads(response.stdout) if response.stdout.strip() else None


def paginate(path):
    page = 1
    while True:
        separator = "&" if "?" in path else "?"
        items = api(f"{path}{separator}per_page=100&page={page}")
        if not isinstance(items, list):
            raise RuntimeError(f"Expected a list from {path}")
        yield from items
        if len(items) < 100:
            return
        page += 1


def should_delete(version, pr):
    tags = version.get("metadata", {}).get("container", {}).get("tags") or []
    return any(re.fullmatch(rf"pr-{pr}-[0-9a-f]{{40}}", tag) for tag in tags)


def repository_packages(repository):
    owner, repository_name = repository.split("/", 1)
    account = api(f"/users/{owner}")
    namespace = "orgs" if account["type"] == "Organization" else "users"
    matrix_path = Path(__file__).resolve().parents[1] / ".github/service-images.json"
    matrix = json.loads(matrix_path.read_text())
    for image in matrix["include"]:
        name = urllib.parse.quote(
            (repository_name + image["image-suffix"]).lower(), safe=""
        )
        path = f"/{namespace}/{owner}/packages/container/{name}"
        package = api(path, missing_ok=True)
        if package is None:
            print(f"skipping unpublished package {name}")
            continue
        linked_repository = (package.get("repository") or {}).get("full_name", "")
        if linked_repository.casefold() != repository.casefold():
            raise RuntimeError(f"Package {name} is not linked to {repository}")
        yield f"{path}/versions"


def main():
    repository = os.environ["REPOSITORY"]
    pr = int(os.environ["PR"])
    if pr <= 0:
        raise ValueError("PR must be positive")
    pull_request = api(f"/repos/{repository}/pulls/{pr}")
    if pull_request["state"] != "closed":
        raise ValueError("Cleanup requires a closed pull request")
    for path in repository_packages(repository):
        versions = list(paginate(path))
        for version in versions:
            if should_delete(version, pr):
                print(f"deleting {path}/{version['id']}")
                api(f"{path}/{version['id']}", method="DELETE", missing_ok=True)


if __name__ == "__main__":
    main()
