import base64
import json
import os
import re
import subprocess
import urllib.parse


IMAGES = (
    "", "/edge", "/payments", "/notifier",
    "/vault-simulator", "/processor-simulator", "/connect-debezium",
)
CHARTS = (
    ("infra/charts/payment-gateway/values-prod.yaml", r"^imageTag:", IMAGES[1:6]),
    ("infra/charts/fraud-service/values-prod.yaml", r"^  tag:", ("",)),
    ("infra/charts/kafka/values-prod.yaml", r"^  imageTag:", ("/connect-debezium",)),
)


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
        items = api(f"{path}?per_page=100&page={page}")
        if not isinstance(items, list):
            raise RuntimeError(f"Expected a list from {path}")
        yield from items
        if len(items) < 100:
            return
        page += 1


def production_pins(repository, ref):
    pins = {image: set() for image in IMAGES}
    for filename, key, images in CHARTS:
        content = api(
            f"/repos/{repository}/contents/{filename}?ref={urllib.parse.quote(ref, safe='')}",
            missing_ok=True,
        )
        if content is None:
            continue
        text = base64.b64decode(content["content"]).decode("utf-8")
        matches = re.findall(key + r'''\s*["']?([A-Za-z0-9_.-]+)["']?\s*$''', text, re.MULTILINE)
        if len(matches) != 1:
            raise RuntimeError(f"Cannot determine production image pin in {repository}@{ref}:{filename}")
        for image in images:
            pins[image].add(matches[0])
    return pins


def should_delete(version, pr_tags, protected_tags):
    tags = set(version.get("metadata", {}).get("container", {}).get("tags") or [])
    return bool(tags & pr_tags) and not tags & (protected_tags | {"latest"})


def main():
    repository = os.environ["REPOSITORY"]
    owner, repo = repository.split("/", 1)
    pr_tags = {commit["sha"][:7] for commit in paginate(f"/repos/{repository}/pulls/{os.environ['PR']}/commits")}
    protected = production_pins(repository, os.environ["DEFAULT_BRANCH"])
    head_pins = production_pins(os.environ["HEAD_REPOSITORY"], os.environ["HEAD_SHA"])
    for image in IMAGES:
        protected[image].update(head_pins[image])
    for image in IMAGES:
        package = urllib.parse.quote(repo + image, safe="")
        path = f"/users/{owner}/packages/container/{package}/versions"
        for version in paginate(path):
            if should_delete(version, pr_tags, protected[image]):
                print(f"deleting {repo}{image} version {version['id']}")
                api(f"{path}/{version['id']}", method="DELETE")


if __name__ == "__main__":
    main()
