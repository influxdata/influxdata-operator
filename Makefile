orgname ?= ORGNAME_UNDER_DOCKER_REGISTRY
operatorname := influxdb-operator
version := $(shell date +'%Y%m%d%H%M%S')
PKGS = $(shell go list ./... | grep -v /vendor/)

.PHONY: all test format dep dep-update clean build deploy

all: format test build

format:
	go fmt $(PKGS)

dep:
	dep ensure -v

dep-update:
	dep ensure -update -v

clean:
	kubectl delete -f deploy/crds/influxdata_v1alpha1_backup_cr.yaml || true
	kubectl delete -f deploy/crds/influxdata_v1alpha1_restore_cr.yaml || true
	kubectl delete -f deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml || true

build:
	operator-sdk generate k8s
	operator-sdk build $(orgname)/$(operatorname):v$(version)
	docker push $(orgname)/$(operatorname):v$(version)
	@echo "Version should be $(version)"

deploy: build
	# testing GKE here
	# kubectl apply -f deploy/gcp_storageclass.yaml
	#sed -E 's=REPLACE_IMAGE=$(orgname)/$(operatorname):v$(version)=g' bundle.yaml > .bundel.yaml
	@sed -E 's=REPLACE_IMAGE=$(orgname)/$(operatorname):v$(version)=g' bundle.yaml | kubectl apply -f -

.PHONY: test-backup
test-backup:
	kubectl create -f deploy/crds/influxdata_v1alpha1_backup_cr.yaml

.PHONY: test-restore
test-restore:
	kubectl create -f deploy/crds/influxdata_v1alpha1_restore_cr.yaml

