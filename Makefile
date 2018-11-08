orgname := aaltameemi
operatorname := influxdb-backup-operator
version := $(shell date +'%Y%m%d%H%M%S')

.PHONY: clean
clean:
	kubectl delete -f deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml

.PHONY: build
build:
	operator-sdk generate k8s
	operator-sdk build $(orgname)/$(operatorname):v$(version)
	docker push $(orgname)/$(operatorname):v$(version)
	@sed -E -i 's/(.*?)$(operatorname):v.*?/\1$(operatorname):v$(version)/g' deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml
	@echo "Version should be $(version)"
	@cat deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml | grep $(operatorname)

.PHONY: deploy
deploy:
	kubectl create -f deploy/crds/influxdata_v1alpha1_influxdb_cr.yaml

.PHONY: test-backup
test-backup:
	kubectl create -f deploy/crds/influxdata_v1alpha1_backup_cr.yaml

.PHONY: test-restore
test-restore:
	kubectl create -f deploy/crds/influxdata_v1alpha1_restore_cr.yaml

