# Historical source verification

The source generator and its production integration are delivered by stacked [PR #75](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/pull/75) and [PR #76](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/pull/76), above the ODCS foundation in [PR #77](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/pull/77). Platform [PR #25](https://github.com/phuchoang2603/talos-proxmox/pull/25) extends the existing workflow controller watch list and enables the installed CNI for prod sidecar injection. Acceptance applies to the combined stack.

## Configuration and provenance

The [configuration](../../src/simulator/configs/historical-v1.yaml) selects dataset `v2`, seed `2603`, nine purchase dates starting `2026-10-12`, 3,000 customers, 40 merchants, and a physical schema cutover on `2026-10-16`. The filename is a configuration name; `dataset_version` controls publication. Customer category preferences are omitted; category skew belongs to merchants. Scenario activations now retain victims and contexts through reconstructable compromise windows.

The accepted output is `s3://synthetic-source/historical/v2/` at `https://minio.home.phuchoang.sbs`. It is immutable Parquet source outside bronze, separate from the pipeline artifact bucket and future Iceberg tables. A last-written `manifest.json` identifies a completed version; `validation.json` contains the native ODCS results.

The simulator image is `ghcr.io/phuchoang2603/realtime-credit-card-fraud-detection/simulator@sha256:e38d8ce5692a04de3011f672bedc52ab8bcad0fc9e4dab938f87a3ca788cecaf`, published by [run 39530678113](https://github.com/phuchoang2603/realtime-credit-card-fraud-detection/actions/runs/39530678113) from revision `dbee944`. The Notebook uses `ghcr.io/kubeflow/kubeflow/notebook-servers/jupyter-scipy@sha256:8f8a000b7eece82c7e874924dc45a3922e9197137eefe67f55c711cc3ce068fd` and installs pinned analysis dependencies recorded in its executed output.

## Observed results

Production Workflow [`historical-v2-vqccb`](historical-data/workflow.json), UID `f6322680-58ef-4ca4-acd5-0e88764dcc09`, succeeded on `2026-10-21` from `02:05:25Z` to `02:07:01Z`. Its generator container completed in 36 seconds. [Earlier failures](historical-data/attempts.json) remain recorded; they were not accepted datasets.

The [manifest](historical-data/manifest.json) SHA-256 is `5b45cbbafcc06b8e1cea23ff968cde8a219f9baa534afbedf7c9a0cae2b8cd3d`; configuration hash is `7d845714b0b162746c3afaaef696d037335058b7070fc14368ab1dbfd6cff160`. All 233 native checks passed in the [validation report](historical-data/validation.json). History ends at `2026-10-21T00:38:00Z`.

| Measurement | Observed |
| --- | --- |
| Delivered events / identical duplicates | 71,007 / 1,420 (1.9998%) |
| Purchases / chargebacks | 13,421 / 791 |
| Event files / label files | 10 / 30, all Zstandard |
| Event bytes / label bytes | 8,816,742 / 814,853 |
| Distinct customers / merchants | 2,967 / 40 |
| Distinct device / network fingerprints | 4,132 / 1,694 |
| Largest merchant share | 23.52% |
| Single-attempt AP / ROC-AUC | 0.4056 / 0.7543 |
| Earlier-history AP / ROC-AUC | 0.6435 / 0.8588 |
| AP improvement | 0.2379 |

The [machine-readable Notebook results](historical-data/sanity.json) include all eight fraud scenarios and the shared identity's three expected `AccessDenied` results. The temporal split trains on the first six purchase dates and evaluates the remaining dates. History improves average precision without making the single-attempt baseline near-perfect. Labels beyond the history end represent simulated delayed availability, not features available at decision time.

The [Notebook deployment capture](historical-data/notebook.json) records its pinned image, pod identity and bound 5 GiB PVC. Authenticated JupyterLab returned the executed output; an anonymous request returned HTTP 302 to sign-in. The PVC retains the executed notebook and source cache across Notebook recreation.

## Reproduction

Enter `devenv shell` in the combined stack. Reconcile the platform prerequisite, shared ClusterSecretStore, workspace and [historical generator chart](../../infra/charts/historical-generator/values.yaml). The only production workflow controller must watch `fraud-data-prod`; do not install another controller. Both namespace-local `historical-source` Secrets reference the same Doppler `prd` keys. Compare their contents in-process without printing credentials and check ExternalSecret readiness.

Use a new dataset version for a new accepted run. Submit the finite template:

```bash
kubectl --kubeconfig ~/.kube/talos-prod.yaml create -f - <<'YAML'
apiVersion: argoproj.io/v1alpha1
kind: Workflow
metadata:
  generateName: historical-source-
  namespace: fraud-data-prod
spec:
  workflowTemplateRef:
    name: historical-generator
YAML
kubectl --kubeconfig ~/.kube/talos-prod.yaml get workflows,pods -n fraud-data-prod
kubectl --kubeconfig ~/.kube/talos-prod.yaml logs -n fraud-data-prod <pod> -c main
```

The chart bounds concurrency, deadline, CPU, memory and temporary disk; the workflow service account can only create/patch executor task results. It restricts both the generator and injected Argo containers. S3 publication copies files directly, avoiding directory-marker writes that RustFS rejects under the prefix-scoped policy. The [immutable-version probe](historical-data/immutable-guard.txt), workflow `historical-v2-guard-d8qd2`, exited 2 because the version was already complete; a subsequent source read confirmed the same manifest SHA-256.

Open the Notebook through the authenticated [Kubeflow route](https://kubeflow.phuchoang.sbs/notebook/fraud-ml-prod/historical-sanity/lab). Its sidecar enforces the Profile's user-header authorization rules; prod Istio uses its existing CNI to avoid a privileged init container. No authentication bypass or public Notebook endpoint is used. Copy the versioned notebook into its persistent workspace and execute it there:

```bash
kubectl --kubeconfig ~/.kube/talos-prod.yaml cp \
  src/simulator/notebooks/historical-sanity.ipynb \
  fraud-ml-prod/historical-sanity-0:/home/jovyan/historical-sanity.ipynb -c historical-sanity
kubectl --kubeconfig ~/.kube/talos-prod.yaml exec -n fraud-ml-prod historical-sanity-0 -c historical-sanity -- \
  jupyter nbconvert --to notebook --execute /home/jovyan/historical-sanity.ipynb \
  --output historical-sanity.executed.ipynb --ExecutePreprocessor.timeout=600
```

The [executed notebook](../../src/simulator/notebooks/historical-sanity.ipynb) verifies physical rows and schema cutover, content-identical duplicates, skew, approximate fingerprint cardinalities, delayed labels and scenario rates. Both learnability models use the same classifier and checkout facts, including merchant category; history predictors use strictly earlier timestamps. Simulation truth is only the target. This source diagnostic does not satisfy the later governed features and delayed-label training acceptance in #53.

![Authenticated Notebook learnability results](historical-data/notebook-learnability.png)

The capture shows the temporal holdout metrics and the passed history-improvement check inside the authenticated production Notebook.

![Source permission checks](historical-data/notebook-permissions.png)

The capture shows denied deletion, out-of-prefix writing and artifact-bucket access, followed by the successful sanity result.

## Access, failures and rotation

The [tracked source policy](../../infra/policies/synthetic-historical-writer.json) allows bucket listing and object read/write/multipart operations under `historical/*`, without deletion or artifact-bucket access. The generator and Notebook deliberately share that write-capable identity. Denial probes target a never-created source key, an out-of-prefix write and artifact-bucket listing; accepted objects are untouched.

Rotate the RustFS source key pair using operator credentials, update `HISTORICAL_RUSTFS_ACCESS_KEY` and `HISTORICAL_RUSTFS_SECRET_KEY` together in Doppler project `realtime-credit-card-fraud-detection`, config `prd`, wait for both ExternalSecrets to refresh, and restart the Notebook and future workflow pods because environment variables do not refresh in existing processes. Verify scoped read/write before revoking the old key. Rotate `ESO_DOPPLER_TOKEN` by replacing `fraud-ml-prod/fraud-doppler-token` key `dopplerToken`, then verify ClusterSecretStore and existing artifact/database secrets remain Ready. Never commit or display key values.

A version without a manifest is incomplete and must not be consumed. The CLI refuses nonempty incomplete prefixes too. Inspect workflow logs and the prefix, then choose a new version or use elevated operator credentials for deliberate cleanup; the source identity cannot delete. Never remove or overwrite an accepted manifest to retry. Record failed attempts alongside successful evidence.

The production root Applications were temporarily switched to the feature branches for this verification. [Restoration verification](historical-data/restoration.json) confirms both original specs are restored, both roots are Healthy/Synced on `main`, and no Application tracks either temporary branch. The temporary Notebook, historical generator Application and data namespace were removed. The bound workspace PVC and immutable external dataset remain. Its Argo tracking annotation was removed so the retained workspace does not keep the existing workspace Application out of sync. Git contains the accepted notebook and evidence; redeploy the merged stack to reopen the retained PVC.
