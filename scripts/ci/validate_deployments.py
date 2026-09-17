"""CI-only Helm rendering and explicit application resource contract validation.

Validates the supported fields of our Argo/Victoria resources directly; this does
not claim full upstream CRD schema validation or live collector acceptance.
"""

import copy
import json
import re
import subprocess
import tempfile
from pathlib import Path
from urllib.parse import parse_qs, urlsplit

import yaml

ROOT = Path(__file__).resolve().parents[2]
CHART = ROOT / "infra/charts/fraud-service"
CATALOG = ROOT / "infra/argocd/app-of-apps"
OUTPUT = ROOT / "artifacts/helm"
REPO = "https://github.com/phuchoang2603/realtime-credit-card-fraud-detection.git"
NAMESPACE = "payment-gateway"
RELEASE = "fraud-service"
SELECTOR = {"app.kubernetes.io/name": RELEASE, "payment-gateway/release": RELEASE}
VERSIONS = {
    "Application": "argoproj.io/v1alpha1",
    "Deployment": "apps/v1",
    "Service": "v1",
    "ConfigMap": "v1",
    "VMPodScrape": "operator.victoriametrics.com/v1beta1",
    "VMRule": "operator.victoriametrics.com/v1beta1",
}


def run(*args):
    return subprocess.check_output([str(arg) for arg in args], text=True, cwd=ROOT)


def documents(text):
    docs = [item for item in yaml.safe_load_all(text) if item is not None]
    assert docs and all(isinstance(item, dict) for item in docs), (
        "Expected YAML resource objects"
    )
    return docs


def resource(doc, kind, namespace):
    assert doc["kind"] == kind
    assert doc["apiVersion"] == VERSIONS[kind]
    assert doc["metadata"]["namespace"] == namespace
    assert isinstance(doc["metadata"]["name"], str) and doc["metadata"]["name"]
    for value in doc["metadata"].get("labels", {}).values():
        assert isinstance(value, str), "Kubernetes label values must be strings"


def validate_root(root, environment, revision):
    resource(root, "Application", "argo-cd")
    assert root["metadata"]["name"] == f"payment-gateway-{environment}-root"
    spec = root["spec"]
    assert spec["project"] == "default"
    assert spec["destination"] == {
        "server": "https://kubernetes.default.svc",
        "namespace": "argo-cd",
    }
    source = spec["source"]
    assert source["repoURL"] == REPO
    assert source["path"] == "infra/argocd/app-of-apps"
    assert source["targetRevision"] == revision
    values = source["helm"]["valuesObject"]
    assert values["global"]["targetRevision"] == revision
    assert values["namePrefix"] == f"payment-gateway-{environment}"
    assert values["destinationName"] == environment
    return values


def validate_child(child, environment, revision):
    resource(child, "Application", "argo-cd")
    assert child["metadata"]["name"] == f"payment-gateway-{environment}-fraud-service"
    spec = child["spec"]
    assert spec["project"] == "default"
    assert spec["destination"] == {"name": environment, "namespace": NAMESPACE}
    assert "CreateNamespace=true" in spec["syncPolicy"]["syncOptions"]
    sources = spec["sources"]
    for source in sources:
        assert source["repoURL"] == REPO
        assert source["targetRevision"] == revision
    chart = sources[0]
    assert chart["path"] == "infra/charts/fraud-service"
    assert chart["helm"]["releaseName"] == RELEASE
    paths = chart["helm"].get("valueFiles", [])
    expected = (
        ["$values/infra/charts/fraud-service/values-prod.yaml"]
        if environment == "prod"
        else []
    )
    assert paths == expected, f"Wrong {environment} values files: {paths}"
    assert len(sources) == (2 if paths else 1)
    if paths:
        assert sources[1]["ref"] == "values"
    files = []
    for path in paths:
        assert path.startswith("$values/")
        resolved = ROOT / path.removeprefix("$values/")
        assert resolved.is_file()
        files.append(resolved)
    return files


def validate_dashboard(dashboard, identity, datasources):
    assert dashboard["uid"] == "fraud-service"
    panels = dashboard["panels"]
    assert len({panel["id"] for panel in panels}) == len(panels)
    selector = '{namespace="payment-gateway",job="fraud-service"}'
    expected = [
        f"sum by (is_fraud) (rate(predictions_total{selector}[5m]))",
        f"histogram_quantile(0.95, sum by (le) (rate(prediction_latency_seconds_bucket{selector}[5m])))",
        f"sum by (le) (increase(fraud_prediction_score_bucket{selector}[5m]))",
    ]
    for panel, expression in zip(panels[:3], expected, strict=True):
        assert panel["datasource"] == {
            "type": "prometheus",
            "uid": datasources["metrics"],
        }
        assert panel["targets"][0]["expr"] == expression
    log_query = f'kubernetes.pod_namespace:="{NAMESPACE}" AND service:="{identity}"'
    logs_panel = next(panel for panel in panels if panel["type"] == "logs")
    assert logs_panel["datasource"] == {
        "type": "victoriametrics-logs-datasource",
        "uid": datasources["logs"],
    }
    assert logs_panel["targets"][0]["expr"] == log_query
    assert logs_panel["targets"][0]["queryType"] == "instant"
    table = next(panel for panel in panels if panel["type"] == "table")
    assert (
        table["targets"][0]["expr"]
        == log_query + " AND trace_id:* | fields _time, event, trace_id"
    )
    assert table["transformations"] == [
        {
            "id": "extractFields",
            "options": {
                "source": "labels",
                "format": "json",
                "replace": True,
                "keepTime": True,
            },
        }
    ]
    override = table["fieldConfig"]["overrides"][0]
    assert override["matcher"] == {"id": "byName", "options": "trace_id"}
    links = override["properties"][0]["value"]
    assert len(links) == 2
    for link, signal in zip(links, ["traces", "logs"], strict=True):
        panes = json.loads(parse_qs(urlsplit(link["url"]).query)["panes"][0])
        pane = panes["trace"]
        assert pane["datasource"] == datasources[signal]
        query = pane["queries"][0]
        assert query["datasource"]["uid"] == datasources[signal]
        if signal == "traces":
            assert query["query"] == "${__value.raw}"
        else:
            assert query["expr"] == log_query + ' AND trace_id:="${__value.raw}"'


def validate_workload(docs, environment, values):
    # An allowlist also rejects platform operators, CRDs, namespaces, ingress,
    # and extra applications regardless of their names.
    expected_kinds = {"Deployment", "Service", "VMPodScrape", "VMRule", "ConfigMap"}
    assert len(docs) == len(expected_kinds)
    assert {doc["kind"] for doc in docs} == expected_kinds
    by_kind = {doc["kind"]: doc for doc in docs}
    for kind, doc in by_kind.items():
        resource(doc, kind, NAMESPACE)
        assert doc["metadata"]["name"] == (
            RELEASE + "-dashboard" if kind == "ConfigMap" else RELEASE
        )
    deployment = by_kind["Deployment"]["spec"]
    assert deployment["selector"]["matchLabels"] == SELECTOR
    labels = deployment["template"]["metadata"]["labels"]
    assert all(labels[key] == value for key, value in SELECTOR.items())
    assert "app.kubernetes.io/instance" not in SELECTOR
    assert deployment["replicas"] == values["replicaCount"]
    containers = deployment["template"]["spec"]["containers"]
    assert len(containers) == 1
    container = containers[0]
    assert (
        container["image"]
        == values["image"]["repository"] + ":" + values["image"]["tag"]
    )
    assert {port["name"]: port["containerPort"] for port in container["ports"]} == {
        "http": 8000,
        "http-metrics": 8010,
    }
    assert container["resources"] == values["resources"]
    assert container["resources"]["requests"] and container["resources"]["limits"]
    for probe in ["livenessProbe", "readinessProbe"]:
        assert container[probe]["httpGet"] == {"path": "/health", "port": "http"}
        for key in [
            "initialDelaySeconds",
            "periodSeconds",
            "timeoutSeconds",
            "failureThreshold",
        ]:
            assert container[probe][key] == values[probe][key]
    env = {entry["name"]: entry["value"] for entry in container["env"]}
    assert len(env) == len(container["env"]), "Duplicate environment variables"
    assert env["MODEL_PATH"] == "/app/models/model.pkl"
    identity = values["telemetry"]["serviceName"]
    assert env["OTEL_SERVICE_NAME"] == identity
    assert env["OTEL_EXPORTER_OTLP_ENDPOINT"] == values["telemetry"]["traceEndpoint"]
    if environment == "prod":
        assert container["imagePullPolicy"] == "Always"
    service = by_kind["Service"]["spec"]
    assert service["type"] == "ClusterIP"
    assert service["selector"] == SELECTOR
    assert {(port["port"], port["targetPort"]) for port in service["ports"]} == {
        (8000, "http"),
        (8010, "http-metrics"),
    }
    scrape = by_kind["VMPodScrape"]
    assert scrape["metadata"]["labels"] == values["podScrape"]["labels"]
    spec = scrape["spec"]
    assert spec["namespaceSelector"] == {"any": False}
    assert spec["selector"]["matchLabels"] == SELECTOR
    assert spec["jobLabel"] == "app.kubernetes.io/name"
    assert len(spec["podMetricsEndpoints"]) == 1
    endpoint = spec["podMetricsEndpoints"][0]
    assert endpoint["port"] == "http-metrics" and endpoint["path"] == "/metrics"
    assert endpoint["interval"] == values["podScrape"]["interval"]
    assert endpoint["scrapeTimeout"] == values["podScrape"]["scrapeTimeout"]
    assert endpoint["relabelConfigs"] == [
        {"targetLabel": "namespace", "replacement": NAMESPACE}
    ]
    selector = '{namespace="payment-gateway",job="fraud-service"}'
    vmrule = by_kind["VMRule"]
    assert vmrule["metadata"].get("labels", {}) == values["rules"]["labels"]
    groups = vmrule["spec"]["groups"]
    assert len(groups) == 1 and groups[0]["name"] == "fraud-service"
    assert len(groups[0]["rules"]) == 1
    rule = groups[0]["rules"][0]
    assert rule["alert"] == "FraudServiceMetricsUnavailable"
    assert rule["expr"] == f"up{selector} == 0 or absent(up{selector})"
    assert rule["for"] == values["rules"]["unavailableFor"]
    assert re.fullmatch(r"[1-9][0-9]*[smhdw]", rule["for"])
    assert rule["labels"] == {"severity": "warning", "service": identity}
    assert rule["annotations"]["summary"] == "Fraud service metrics are unavailable"
    config = by_kind["ConfigMap"]
    assert config["metadata"]["labels"] == values["dashboards"]["labels"]
    dashboards = [json.loads(value) for value in config["data"].values()]
    assert len({item["uid"] for item in dashboards}) == len(dashboards)
    for dashboard in dashboards:
        validate_dashboard(dashboard, identity, values["dashboards"]["datasources"])
    return groups


def merge(base, overlay):
    result = copy.deepcopy(base)
    for key, value in overlay.items():
        result[key] = (
            merge(result[key], value)
            if isinstance(value, dict) and isinstance(result.get(key), dict)
            else value
        )
    return result


def rejects(check, value):
    try:
        check(value)
    except (AssertionError, KeyError, ValueError, TypeError, yaml.YAMLError):
        return
    raise AssertionError("Validator accepted deliberately invalid configuration")


def negative_checks(root, child, docs, environment, values):
    revision = root["spec"]["source"]["targetRevision"]
    bad = copy.deepcopy(root)
    bad["spec"]["destination"]["namespace"] = "payment-gateway"
    rejects(lambda item: validate_root(item, environment, revision), bad)
    bad = copy.deepcopy(child)
    bad["spec"]["destination"]["name"] = "wrong-cluster"
    rejects(lambda item: validate_child(item, environment, revision), bad)
    for kind, change in [
        ("Service", lambda doc: doc["spec"].update(selector={"app": "unrelated"})),
        (
            "VMPodScrape",
            lambda doc: doc["spec"]["podMetricsEndpoints"][0].update(port="wrong-port"),
        ),
        ("VMRule", lambda doc: doc["spec"]["groups"][0]["rules"][0].update(expr="up")),
        (
            "ConfigMap",
            lambda doc: doc["data"].update({"fraud-service.json": "{invalid-json"}),
        ),
        ("Deployment", lambda doc: doc["metadata"].update(namespace="monitoring")),
    ]:
        bad = copy.deepcopy(docs)
        change(next(doc for doc in bad if doc["kind"] == kind))
        rejects(lambda items: validate_workload(items, environment, values), bad)
    rejects(
        lambda items: validate_workload(items, environment, values),
        docs + [{"kind": "CustomResourceDefinition"}],
    )
    rejects(documents, "invalid: [yaml")


def alert_checks(groups, environment):
    rule_path = OUTPUT / f"{environment}-alerts.yaml"
    rule_path.write_text(yaml.safe_dump({"groups": groups}))
    run("promtool", "check", "rules", rule_path)
    base_labels = {
        "job": RELEASE,
        "namespace": NAMESPACE,
        "severity": "warning",
        "service": "fraud-service",
    }
    annotation = {"summary": "Fraud service metrics are unavailable"}

    def expected(labels):
        return {"exp_labels": labels, "exp_annotations": annotation}

    def case(series, alerts, time="10m"):
        return {
            "interval": "1m",
            "input_series": series,
            "alert_rule_test": [
                {
                    "eval_time": time,
                    "alertname": "FraudServiceMetricsUnavailable",
                    "exp_alerts": alerts,
                }
            ],
        }

    own = 'up{namespace="payment-gateway",job="fraud-service",instance="pod:8010"}'
    cases = [
        case([{"series": own, "values": "1+0x15"}], []),
        case(
            [{"series": own, "values": "0+0x15"}],
            [expected({**base_labels, "instance": "pod:8010"})],
        ),
        case([], [expected(base_labels)]),
        case([{"series": own, "values": "0+0x15"}], [], "9m"),
        case(
            [
                {"series": own, "values": "1+0x15"},
                {
                    "series": 'up{namespace="ecommerce",job="fraud-service"}',
                    "values": "0+0x15",
                },
            ],
            [],
        ),
        case(
            [
                {
                    "series": 'up{namespace="ecommerce",job="fraud-service"}',
                    "values": "1+0x15",
                }
            ],
            [expected(base_labels)],
        ),
    ]
    path = OUTPUT / f"{environment}-alert-tests.yaml"
    path.write_text(
        yaml.safe_dump(
            {
                "rule_files": [str(rule_path)],
                "evaluation_interval": "1m",
                "tests": cases,
            }
        )
    )
    print(run("promtool", "test", "rules", path))


def render(environment, branch=False):
    root_text = (ROOT / f"infra/argocd/{environment}/root.yaml").read_text()
    if branch:
        # Editing the anchor changes both revisions, just as documented for dev.
        root_text = re.sub(
            r"targetRevision: &revision \S+",
            "targetRevision: &revision ci-review-branch",
            root_text,
        )
    roots = documents(root_text)
    assert len(roots) == 1
    root = roots[0]
    revision = root["spec"]["source"]["targetRevision"]
    assert isinstance(revision, str) and revision
    if branch:
        assert revision == "ci-review-branch"
    values = validate_root(root, environment, revision)
    suffix = f"{environment}-branch" if branch else environment
    with tempfile.TemporaryDirectory() as directory:
        catalog_values = Path(directory) / "catalog.yaml"
        catalog_values.write_text(yaml.safe_dump(values))
        print(run("helm", "lint", CATALOG, "-f", catalog_values))
        children = documents(
            run(
                "helm",
                "template",
                f"payment-gateway-{environment}",
                CATALOG,
                "-f",
                catalog_values,
            )
        )
        assert len(children) == 1
        child = children[0]
        files = validate_child(child, environment, revision)
        OUTPUT.joinpath(f"{suffix}-applications.yaml").write_text(
            yaml.safe_dump_all([root, child])
        )
        flags = [arg for file in files for arg in ("-f", str(file))]
        print(run("helm", "lint", CHART, *flags))
        text = run("helm", "template", RELEASE, CHART, "--namespace", NAMESPACE, *flags)
        docs = documents(text)
        OUTPUT.joinpath(f"{suffix}-workload.yaml").write_text(text)
        chart_values = yaml.safe_load(CHART.joinpath("values.yaml").read_text())
        for file in files:
            chart_values = merge(chart_values, yaml.safe_load(file.read_text()))
        groups = validate_workload(docs, environment, chart_values)
        if not branch:
            negative_checks(root, child, docs, environment, chart_values)
            alert_checks(groups, environment)
        else:
            # Exercise configurable platform integration and service identity.
            custom = {
                "telemetry": {
                    "serviceName": "custom-fraud",
                    "traceEndpoint": "custom-traces:4317",
                },
                "podScrape": {"labels": {"owner": "custom"}},
                "rules": {"labels": {"owner": "custom"}, "unavailableFor": "2m"},
                "dashboards": {
                    "labels": {"owner": "custom"},
                    "datasources": {
                        "metrics": "custom-metrics",
                        "logs": "custom-logs",
                        "traces": "custom-traces",
                    },
                },
            }
            overrides = Path(directory) / "custom.yaml"
            overrides.write_text(yaml.safe_dump(custom))
            custom_docs = documents(
                run(
                    "helm",
                    "template",
                    RELEASE,
                    CHART,
                    "--namespace",
                    NAMESPACE,
                    *flags,
                    "-f",
                    overrides,
                )
            )
            validate_workload(custom_docs, environment, merge(chart_values, custom))
    return {root["metadata"]["name"], child["metadata"]["name"]}


def main():
    OUTPUT.mkdir(parents=True, exist_ok=True)
    dev_names = render("dev")
    prod_names = render("prod")
    assert dev_names.isdisjoint(prod_names)
    render("dev", branch=True)
    release = yaml.safe_load(ROOT.joinpath(".github/workflows/release.yml").read_text())
    destinations = release["jobs"]["update-helm-values"]["env"]
    for key in ("CHART_FILE", "VALUE_FILE"):
        assert (ROOT / destinations[key]).is_file()
    assert not yaml.safe_load(CHART.joinpath("Chart.yaml").read_text()).get(
        "dependencies"
    )
    print(
        "Validated dev/prod roots, child Applications, workloads, dashboards, alerts, overrides, and rejection cases."
    )


if __name__ == "__main__":
    main()
