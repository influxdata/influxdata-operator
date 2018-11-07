# Influxdb Operator

A Kubernetes operator to manage Influxdb instances.

## Overview

This Operator is built using the [Operator SDK](https://github.com/operator-framework/operator-sdk), which is part of the [Operator Framework](https://github.com/operator-framework/) and manages one or more Influxdb instances deployed on Kubernetes.

## Usage

The first step is to deploy the Influxdb Operator into the cluster where it
will watch for requests to create `Influxdb` resources, much like the native
Kubernetes Deployment Controller watches for Deployment resource requests.

#### Deploy Influxdb Operator

The `deploy` directory contains the manifests needed to properly install the
Operator.

```
kubectl apply -f deploy
```

You can watch the list of pods and wait until the Operator pod is in a Running
state, it should not take long.

```
kubectl get pods -wl name=influxdata-operator
```

You can have a look at the logs for troubleshooting if needed.

```
kubectl logs -l name=influxdata-operator
```

Once the Influxdb Operator is deployed, Have a look in the `examples` directory for example manifests that create `Influxdb` resources.

#### Create Influxdb Cluster

Once the Operator is deployed and running, we can create an example Influxdb
cluster. The `example` directory contains several example manifests for creating
Influxdb clusters using the Operator.

```
kubectl apply -f example/influxdb-minimal.yaml
```

Watch the list of pods to see that each requested node starts successfully.

```
kubectl get pods -wl cluster=influxdb-minimal-example
```

#### Destroy Influxdb Cluster

Simply delete the `Influxdb` Custom Resource to remove the cluster.

```
kubectl delete -f example/influxdb-minimal.yaml
```

#### Persistent Volumes

The Influxdb Operator supports the use of Persistent Volumes for each node in
the Influxdb cluster. See [influxdb-custom.yaml](example/influxdb-custom.yaml)
for the syntax to enable.

```
kubectl apply -f example/influxdb-custom.yaml
```

When deleting a Influxdb cluster that uses Persistent Volumes, remember to
remove the left-over volumes when the cluster is no longer needed, as these will
not be removed automatically.

```
kubectl delete influxdb,pvc -l cluster=influxdb-custom-example
```

## Development

Clone the repository to a location on your workstation, generally this should be in someplace like `$GOPATH/src/github.com/ORG/REPO`.

Navigate to the location where the repository has been cloned and install the dependencies.

```
cd YOUR_REPO_PATH
dep ensure
```