## Development

Clone the repository to a location on your workstation, generally this should be in someplace like `$GOPATH/src/github.com/`.

Navigate to the location where the repository has been cloned and install the dependencies.

```
cd YOUR_REPO_PATH
dep ensure
```

# InfluxDB Operator

A Kubernetes operator to manage InfluxDB instances.

## Overview

This Operator is built using the [Operator SDK](https://github.com/operator-framework/operator-sdk), which is part of the [Operator Framework](https://github.com/operator-framework/) and manages one or more InfluxDB instances deployed on Kubernetes.

## Usage

<<<<<<< HEAD
The first step is to deploy a pvc backed by a persisten volume where the InfluxDB data will be stored. Next you will deploy one file that will install the Operator, and install the manifest for InfluxDB.

#### Persistent Volumes

The InfluxDB Operator supports the use of Persistent Volumes for each node in
the InfluxDB cluster. If deploying on GKE clusters see [gcp_storage.yaml](deploy/gcp_storage.yaml).
If deploying on EKS clusters see [aws_storage.yaml](deploy/aws_storage.yaml).
The storage class created by each file supports resize of the persistent volume. 
Note: Resize is only supperted on Kubernetes 1.11 and higher. [Persistent Volume Resize](https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/)

```
kubectl apply -f deploy/gcp-storage.yaml
```

When deleting a InfluxDB deployment that uses Persistent Volumes, remember to
remove the left-over volumes when the cluster is no longer needed, as these will
not be removed automatically.

```
kubectl delete pvc -l name=influxdb-data-pvc
```

#### Deploy InfluxDB Operator & Create InfluxDB

The `deploy` directory contains the manifests needed to properly install the
Operator and InfluxDB.

```
kubectl apply -f deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml
=======
The first step is to deploy a pvc backed by a persisten volume where the Influxdb data will be stored. Next you will deploy one file that will install the Operator, and install the manifest for Influxdb.

#### Persistent Volumes

The Influxdb Operator supports the use of Persistent Volumes for each node in
the Influxdb cluster. If deploying on GKE clusters see [gcp_storage.yaml](deploy/gcp_storage.yaml).
If deploying on EKS clusters see [aws_storage.yaml](deploy/aws_storage.yaml).

```
kubectl apply -f deploy/gcp-storage.yaml
>>>>>>> master
```

When deleting a Influxdb deployment that uses Persistent Volumes, remember to
remove the left-over volumes when the cluster is no longer needed, as these will
not be removed automatically.

```
kubectl delete pvc -l name=influxdb-data-pvc
```
```
kubectl get pods -wl name=influxdb-0
```

#### Deploy Influxdb Operator & Create Influxdb

The `deploy` directory contains the manifests needed to properly install the
Operator and Influxdb.

```
kubectl apply -f deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml
```
```
kubectl logs -l name=influxdb-0
```

<<<<<<< HEAD
This one file deploys the Operator, Service for InfluxDB, and create the manifest for InfluxDB. 

#### Destroy InfluxDB Cluster

Simply delete the `InfluxDB` Custom Resource to remove the cluster.

```
kubectl delete -f deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml
```


#### Create "on-demand" Backups

First you need to change the database name field in the yaml file to contain the database name that you wants to backup .
Ex : the yaml file below will backed up the testdb database .

```
apiVersion: influxdata.com/v1alpha1
kind: Backup
metadata:
  name: influxdb-backup
spec:
  # Add fields here
  database: testdb
```


```
kubectl create -f deploy/crds/influxdata_v1alpha1_backup_cr.yaml
```

You can have a look at the logs for troubleshooting if needed.


```
kubectl get pods -wl name=influxdata-operator
=======
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
>>>>>>> master
```


