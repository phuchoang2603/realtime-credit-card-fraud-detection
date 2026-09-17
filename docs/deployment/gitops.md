# GitOps deployment

The fraud service is deployed to shared Talos `dev` and `prod` clusters through Argo CD. The marketplace repository owns cluster infrastructure and Victoria operators/backends; this repository owns payment-gateway application resources.

Argo roots are [`infra/argocd/dev/root.yaml`](../../infra/argocd/dev/root.yaml) and [`infra/argocd/prod/root.yaml`](../../infra/argocd/prod/root.yaml). They create environment-prefixed child Applications targeting namespace `payment-gateway`.

The service is an internal `ClusterIP` exposing ports 8000 and 8010. Inspect it with:

```bash
kubectl -n payment-gateway port-forward svc/fraud-service 8000:8000 8010:8010
```

See [shared observability](shared-observability.md) for telemetry dependencies and [CI and release](ci.md) for automation.

## Argo CD

![Argo CD application dashboard](images/argocd.png)
