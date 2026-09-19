# Payment Gateway and Real-Time Fraud Detection

This project is evolving from a terminal-oriented fraud prediction API into an ecommerce payment gateway with fraud decisioning. The planned gateway owns hosted checkout, payment execution and risk decisions; the marketplace keeps ownership of customers, sellers, orders and inventory. Initial processing uses synthetic payment-method tokens and a processor simulator.

The current repository contains the Python fraud API, a bundled model, tests, container/CI configuration, Talos GitOps manifests and application telemetry. Gateway services, ecommerce feature pipelines and the coursework extensions are planned. Source/configuration presence does not establish a live deployment or completed coursework.

## Table of Contents

- [Architecture and roadmap](#architecture-and-roadmap)
- [Repository structure](#repository-structure)
- [Documentation](#documentation)
- [Historical demo and diagram](#historical-demo-and-diagram)

## Architecture and roadmap

The [architecture guide](docs/architecture/payment-gateway.md) separates the existing foundation from the planned Go edge, Accounts, event-sourced Payments and webhook delivery services, with Python fraud, inference, drift and data/training workloads. It includes deployment, checkout/recovery and data/ML diagrams, ownership boundaries and deferred decisions.

The [coursework guide](docs/coursework/README.md) maps required capabilities to implementation tickets and proof. Mini-coursework is the versioned data-platform baseline that final ML extends. Track delivery in the [GitHub project](https://github.com/users/phuchoang2603/projects/3), starting with [PG-01 (#29)](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/29) and [baseline acceptance (#67)](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/issues/67).

## Repository structure

```text
.
├── .github/workflows/      # Fraud-service checks and image publication
├── docs/
│   ├── architecture/      # Planned gateway guide and historical Excalidraw assets
│   ├── coursework/        # Mini/final capability coverage and evidence conventions
│   ├── deployment/        # GitOps, CI, observability and historical screenshots
│   ├── development/       # Local setup
│   └── research/          # Background, experiments and images
├── infra/
│   ├── argocd/            # Application catalog and dev/prod roots
│   ├── charts/            # Fraud-service Helm chart
│   └── docker/            # Service Dockerfile
├── openspec/              # Capability specifications and change plans
├── src/fraud-service/     # Existing API, bundled model, tests and client tool
├── devenv.nix             # Local development environment
├── devenv.yaml            # Devenv inputs
└── devenv.lock            # Locked environment dependencies
```

## Documentation

| Topic | Guide |
| --- | --- |
| Current foundation and target architecture | [Payment gateway](docs/architecture/payment-gateway.md) |
| Coursework evidence and baseline handoff | [Coursework guide](docs/coursework/README.md) |
| Data-platform capabilities | [Mini-coursework](docs/coursework/mini-coursework.md) |
| Feature, training and serving extensions | [Final ML](docs/coursework/final-ml.md) |
| Local development | [Setup](docs/development/local-setup.md) |
| Application deployment | [GitOps](docs/deployment/gitops.md) |
| Checks and image publication | [CI and release](docs/deployment/ci.md) |
| Metrics, logs and traces | [Shared observability](docs/deployment/shared-observability.md) |
| Prior research | [Background](docs/research/ccfd-background.md), [experiments](docs/research/ccfd-experiements-report.md) |

## Historical demo and diagram

The demo and diagram describe earlier fraud-service work; they do not demonstrate the planned gateway or current operational status.

[![Historical demo](https://img.youtube.com/vi/SOBmdxpqs5E/0.jpg)](https://youtu.be/SOBmdxpqs5E)

![Historical fraud architecture](docs/architecture/mlops1-arch.excalidraw.svg)

[Editable Excalidraw source](docs/architecture/mlops1-arch.excalidraw)
