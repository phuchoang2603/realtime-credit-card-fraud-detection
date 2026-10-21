# Payment Gateway and Real-Time Fraud Detection

This project builds an ecommerce payment gateway with fraud decisioning. The planned gateway owns hosted checkout, payment execution and risk decisions; the marketplace keeps ownership of customers, sellers, orders and inventory. Initial processing uses synthetic payment-method tokens and a processor simulator.

The current repository contains a health-only Go HTTP edge and a Python gRPC fraud decision service with a stateless rules baseline, behavior tests, container/CI configuration, Talos GitOps manifests and application telemetry. Gateway services, ecommerce feature pipelines and the coursework extensions are planned. Source/configuration presence does not establish a live deployment or completed coursework.

## Repository structure

```text
.
├── .github/workflows/      # Independent check jobs, image build workflow, wiki publication
├── CONTRIBUTING.md         # Initial setup and wiki link
├── contracts/              # Versioned protobuf service contracts
├── docs/
│   ├── architecture/      # Gateway overview, payment and data/ML flows, conventions, roadmap
│   ├── deployment/        # GitOps, runtime configuration and observability
│   ├── verification/      # Acceptance criteria and verification guidance
│   ├── data-products/     # Source interfaces and data contracts
│   └── research/          # Background, experiments and images
├── infra/
│   ├── argocd/            # Application catalog and prod root
│   ├── charts/            # Application Helm charts
│   └── docker/            # Service Dockerfiles and reusable Go template
├── openspec/              # Capability specifications and change plans
├── src/edge/              # Public HTTP edge and generated gRPC client
├── src/fraud-service/     # gRPC fraud decision service, tests and generated bindings
├── devenv.nix             # Local development environment
├── devenv.yaml            # Devenv inputs
└── devenv.lock            # Locked environment dependencies
```

## Documentation

Browse the [project wiki](https://github.com/phuchoang2603/payment-gateway-fraud-detection/wiki) for architecture, the implementation roadmap, deployment and verification guidance; see [CONTRIBUTING](CONTRIBUTING.md) to work in the repository.
