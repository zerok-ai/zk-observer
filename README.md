# Zerok-Observer
The Zerok Observer is part of the [Zerok System](https://zerok-ai.github.io/helm-charts/), which is a set of tools for observability in Kubernetes clusters. The Zerok System works along with the OpenTelemetry Operator. Check out these docs [add link here] to learn more about how Zerok can benefit you.

The Zerok Observer is responsible for collecting traces from the OpenTelemetry collector and storing them in the badger DB. The observer also provides an API to query the traces stored in the badger DB. The Zerok Observer only stores traces that satisfy the probe conditions defined in the ZerokProbe CRD. The ZerokProbe CRD is managed by the [Zerok Operator](https://github.com/zerok-ai/zk-operator). 

## Prerequisites
Zerok Operator needs to be installed in the cluster in zk-client namespace for the observer to work. You can refer to the [link](https://github.com/zerok-ai/zk-operator) for details about installing the Zerok Operator.

## Get Helm Repositories Info

```console
helm repo add zerok-ai https://zerok-ai.github.io/helm-charts
helm repo update
```

_See [`helm repo`](https://helm.sh/docs/helm/helm_repo/) for command documentation._

## Install Helm Chart

Install the Zerok Observer.
```console
helm install [RELEASE_NAME] zerok-ai/zk-observer
```

_See [helm install](https://helm.sh/docs/helm/helm_install/) for command documentation._

## Uninstall Helm Chart

```console
helm uninstall [RELEASE_NAME]
```

This removes all the Kubernetes components associated with the chart and deletes the release.

_See [helm uninstall](https://helm.sh/docs/helm/helm_uninstall/) for command documentation._

## Upgrading Helm Chart

```console
helm upgrade [RELEASE_NAME] [CHART] --install
```

_See [helm upgrade](https://helm.sh/docs/helm/helm_upgrade/) for command documentation._

### Contributing
Contributions to the Zerok Observer are welcome! Submit bug reports, feature requests, or code contributions.

### Reporting Issues
Encounter an issue? Please file a report on our GitHub issues page with detailed information to aid in quick resolution.