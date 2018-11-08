# Influxdb Operator

A Kubernetes operator to manage Influxdb instances.

## Overview

This Operator is built using the [Operator SDK](https://github.com/operator-framework/operator-sdk), which is part of the [Operator Framework](https://github.com/operator-framework/) and manages one or more Influxdb instances deployed on Kubernetes.

## Usage

The first step is to deploy a pvc backed by a persisten volume where the Influxdb data will be stored. Next you will deploy one file that will install the Operator, and install the manifest for Influxdb.

#### Persistent Volumes

The Influxdb Operator supports the use of Persistent Volumes for each node in
the Influxdb cluster. If deploying on GKE clusters see [gcp_storage.yaml](deploy/gcp_storage.yaml).
If deploying on EKS clusters see [aws_storage.yaml](deploy/aws_storage.yaml).

```
kubectl apply -f deploy/gcp-storage.yaml
```

When deleting a Influxdb deployment that uses Persistent Volumes, remember to
remove the left-over volumes when the cluster is no longer needed, as these will
not be removed automatically.

```
kubectl delete pvc -l name=influxdb-data-pvc
```

#### Deploy Influxdb Operator & Create Influxdb

The `deploy` directory contains the manifests needed to properly install the
Operator and Influxdb.

```
kubectl apply -f deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml
```

You can watch the list of pods and wait until the Operator pod is in a Running
state, it should not take long.

```
kubectl get pods -wl name=influxdata-operator
```
```
kubectl get pods -wl name=influxdb-0
```

You can have a look at the logs for troubleshooting if needed.

```
kubectl logs -l name=influxdata-operator
```
```
kubectl logs -l name=influxdb-0
```

This one file deploys the Operator, Service for Influxdb, and create the manifest for Influxdb. 

#### Destroy Influxdb Cluster

Simply delete the `Influxdb` Custom Resource to remove the cluster.

```
kubectl delete -f deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml
```

## Development

Clone the repository to a location on your workstation, generally this should be in someplace like `$GOPATH/src/github.com/ORG/REPO`.

Navigate to the location where the repository has been cloned and install the dependencies.

```
cd YOUR_REPO_PATH
dep ensure
```