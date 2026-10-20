import argparse
import json
from pathlib import Path

import yaml

from payment_data_contracts.contract import Contract, load_contract


def documentation(contract: Contract) -> str:
    from datacontract.data_contract import DataContract

    sections = [
        DataContract(data_contract=contract.model).export("markdown"),
        "",
        "## Contract provenance",
        "",
        f"ODCS: {contract.model.apiVersion}; SHA-256: `{contract.sha256}`.",
        "",
        "## Ownership and governance",
        "",
        "```yaml",
        yaml.safe_dump(
            contract.model.model_dump(mode="json", include={"team", "tags", "customProperties"}, exclude_none=True),
            sort_keys=False,
        ).rstrip(),
        "```",
        "",
        "## Physical schema versions",
        "",
    ]
    for dataset in contract.model.schema_ or []:
        sections.append(f"- `{dataset.name}`: {list(contract.versions(dataset.name))}.")
    sections.extend(
        ["", "Fields with `sinceSchemaVersion` are physically absent before that version.", "", "## Quality rules"]
    )
    for dataset in contract.model.schema_ or []:
        for rule in dataset.quality or []:
            definition = rule.model_dump(mode="json", exclude_none=True)
            sections.extend(
                ["", f"### {rule.id}", "", "```yaml", yaml.safe_dump(definition, sort_keys=False).rstrip(), "```"]
            )
    return "\n".join(sections).rstrip() + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(description="Validate and document the historical payment ODCS contract.")
    parser.add_argument("--contract", type=Path)
    commands = parser.add_subparsers(dest="command", required=True)
    commands.add_parser("lint")
    validate = commands.add_parser("validate")
    validate.add_argument("dataset", type=Path)
    validate.add_argument("--report", type=Path, required=True)
    docs = commands.add_parser("docs")
    docs.add_argument("output", type=Path)
    docs.add_argument("--check", action="store_true")
    args = parser.parse_args()
    try:
        from payment_data_contracts.validation import lint_contract, validate_dataset

        contract = load_contract(args.contract)
        if args.command == "docs":
            lint = lint_contract(contract)
            if not lint.passed:
                print(json.dumps(lint.to_dict(), indent=2))
                return 1
            content = documentation(contract)
            if args.check:
                if not args.output.exists() or args.output.read_text() != content:
                    print(f"Contract documentation is stale: {args.output}")
                    return 1
            else:
                args.output.parent.mkdir(parents=True, exist_ok=True)
                args.output.write_text(content)
            return 0
        report = validate_dataset(args.dataset, contract) if args.command == "validate" else lint_contract(contract)
        payload = report.to_dict()
    except Exception as exc:
        payload = {"report_version": 1, "status": "failed", "checks": [{"result": "error", "reason": str(exc)}]}
    text = json.dumps(payload, indent=2) + "\n"
    if args.command == "validate":
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(text)
    print(text, end="")
    return 0 if payload["status"] == "passed" else 1


if __name__ == "__main__":
    raise SystemExit(main())
