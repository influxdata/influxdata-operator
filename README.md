This repository has been archived and the work has been paused. If/when the
work starts again, it will continue here.

Table of Contents
=================
   * [Development](#development)
   * [InfluxDB Operator](#influxdb-operator)
      * [Overview](#overview)
      * [Usage](#usage)
         * [Persistent Volumes](#persistent-volumes)
         * [Deploy InfluxDB Operator &amp; Create InfluxDB](#deploy-influxdb-operator--create-influxdb)
            * [Destroy InfluxDB Cluster](#destroy-influxdb-cluster)
            * [How to build &amp; deploy your own operator docker image](#how-to-build--deploy-your-own-operator-docker-image)
         * [Backup and Restore in AWS](#backup-and-restore-in-aws)
            * [Create "on-demand" Backups &amp; Store it in S3 Bucket](#create-on-demand-backups--store-it-in-s3-bucket)
            * [Use backups to restore a database from S3 Bucket](#use-backups-to-restore-a-database-from-s3-bucket)
         * [Backup and Restore in GCP](#backup-and-restore-in-gcp)
            * [Create "on-demand" Backups &amp; Store it in a GCS Bucket](#create-on-demand-backups--store-it-in-a-gcs-bucket)
            * [Use backups to restore a database from GCS Bucket](#use-backups-to-restore-a-database-from-gcs-bucket)
         * [Backup and Restore in Azure](#backup-and-restore-in-azure)
            * [Create "on-demand" Backups &amp; Store it in PV](#create-on-demand-backups--store-it-in-pv)
            * [Use backups to restore a database](#use-backups-to-restore-a-database)
            * [Manual Procedure for upsizing the Persistent Volume in Azure](#manual-procedure-for-upsizing-the-persistent-volume-in-azure)
         * [Backup and Restore in OpenShift Container Platform (OCP)](#backup-and-restore-in-openshift-container-platform-ocp)
            * [Overview of Persistent Volume support](#overview-of-persistent-volume-support)
            * [Dynamic provisioning support](#dynamic-provisioning-support)
            * [Expanding of Persistent Volume support in OCP](#expanding-of-persistent-volume-support-in-ocp)
            * [Cluster installation](#cluster-installation)
            * [StorageClass for NFS](#storageclass-for-nfs)
            * [StorageClass for Ceph RBD](#storageclass-for-ceph-rbd)
               * [limitation in Ceph RBD expanding](#limitation-in-ceph-rbd-expanding)
            * [StorageClass for GlusterFS](#storageclass-for-glusterfs)
            * [StorageClass for VSphere](#storageclass-for-vsphere)
            * [Backup PV for OCP](#backup-pv-for-ocp)
            * [Restore PV for OCP](#restore-pv-for-ocp)


# Development

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

The first step is to deploy a pvc backed by a persistent volume where the InfluxDB data will be stored. Next you will deploy one file that will install the Operator, and install the manifest for InfluxDB.

### Persistent Volumes

The InfluxDB Operator supports the use of Persistent Volumes for each node in
the InfluxDB cluster.

If deploying on GKE clusters see [gcp_storageclass.yaml](deploy/gcp_storageclass.yaml).

If deploying on EKS clusters see [aws_storageclass.yaml](deploy/aws_storageclass.yaml).

If deploying on AKS clusters see [azure_storageclass.yaml](deploy/azure_storageclass.yaml).

If deploying on Local Workstation  see [local_storage.yaml](deploy/local_storage.yaml).

Please refer to the Openshift section for the storage on OCP (Openshift Container Platform). There are many types of storage class that OCP supports. 

The storage class created by each file supports resize of the persistent volume. 
Note: Resize is only supported on Kubernetes 1.11 and higher. [Persistent Volume Resize](https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/)

To create a storage class, 
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

#### How to build & deploy your own operator docker image

Assumptions:
* docker push access to your favorite docker registry is configured properly.
* `operator-sdk` 0.0.7+ is installed on your workstation.

If the changes of the operator code are desired, `make deploy orgname=<docker_repo_name>` performs the following steps:
1. Build and push a new operator image with new tag.
2. Update the `influxdata-operator` pod with the new operator image in the current kubernetes cluster associated with the current kubernetes context. `kubectl config current-context` can show the name of current kubernetes context. 

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

There is a [helper script](scripts/create_gcs_sa.sh) that takes cares of creating a service account, granting the admin IAM role for the GCS bucket used for influxdb data, generating key in JSON, as well as outputting in a base64 encoded format.

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
 kubectl logs influxdata-operator-76f9d76c57-r2vpd
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

### Backup and Restore in Azure

#### Create "on-demand" Backups & Store it in PV

The backup CRD stores the backed up files to a PV(Persistent Volume) as an Azure Disk Storage.

In order to take a backup for one database, you need to specify the database name in Backup CR file [backup_cr_pv.yaml](deploy/crds/influxdata_v1alpha1_backup_cr_pv.yaml)

The below CR file will take a backup for "testdb" and store it in a Azure Disk storage `standard-resize`.

To backup all databases leave [databases:] blank.
* Please see [InfluxDB OSS Backup](https://docs.influxdata.com/influxdb/v1.7/administration/backup_and_restore/#backup)
* Please note the `provider` is set to `pv` as shown in the below yaml.

```
apiVersion: influxdata.com/v1alpha1
kind: Backup
metadata:
  name: influxdb-backup
spec:
  podname: "influxdb-0"
  containername: "influxdb"
  # [ -database <db_name> ] Optional: If not specified, all databases are backed up.
  databases:"testdb"
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
    provider: pv
```

```
kubectl create -f deploy/crds/influxdata_v1alpha1_backup_cr_pv.yaml
```

You can have a look at the logs for troubleshooting if needed.

```
 kubectl logs influxdata-operator-64898d58f4-82lg8
```

you'll see something like this in logs `2019/01/26 00:33:26 backing up db=NOAA_water_database rp=autogen shard=2 to /var/lib/influxdb/backup/20190126003326/NOAA_water_database.autogen.00002.00 since 0001-01-01T00:00:00Z`


Note: 20190126003326 this is the directory name that stored the backup. 


#### Use backups to restore a database


You need to specify the database name that you want to restore. If restoring from a multiple db backup, all db will be restored unless a db name is explicitly specified. 

Ex : the yaml file below will restore the "testdb" database from /var/lib//backup/20190126003326. 

* Please see [InfluxDB OSS Restore](https://docs.influxdata.com/influxdb/v1.7/administration/backup_and_restore/#restore).
* Please note the `provider` is set to `pv` as shown in the below yaml.
  

```
  apiVersion: influxdata.com/v1alpha1
kind: Restore
metadata:
  name: influxdb-restore
spec:
  backupId: "20190126003326"
  podname: "influxdb-0"
  containername: "influxdb"
  # [ -database <db_name> ] Optional:  If not specified, all databases will be restored.
  database: testdb
  # [ -newdb <newdb_name> ] Optional: If not specified, then the value for -db is used. 
  restoreTo: 
  # [ -rp <rp_name> ] Optional: Requires that -db is set. If not specified, all retention policies will be used.
  rp:
  # [ -newrp <newrp_name> ] Optional: Requires that -rp is set. If not specified, then the -rp value is used.
  newRp:
  # [ -shard <shard_ID> ] Optional: If specified, then -db and -rp are required.
  shard:
  storage:
    provider: pv
```

```
kubectl create -f deploy/crds/influxdata_v1alpha1_restore_cr_pv.yaml
```

You can have a look at the logs for troubleshooting if needed.

```
kubectl logs influxdata-operator-64898d58f4-82lg8

2019/01/26 01:00:38 Restore DB: , To DB: , Backup key: 20190126003326
2019/01/26 01:00:38 influxd restore -portable /var/lib/influxdb/backup/20190126003326
```

#### Manual Procedure for upsizing the Persistent Volume in Azure 
First of all, please make sure AKS version is *v1.11* up as resizing PV is beta after v1.11, according to https://kubernetes.io/blog/2018/07/12/resizing-persistent-volumes-using-kubernetes/. 

As https://github.com/kubernetes/kubernetes/issues/68427 states,if the PVC is already attached to a VM, resize azure disk PVC would fail, you need to delete the pod to let that azure disk unattached first before upsizing on PV can take place. 
As the Stateful will keep the influxdb pod's minimal size of 0, we can't simply just delete the pod. 
Here is a workaround. 

1. `kubectl edit sts influxdb` so that it uses a fake pvc name such as from `influxdb-data-pvc` to `influxdb-data-pvc-0`.
2. Wait for `influxdb-0` in `pending` state, then `kubectl edit pvc influxdb-data-pvc` to increase PV size via pvc. 
3. Wait for the pv is updated with the new size, then `kubectl edit sts influxdb` and revert the pvc change made in step 1. 
4. `kubectl delete pod -l app=influxdb` will kill the pending pod and create a new influxdb pod that mounts with resized PV.
5. `kubectl get pv,pvc` shows the both pv and pvc are updated with the new size.


### Backup and Restore in OpenShift Container Platform (OCP)

#### Overview of Persistent Volume support
[Managing storage with Persistent Volume](https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/index.html) is a distinct problem from managing compute resources. OpenShift Container Platform uses the Kubernetes persistent volume (PV) framework to allow cluster administrators to provision persistent storage for a cluster. Developers can use persistent volume claims (PVCs) to request PV resources without having specific knowledge of the underlying storage infrastructure.

OpenShift Container Platform supports many types PersistentVolume plug-ins. We highlight the following plugins. Furthermore there are many other plugins along with examples can be found from the OpenShift documentation.

* [NFS](https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/persistent_storage_nfs.html#install-config-persistent-storage-persistent-storage-nfs)
* [Ceph Rados Block Device(RBD)](https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/persistent_storage_ceph_rbd.html#install-config-persistent-storage-persistent-storage-ceph-rbd)
* [GlusterFS](https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/persistent_storage_glusterfs.html#install-config-persistent-storage-persistent-storage-glusterfs)
* [VMWare vSphere volumes](https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/persistent_storage_vsphere.html)


#### Dynamic provisioning support
[Dynamic provisioning and creating storage classes](https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/dynamically_provisioning_pvs.html#install-config-persistent-storage-dynamically-provisioning-pvs). The *StorageClass* resource object describes and classifies storage that can be requested, as well as provides a means for passing parameters for dynamically provisioned storage on demand. Please make sure to add the specified ansible variables during the cluster deployment. Ceph RBD, GlusterFS and VMWare provisioner support dynamic provisioning for OCP v3.11.

In the ansible inventory file (/etc/)
```
[OSEv3:vars]

openshift_master_dynamic_provisioning_enabled=True
```

#### Expanding of Persistent Volume support in OCP

Please note this feature is not turned on in OCP v3.11 by default. Please follow [Enabling Expansion of Persistent Volume](https://access.redhat.com/documentation/en-us/openshift_container_platform/3.11/html/developer_guide/expanding_persistent_volumes) on how to configure OCP to expand PV.  

According to [OCP 3.11 release note](https://docs.openshift.com/container-platform/3.11/release_notes/ocp_3_11_release_notes.html), Block storage volume types such as GCE-PD, AWS-EBS, Azure Disk, Cinder, and Ceph RBD typically require a file system expansion before the additional space of an expanded volume is usable by pods. Kubernetes takes care of this automatically whenever the pod or pods referencing your volume are restarted. Network attached file systems, such as GlusterFS and Azure File, can be expanded without having to restart the referencing pod, as these systems do not require unique file system expansion.


#### Cluster installation
Please make sure you have a valid Red Hat subscription and then follow the steps from [Installing Openshift Container Platform cluster](https://docs.openshift.com/container-platform/3.11/install/index.html).  *openshift-ansible* playbook is the key for installation and configuration of the cluster.

Here is the cluster and OCP version we are using
```
$ oc version
oc v3.11.69
kubernetes v1.11.0+d4cacc0
features: Basic-Auth GSSAPI Kerberos SPNEGO

Server https://ocp-master-1.c.influx-db-operator.internal:8443
openshift v3.11.69
kubernetes v1.11.0+d4cacc0
```

#### StorageClass for NFS
Please see [nfs_storage.yaml](deploy/nfs_storage.yaml). Only static provisioning is supported for NFS. 

```
oc describe pv influxdb-data-pv-nfs
Name:            influxdb-data-pv-nfs
Labels:          <none>
Annotations:     kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"v1","kind":"PersistentVolume","metadata":{"annotations":{},"name":"influxdb-data-pv-nfs","namespace":""},"spec":{"accessModes":["ReadWri...
Finalizers:      [kubernetes.io/pv-protection]
StorageClass:    no-provisioner
Status:          Available
Claim:
Reclaim Policy:  Retain
Access Modes:    RWO
Capacity:        8Gi
Node Affinity:   <none>
Message:
Source:
    Type:      NFS (an NFS mount that lasts the lifetime of a pod)
    Server:    nfs.openshift.example.com
    Path:      /data
    ReadOnly:  false
Events:        <none>
```

#### StorageClass for Ceph RBD 
[Using Ceph RBD for dynamic provisioning](https://docs.openshift.com/container-platform/3.11/install_config/storage_examples/ceph_rbd_dynamic_example.html)
Please see [ceph_storage.yaml](deploy/ceph_storage.yaml).

```
oc describe sc ceph-resize
Name:                  ceph-resize
IsDefaultClass:        No
Annotations:           <none>
Provisioner:           kubernetes.io/rbd
Parameters:            adminId=admin,adminSecretName=ceph-secret,adminSecretNamespace=rook-ceph,fsType=ext4,imageFeatures=layering,imageFormat=2,monitors=172.30.89.136:6789,172.30.241.58:6789,172.30.249.2:6789,pool=kube,userId=kube,userSecretName=ceph-user-secret
AllowVolumeExpansion:  <unset>
MountOptions:          <none>
ReclaimPolicy:         Delete
VolumeBindingMode:     Immediate
```
##### limitation in Ceph RBD expanding
We ran into the known issue [resize pv failed](https://github.com/kubernetes/kubernetes/issues/72393) while expanding Ceph RBD. The bug fix would be part of future OCP releases.


#### StorageClass for GlusterFS
[Using GlusterFS for dynamic provisioning](https://docs.openshift.com/container-platform/3.11/install_config/persistent_storage/dynamically_provisioning_pvs.html#glusterfs)
Please see [glusterfs_storage.yaml](deploy/glusterfs_storage.yaml).


#### StorageClass for VSphere
[Using vSphere for dynamic provisioning](https://kubernetes.io/docs/concepts/storage/storage-classes/#vsphere)
Please see [vsphere_storage.yaml](deploy/vsphere_storage.yaml).


#### Backup PV for OCP
The backup CRD stores the backed up files to a PV(Persistent Volume). 

In order to take a backup for one database, you need to specify the database name in Backup CR file [backup_cr_pv.yaml](deploy/crds/influxdata_v1alpha1_backup_cr_pv.yaml)

The below CR file will backup all databases by leaving [databases:] blank.
* Please see [InfluxDB OSS Backup](https://docs.influxdata.com/influxdb/v1.7/administration/backup_and_restore/#backup)
* Please note the `provider` is set to `pv` as shown in the below yaml.
  
```
apiVersion: influxdata.com/v1alpha1
kind: Backup
metadata:
  name: influxdb-backup
spec:
  podname: "influxdb-0"
  containername: "influxdb"
  # [ -database <db_name> ] Optional: If not specified, all databases are backed up.
  databases:
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
    provider: pv
```

```
kubectl create -f deploy/crds/influxdata_v1alpha1_restore_cr_pv.yaml
```
You can have a look at the logs for troubleshooting if needed.
```
oc logs -f influxdata-operator-6788d5ffc5-m2d7r
.....
2019/02/12 20:39:18 backup complete:
2019/02/12 20:39:18 	/var/lib/influxdb/backup/20190212203917/20190212T203918Z.meta
2019/02/12 20:39:18 	/var/lib/influxdb/backup/20190212203917/20190212T203918Z.s1.tar.gz
2019/02/12 20:39:18 	/var/lib/influxdb/backup/20190212203917/20190212T203918Z.s2.tar.gz
2019/02/12 20:39:18 	/var/lib/influxdb/backup/20190212203917/20190212T203918Z.s3.tar.gz
2019/02/12 20:39:18 	/var/lib/influxdb/backup/20190212203917/20190212T203918Z.s4.tar.gz
2019/02/12 20:39:18 	/var/lib/influxdb/backup/20190212203917/20190212T203918Z.s5.tar.gz
2019/02/12 20:39:18 	/var/lib/influxdb/backup/20190212203917/20190212T203918Z.s6.tar.gz
2019/02/12 20:39:18 	/var/lib/influxdb/backup/20190212203917/20190212T203918Z.manifest
2019/02/12 20:39:18 Done with reconcile!
```

#### Restore PV for OCP

You need to specify the database name that you want to restore. If restoring from a multiple db backup, all db will be restored unless a db name is explicitly specified. 

Ex : the yaml file below restore all the  database from /var/lib//backup/20190212203917.

* Please see [InfluxDB OSS Restore](https://docs.influxdata.com/influxdb/v1.7/administration/backup_and_restore/#restore).
* Please note the `provider` is set to `pv` as shown in the below yaml.

```
apiVersion: influxdata.com/v1alpha1
kind: Restore
metadata:
  name: influxdb-restore
spec:
  backupId: "20190212203917"
  podname: "influxdb-0"
  containername: "influxdb"
  # [ -database <db_name> ] Optional:  If not specified, all databases will be restored.
  database: 
  # [ -newdb <newdb_name> ] Optional: If not specified, then the value for -db is used. 
  restoreTo: 
  # [ -rp <rp_name> ] Optional: Requires that -db is set. If not specified, all retention policies will be used.
  rp:
  # [ -newrp <newrp_name> ] Optional: Requires that -rp is set. If not specified, then the -rp value is used.
  newRp:
  # [ -shard <shard_ID> ] Optional: If specified, then -db and -rp are required.
  shard:
  storage:
    provider: pv
```

You can have a look at the logs for troubleshooting if needed.
```
oc logs -f influxdata-operator-6788d5ffc5-m2d7
2019/02/12 20:49:01 Restore DB: , To DB: , Backup key: 20190212203917
2019/02/12 20:49:01 influxd restore -portable /var/lib/influxdb/backup/20190212203917
```
