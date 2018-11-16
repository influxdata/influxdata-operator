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

The first step is to deploy a pvc backed by a persisten volume where the InfluxDB data will be stored. Next you will deploy one file that will install the Operator, and install the manifest for InfluxDB.

#### Persistent Volumes

The InfluxDB Operator supports the use of Persistent Volumes for each node in
the InfluxDB cluster.

If deploying on GKE clusters see [gcp_storageclass.yaml](deploy/gcp_storageclass.yaml).

If deploying on EKS clusters see [aws_storageclass.yaml](deploy/aws_storageclass.yaml).

If deploying on Local Workstation  see [local_storage.yaml](deploy/local_storage.yaml).


The storage class created by each file supports resize of the persistent volume. 


Note: Resize is only supperted on Kubernetes 1.11 and higher. [Persistent Volume Resize](https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/)


```
kubectl apply -f deploy/gcp-storageclass.yaml
```


#### Deploy InfluxDB Operator & Create InfluxDB

The `bundle.yaml` file contains the manifests needed to properly install the
Operator and InfluxDB.

```
kubectl apply -f bundle.yaml
```

You can watch the list of pods and wait until the Operator pod is in a Running
state, it should not take long.

```
kubectl get pods 
```

You can have a look at the logs for troubleshooting if needed.

```
kubectl logs -l name=influxdata-operator
```
```
kubectl logs -l name=influxdb-0
```

This one file deploys the Operator, Service for InfluxDB, and create the manifest for InfluxDB. 

#### Destroy InfluxDB Cluster

Simply delete the `InfluxDB` Custom Resource to remove the cluster.

```
kubectl delete -f bundle.yaml
```


#### Create "on-demand" Backups & Store it in S3 Bucket .

As the backup files stores in S3 bucket , you need first need to create a Kubernetes Secret for sutenticating to AWS. Deploy the secret custom resource file [aws_creds.yaml](deploy/crds/influxdata_v1alpha1_aws_creds.yaml).

Note : awsAccessKeyId & awsSecretAccessKey are <base64encoded>.


in order to take a backup for database testdb , you need to specify the database name in Backup CR file [backup_cr.yaml](deploy/crds/influxdata_v1alpha1_backup_cr.yaml)


The below yaml file will take a backup for testdb and store it in `s3://influxdb-backup-restore/backup/` `in US-WEST-2 Region`.

```
apiVersion: influxdata.com/v1alpha1
kind: Backup
metadata:
  name: influxdb-backup
spec:
  databases:
    - testdb
  podname: "influxdb-0"
  containername: "influxdb"
  #shard:
  #retention:
  #start:
  #end:
  #since:
  storage:
    s3:
      aws_key_id:
        valueFrom:
          secretKeyRef:
            name: influxdb-backup
            key: awsAccessKeyId
      aws_secret_key:
        valueFrom:
          secretKeyRef:
            name: influxdb-backup
            key: awsSecretAccessKey
      bucket: influxdb-backup-restore 
      folder: backup
      region: us-west-2

```


```
kubectl create -f deploy/crds/influxdata_v1alpha1_backup_cr.yaml
```

You can have a look at the logs for troubleshooting if needed.


```
 kubectl logs influxdata-operator-5c97ffc89d-vc7nk
```

you'll see something like this in logs 2018/11/14 18:17:03 Backups stored to s3://influxdb-backup-restore/backup/20181114181703 


Note: 20181114181703 this is the directory name that stored the backup in S3 bucket . 


#### Use backups to restore a database from S3 Bucket


You need to specify the database that you wants to restore , also there is an option to restore database to new database name .


Ex : the yaml file below will restore the database from s3://influxdb-backup-restore/backup/20181114181703. 


```
apiVersion: influxdata.com/v1alpha1
kind: Restore
metadata:
  name: influxdb-restore
spec:
  database: "testdb"
  restoreTo: "testdb2"
  backupId: "20181116184626"
  podname: "influxdb-0"
  containername: "influxdb"
  #rp:
  storage:
    s3:
      aws_key_id:
        valueFrom:
          secretKeyRef:
            name: influxdb-backup
            key: awsAccessKeyId
      aws_secret_key:
        valueFrom:
          secretKeyRef:
            name: influxdb-backup
            key: awsSecretAccessKey
      bucket: influxdb-backup-restore 
      folder: backup
      region: us-west-2

```



```
kubectl create -f deploy/crds/influxdata_v1alpha1_restore_cr.yaml
```

You can have a look at the logs for troubleshooting if needed.


```
kubectl logs influxdata-operator-5c97ffc89d-vc7nk
```



