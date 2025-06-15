# Real-Time Credit Card Fraud Detection System

This repository contains a complete end-to-end system for real-time credit card fraud detection, including data analysis notebooks, a machine learning API, and infrastructure-as-code for cloud deployment. The system is designed with a modern, observable, and scalable architecture.

![mlops1-arch](./deployments/images/mlops1-arch.excalidraw.svg)

## Table of Contents

<!--toc:start-->

- [Real-Time Credit Card Fraud Detection System](#real-time-credit-card-fraud-detection-system)
  - [Table of Contents](#table-of-contents)
  - [📺 Demo Video](#📺-demo-video)
  - [📂 Repository Structure](#📂-repository-structure)
  - [🏛️ System Architecture](#🏛️-system-architecture)
  - [🚀 Installation and Usage](#🚀-installation-and-usage)
    - [🔬 Running the Notebooks](#🔬-running-the-notebooks)
    - [🐳 Local Deployment with Docker Compose](#🐳-local-deployment-with-docker-compose)
    - [☸️ Local Deployment with Kubernetes (on Proxmox)](#️-local-deployment-with-kubernetes-on-proxmox)
    - [☁️ Cloud Deployment on GKE with Terraform & GitOps](#️-cloud-deployment-on-gke-with-terraform-gitops)
  - [🤖 CI/CD Pipeline](#🤖-cicd-pipeline) - [Continuous Integration & Testing](#continuous-integration-testing) - [Continuous Deployment](#continuous-deployment)
  <!--toc:end-->

## 📺 Demo Video

Watch a brief overview and demo of the system in action:

[![Demo](https://img.youtube.com/vi/SOBmdxpqs5E/0.jpg)](https://youtu.be/SOBmdxpqs5E)

## 📂 Repository Structure

The repository is organized into distinct directories, each serving a specific purpose.

```
.
├── app/                  # Contains the FastAPI application source code.
├── client/               # A Python client to simulate requests to the API.
├── config/               # Configuration files for monitoring tools (Alloy, Grafana, Loki, Prometheus).
├── deployments/          # Infrastructure (Terraform) and application (Argo CD, Helm) manifests for GitOps.
├── notebooks/            # Jupyter notebooks for data exploration, model training, and experimentation.
├── tests/                # Unit and integration tests for the application.
├── docker-compose.yaml   # Docker Compose file for local development and testing.
└── pyproject.toml        # Project metadata and dependency management.
```

## 🏛️ System Architecture

The system is designed to run in two primary environments: locally via Docker Compose for development and on Google Kubernetes Engine (GKE) for production, managed via a GitOps workflow.

- **Fraud Detection API**: A FastAPI server that exposes a prediction endpoint. It is instrumented with OpenTelemetry for collecting metrics, logs, and traces.
- **Client Simulator**: A Python script that continuously sends transaction data to the API to simulate real-world traffic.
- **Observability Stack**:
  - **Grafana Alloy**: The collector agent that gathers telemetry data from the API and host environment.
  - **Loki**: The backend for log aggregation and storage.
  - **Tempo**: The backend for distributed trace storage.
  - **Prometheus**: The backend for metrics storage and alerting.
  - **Grafana**: The unified dashboard for visualizing all logs, metrics, and traces.

## 🚀 Installation and Usage

This project can be run locally for development or deployed to a cloud environment.

### 🔬 Running the Notebooks

The `notebooks` directory contains all the research and analysis for this project, covering everything from data exploration to model training and evaluation.

➡️ **To run the Jupyter Lab environment, see the instructions in the [notebooks/README.md](./notebooks/README.md) file.**

### 🐳 Local Deployment with Docker Compose

For a quick and easy local setup, use the provided Docker Compose configuration. This will spin up the API, the client simulator, and the entire observability stack on your local machine.

Simply run the following command from the root of the repository:

```bash
docker compose up --build -d
```

### ☸️ Local Deployment with Kubernetes (on Proxmox)

For those looking to replicate a full cloud-native environment on-premise, this project can be deployed on a Kubernetes cluster running on Proxmox VE. This setup offers a powerful local alternative to GKE for development and testing.

➡️ **For a complete guide on setting up the Kubernetes cluster, see the [kubernetes-proxmox](https://github.com/phuchoang2603/kubernetes-proxmox) repository.**

### ☁️ Cloud Deployment on GKE with Terraform & GitOps

For a production-grade setup, you can provision the infrastructure on Google Kubernetes Engine (GKE) using Terraform and manage all applications via a GitOps workflow with Argo CD.

➡️ **For a complete, end-to-end guide, see the [deployments/README.md](./deployments/README.md) file.**

## 🤖 CI/CD Pipeline

### Continuous Integration & Testing

This repository uses **GitHub Actions** to automate code quality checks and testing. The workflow, defined [here](.github/workflows/lint-test.yml), runs on every push and pull request to ensure the codebase remains clean and functional.

The CI pipeline includes the following stages:

- **Linting**: Code is linted using `ruff` to enforce style consistency and catch common errors.
- **Testing**: Unit tests for the FastAPI application are executed using **Pytest**. This ensures that the core API logic remains correct and reliable. The tests can be found in the `tests/` directory.

### Continuous Deployment

This project automates its release process using a **Continuous Deployment** pipeline powered by GitHub Actions, as defined [here](.github/workflows/release.yml). This workflow prepares new versions of the application for deployment in the GitOps-managed environment.

The CD pipeline is triggered automatically when a new version tag (e.g., `v1.2.3`) is changed in `pyproject.toml` file and pushed to the repository. It then performs the following steps:

1. **Build & Push Images**: It builds multi-platform Docker images for the `api` services.
2. **Push to Registry**: The newly created images are pushed to the Github Artifact Registry (GHCR).
3. **Update Manifests**: The workflow checks out the repository, updates the Helm chart's `values.yaml` file with the new image tag, and commits this change back to the repository.

This final commit triggers Argo CD to automatically detect the change in the Git repository and deploy the new version of the application to the Kubernetes cluster, completing the GitOps cycle.
