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

### Persistent Volumes

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

### Deploy InfluxDB Operator & Create InfluxDB

The `bundle.yaml` file contains the manifests needed to properly install the
Operator and InfluxDB. Please substitute `REPLACE_IMAGE` in `bundle.yaml` with the operator docker image url, 
then run the following command,

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

### Backup and Restore in AWS

#### Create "on-demand" Backups & Store it in S3 Bucket

The backup CRD stores the backed up files to an S3 bucket. You first need to create a Kubernetes Secret for authenticating to AWS. Deploy the secret custom resource file [aws_creds.yaml](deploy/crds/influxdata_v1alpha1_aws_creds.yaml).

Note : awsAccessKeyId & awsSecretAccessKey are `base64encoded`.

```
apiVersion: v1
kind: Secret
metadata:
    name: influxdb-backup-s3
type: Opaque
data:
    awsAccessKeyId: <base64encoded> 
    awsSecretAccessKey: <base64encoded>
```

```
kubectl create -f deploy/crds/influxdata_v1alpha1_aws_creds.yaml
```


In order to take a backup for one database, you need to specify the database name in Backup CR file [backup_cr.yaml](deploy/crds/influxdata_v1alpha1_backup_cr.yaml)

The below CR file will take a backup for "testdb" and store it in `s3://influxdb-backup-restore/backup/` `in US-WEST-2 Region`.

To backup all databases leave [databases:] blank.

* Please see [InfluxDB OSS Backup](https://docs.influxdata.com/influxdb/v1.7/administration/backup_and_restore/#backup)
* Please note the `provider` is set to `s3` as shown in the below yaml.
```
apiVersion: influxdata.com/v1alpha1
kind: Backup
metadata:
  name: influxdb-backup
spec:
  podname: "influxdb-0"
  containername: "influxdb"
  # [ -database <db_name> ] Optional: If not specified, all databases are backed up.
  databases: "testdb"
  # [ -shard <ID> ] Optional: If specified, then -retention <name> is required.
  shard:
  # [ -retention <rp_name> ] Optional: If not specified, the default is to use all retention policies. If specified, then -database is required.
  retention:
  # [ -start <timestamp> ] Optional: Not compatible with -since.
  start:
  # [ -end <timestamp> ] Optional:  Not compatible with -since. If used without -start, all data will be backed up starting from 1970-01-01.
  end:
  # [ -since <timestamp> ] Optional: Use -start instead, unless needed for legacy backup support.
  since:
  storage:
    provider: s3
    s3:
      aws_key_id:
        valueFrom:
          secretKeyRef:
            name: influxdb-backup-s3
            key: awsAccessKeyId
      aws_secret_key:
        valueFrom:
          secretKeyRef:
            name: influxdb-backup-s3
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

You need to specify the database name that you want to restore. If restoring from a multiple db backup, all db will be restored unless a db name is explicitly specified. 

Ex : the yaml file below will restore the "testdb" database from s3://influxdb-backup-restore/backup/20181114181703. 

* Please see [InfluxDB OSS Restore](https://docs.influxdata.com/influxdb/v1.7/administration/backup_and_restore/#restore).
* Please note the `provider` is set to `s3` as shown in the below yaml.
  
```
apiVersion: influxdata.com/v1alpha1
kind: Restore
metadata:
  name: influxdb-restore
spec:
  backupId: "20181119213530"
  podname: "influxdb-0"
  containername: "influxdb"
  # [ -database <db_name> ] Optional:  If not specified, all databases will be restored.
  database: "testdb"
  # [ -newdb <newdb_name> ] Optional: If not specified, then the value for -db is used. 
  restoreTo: 
  # [ -rp <rp_name> ] Optional: Requires that -db is set. If not specified, all retention policies will be used.
  rp:
  # [ -newrp <newrp_name> ] Optional: Requires that -rp is set. If not specified, then the -rp value is used.
  newRp:
  # [ -shard <shard_ID> ] Optional: If specified, then -db and -rp are required.
  shard:
  storage:
    provider: s3
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


### Backup and Restore in GCP

#### Create "on-demand" Backups & Store it in a GCS Bucket

The backup CRD stores the backed up files to a GCS bucket. You first need to create a Kubernetes Secret for authenticating to GCP. Deploy the secret custom resource file [gcp_sa.yaml](deploy/crds/influxdata_v1alpha1_gcp_sa.yaml).

Note : the gcp service account is `base64encoded`.

There is a [helper script](scripts/create_gcs_sa.sh) that takes cares of creating a service account, granting the admin IAM role for the GCS bucket used for influxdb data, generating key in JSON, as well as outputing in a base64 encoded format.

```
apiVersion: v1
kind: Secret
metadata:
  name: influxdata-backup-gcs
type: Opaque
data:
  sa: <base64encoded>
```

```
kubectl create -f deploy/crds/influxdata_v1alpha1_gcp_sa.yaml
```

In order to take a backup for one database, you need to specify the database name in Backup CR file [backup_cr.yaml](deploy/crds/influxdata_v1alpha1_backup_cr.yaml)

The below CR file will take a backup for "testdb" and store it in `s3://influxdb-backup-restore/backup/` `in US-WEST-2 Region`.

To backup all databases leave [databases:] blank.
* Please see [InfluxDB OSS Backup](https://docs.influxdata.com/influxdb/v1.7/administration/backup_and_restore/#backup)
* Please note the `provider` is set to `gcs` as shown in the below yaml.


```
apiVersion: influxdata.com/v1alpha1
kind: Backup
metadata:
  name: influxdb-backup
spec:
  podname: "influxdb-0"
  containername: "influxdb"
  # [ -database <db_name> ] Optional: If not specified, all databases are backed up.
  databases: "testdb"
  # [ -shard <ID> ] Optional: If specified, then -retention <name> is required.
  shard:
  # [ -retention <rp_name> ] Optional: If not specified, the default is to use all retention policies. If specified, then -database is required.
  retention:
  # [ -start <timestamp> ] Optional: Not compatible with -since.
  start:
  # [ -end <timestamp> ] Optional:  Not compatible with -since. If used without -start, all data will be backed up starting from 1970-01-01.
  end:
  # [ -since <timestamp> ] Optional: Use -start instead, unless needed for legacy backup support.
  since:
  storage:
    provider: gcs
    gcs:
      sa_json:
        valueFrom:
          secretKeyRef:
            name: influxdb-backup-gcs
            key: sa
      bucket: influxdb-backup-restore
      folder: backup
```


```
kubectl create -f deploy/crds/influxdata_v1alpha1_backup_cr.yaml
```

You can have a look at the logs for troubleshooting if needed.
```
 k logs influxdata-operator-76f9d76c57-r2vpd
```

you'll see something like this in logs 2018/11/14 18:17:03 Backups stored to gs://influxdb-backup-restore/backup/20190105223039


Note: 20190105223039 this is the directory name that stored the backup in GCS bucket . 


#### Use backups to restore a database from GCS Bucket

You need to specify the database name that you want to restore. If restoring from a multiple db backup, all db will be restored unless a db name is explicitly specified. 

Ex : the yaml file below will restore the "testdb" database from gs://influxdb-backup-restore/backup/20190105223039. 

* Please see [InfluxDB OSS Restore](https://docs.influxdata.com/influxdb/v1.7/administration/backup_and_restore/#restore).
* Please note the `provider` is set to `gcs` as shown in the below yaml.
  

```
apiVersion: influxdata.com/v1alpha1
kind: Restore
metadata:
  name: influxdb-restore
spec:
  backupId: "20190105223039"
  podname: "influxdb-0"
  containername: "influxdb"
  # [ -database <db_name> ] Optional:  If not specified, all databases will be restored.
  database: "testdb"
  # [ -newdb <newdb_name> ] Optional: If not specified, then the value for -db is used. 
  restoreTo: 
  # [ -rp <rp_name> ] Optional: Requires that -db is set. If not specified, all retention policies will be used.
  rp:
  # [ -newrp <newrp_name> ] Optional: Requires that -rp is set. If not specified, then the -rp value is used.
  newRp:
  # [ -shard <shard_ID> ] Optional: If specified, then -db and -rp are required.
  shard:
  storage:
    provider: gcs
    gcs:
      sa_json:
        valueFrom:
          secretKeyRef:
            name: influxdb-backup-gcs
            key: sa
      bucket: influxdb-backup-restore
      folder: backup

```

```
kubectl create -f deploy/crds/influxdata_v1alpha1_restore_cr.yaml
```

You can have a look at the logs for troubleshooting if needed.


```
kubectl logs influxdata-operator-76f9d76c57-r2vpd
```

