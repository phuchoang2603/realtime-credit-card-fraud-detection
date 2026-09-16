# Scalable Real-Time Credit Card Fraud Detection System

This repository contains a complete end-to-end system for real-time credit card fraud detection, including data analysis notebooks, a machine learning API, and infrastructure-as-code for cloud deployment. The system is designed with a modern, observable, and scalable architecture.

![mlops1-arch](./docs/images/mlops1-arch.excalidraw.svg)

## Table of Contents

<!--toc:start-->

- [Scalable Real-Time Credit Card Fraud Detection System](#scalable-real-time-credit-card-fraud-detection-system)
  - [Table of Contents](#table-of-contents)
  - [Demo Video](#demo-video)
  - [Repository Structure](#repository-structure)
  - [System Architecture](#system-architecture)
  - [Installation and Usage](#installation-and-usage)
    - [Local development](#local-development)
    - [Local Deployment with Kubernetes (on Proxmox)](#local-deployment-with-kubernetes-on-proxmox)
    - [Cloud Deployment on GKE with Terraform & GitOps](#cloud-deployment-on-gke-with-terraform-gitops)
  - [CI/CD Pipeline](#cicd-pipeline) - [Continuous Integration & Testing](#continuous-integration-testing) - [Continuous Deployment](#continuous-deployment)
  <!--toc:end-->

## Demo Video

Watch a brief overview and demo of the system in action:

[![Demo](https://img.youtube.com/vi/SOBmdxpqs5E/0.jpg)](https://youtu.be/SOBmdxpqs5E)

## Repository Structure

The repository is organized into distinct directories, each serving a specific purpose.

```
.
├── src/fraud-service/    # Fraud service application, model, and tests.
├── deployments/          # Existing Kubernetes and Terraform manifests.
├── infra/                # Container and deployment configuration.
└── devenv.nix            # Local development environment.
```

## System Architecture

The system runs as a service under `src/fraud-service`; deployment remains managed through the existing Kubernetes/GitOps workflow.

- **Fraud Detection API**: A FastAPI server that exposes a prediction endpoint. It is instrumented with OpenTelemetry for collecting metrics, logs, and traces.
- **Client Simulator**: A Python script that continuously sends transaction data to the API to simulate real-world traffic.
- **Observability Stack**:
  - **Alloy**: The collector agent that gathers telemetry data from the API and host environment.
  - **Loki**: The backend for log aggregation and storage.
  - **Tempo**: The backend for distributed trace storage.
  - **Prometheus**: The backend for metrics storage and alerting.
  - **Grafana**: The unified dashboard for visualizing all logs, metrics, and traces.
- **Traefik**: The Ingress Controller managing external access to services.
- **Cert-Manager**: Provides automatic TLS certificate provisioning.

## Installation and Usage

This project can be run locally for development or deployed to a cloud environment.

### Local development

Enter the Python 3.14 development shell and install locked development dependencies:

```bash
devenv shell
cd src/fraud-service
uv sync --locked --dev
```

Linting, tests, and Helm validation run in GitHub Actions. See [local setup](docs/development/local-setup.md) for exact check commands and the [service README](src/fraud-service/README.md) for API, manual client, and Docker commands. Compose has been retired.

### Local Deployment with Kubernetes (on Proxmox)

For those looking to replicate a full cloud-native environment on-premise, this project can be deployed on a Kubernetes cluster running on Proxmox VE. This setup offers a powerful local alternative to GKE for development and testing.

**For a complete guide on setting up the Kubernetes cluster, see the [kubernetes-proxmox](https://github.com/phuchoang2603/kubernetes-proxmox) repository.**

### Cloud Deployment on GKE with Terraform & GitOps

For a production-grade setup, you can provision the infrastructure on Google Kubernetes Engine (GKE) using Terraform and manage all applications via a GitOps workflow with Argo CD.

**For a complete, end-to-end guide, see the instructions in the [deployments](./docs/deployment.md) folder.**

## CI/CD Pipeline

### Continuous Integration & Testing

This repository uses **GitHub Actions** to automate code quality checks and testing. The workflow, defined [here](.github/workflows/ci.yml), runs on every pull request and push to `main`.

The CI pipeline includes the following stages:

- **Linting**: Code is linted using `ruff` to enforce style consistency and catch common errors.
- **Testing**: Unit tests for the FastAPI application live in `src/fraud-service/tests/` and run with `uv run --locked pytest`.

### Continuous Deployment

This project automates its release process using a **Continuous Deployment** pipeline powered by GitHub Actions, as defined [here](.github/workflows/release.yml). This workflow prepares new versions of the application for deployment in the GitOps-managed environment.

The existing release workflow watches pull requests changing `src/fraud-service/pyproject.toml`. When its version check detects a version change, it builds the Python 3.14 image using `infra/docker/fraud-service.Dockerfile` and the `src/fraud-service` context, publishes the existing GHCR version tag, and updates the existing Helm chart values.

See [CI and release details](docs/deployment/ci.md). Kubernetes/GKE and Argo CD documentation describes the legacy infrastructure; this tooling migration does not change that deployment setup.
