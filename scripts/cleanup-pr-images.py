import json
import os
import re
import subprocess
import urllib.parse


def api(path, method="GET", missing_ok=False):
    response = subprocess.run(
        ["gh", "api", "--method", method, path],
        capture_output=True, text=True, check=False,
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
    owner = repository.split("/", 1)[0]
    account = api(f"/users/{owner}")
    namespace = "orgs" if account["type"] == "Organization" else "users"
    for package in paginate(f"/{namespace}/{owner}/packages?package_type=container"):
        linked_repository = (package.get("repository") or {}).get("full_name", "")
        if linked_repository.casefold() == repository.casefold():
            name = urllib.parse.quote(package["name"], safe="")
            yield f"/{namespace}/{owner}/packages/container/{name}/versions"


def main():
    repository = os.environ["REPOSITORY"]
    pr = int(os.environ["PR"])
    if pr <= 0:
        raise ValueError("PR must be positive")
    for path in repository_packages(repository):
        versions = list(paginate(path))
        for version in versions:
            if should_delete(version, pr):
                print(f"deleting {path}/{version['id']}")
                api(f"{path}/{version['id']}", method="DELETE", missing_ok=True)


if __name__ == "__main__":
    main()
