FROM alpine:3.6

# install ca-certificates for `google-cloud-go`  
# to work with Google Cloud API such as GCS.
# Otherwise it won't work
RUN apk update && \
   apk add ca-certificates && \
   update-ca-certificates && \
   rm -rf /var/cache/apk/*
USER nobody
ADD build/_output/bin/influxdata-operator /usr/local/bin/influxdata-operator
