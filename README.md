# Scalable Real-Time Credit Card Fraud Detection System

This repository contains a complete end-to-end system for real-time credit card fraud detection, including data analysis notebooks, a machine learning API, and infrastructure-as-code for cloud deployment. The system is designed with a modern, observable, and scalable architecture.

## Table of Contents

<!--toc:start-->

- [Demo Video](#demo-video)
- [Repository Structure](#repository-structure)
- [System Architecture](#system-architecture)
- [Documentation](#documentation)

<!--toc:end-->

## Demo Video

Watch a brief overview and demo of the system in action:

[![Demo](https://img.youtube.com/vi/SOBmdxpqs5E/0.jpg)](https://youtu.be/SOBmdxpqs5E)

## Repository Structure

```text
.
├── .github/workflows/       # CI and release automation
├── docs/
│   ├── architecture/       # Excalidraw source and SVG diagram
│   ├── deployment/         # GitOps, CI, observability, and screenshots
│   ├── development/        # Local setup
│   └── research/           # Research notes, experiments, and images
├── infra/
│   ├── argocd/             # App-of-apps chart and dev/prod roots
│   ├── charts/             # Fraud-service Helm chart
│   └── docker/             # Service Dockerfile
├── openspec/               # Specifications and change plans
├── src/fraud-service/      # Application, model, tests, and client tool
├── devenv.nix              # Local development environment
├── devenv.yaml             # Devenv inputs
└── devenv.lock             # Locked environment dependencies
```

## System Architecture

![System architecture](docs/architecture/mlops1-arch.excalidraw.svg)

