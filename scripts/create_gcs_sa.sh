#!/usr/bin/env bash
set -euo pipefail

SA_ID=influxdata-sa
BUCKET=influxdb-backup-restore

GOOGLE_CLOUD_PROJECT=$(gcloud config get-value project)
GCS_SA_EMAIL="${SA_ID}@${GOOGLE_CLOUD_PROJECT}.iam.gserviceaccount.com"
gcloud iam service-accounts create ${SA_ID} --project ${GOOGLE_CLOUD_PROJECT} \
--display-name "Service Account for Influxdata backup and restore"

# grant GCS storage account 
gsutil iam ch serviceAccount:${GCS_SA_EMAIL}:admin gs://${BUCKET}
# create service account key
gcloud iam service-accounts keys create ${SA_ID}-key.json --iam-account=${GCS_SA_EMAIL}

# ouptut the base64 encoded value of the service account key
cat ${SA_ID}-key.json | base64 