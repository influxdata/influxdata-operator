# Influxdata-Operator

The Influxdata Operator creates, configures and manages Influxdb OSS running on Kubernetes.

InfluxData delivers a complete Time Series Platform built specifically for metrics, events, and other time-based data — a modern time-series 
platform. Whether the data comes from humans, sensors, or machines, InfluxData empowers developers to build next-generation monitoring, 
analytics, and IoT applications faster, easier, and to scale delivering real business value quickly.

Overview
This project is a component of the Operator Framework, an open source toolkit to manage Kubernetes native applications, called Operators, 
in an effective, automated, and scalable way.

Operators make it easy to manage complex stateful applications on top of Kubernetes. However writing an operator today can be difficult 
because of challenges such as using low level APIs, writing boilerplate, and a lack of modularity which leads to duplication.

The Operator SDK is a framework designed to make writing operators easier by providing:

- High level APIs and abstractions to write the operational logic more intuitively
- Tools for scaffolding and code generation to bootstrap a new project fast
- Extensions to cover common operator use cases

Workflow
The SDK provides the following workflow to develop a new operator:

- Create a new operator project using the SDK Command Line Interface(CLI)
- Define new resource APIs by adding Custom Resource Definitions(CRD)
- Specify resources to watch using the SDK API
- Define the operator reconciling logic in a designated handler and use the SDK API to interact with resources
- Use the SDK CLI to build and generate the operator deployment manifests

At a high level an operator using the SDK processes events for watched resources in a user defined handler and takes actions to reconcile 
the state of the application.



Prerequisites: 
dep https://golang.github.io/dep/docs/installation.html
git https://git-scm.com/downloads
go  https://golang.org/dl/
docker.
kubectl.
Access to a kubernetes cluster.


Operator-SDK Installation:
First, checkout and install the operator-sdk CLI:

$ mkdir -p $GOPATH/src/github.com/operator-framework
$ cd $GOPATH/src/github.com/operator-framework
$ git clone https://github.com/operator-framework/operator-sdk
$ cd operator-sdk
$ make dep
$ make install

Create and deploy an influxdata-operator using the SDK CLI:

$ operator-sdk new influxdata-operator --api-version=dev9-labs.bitbucket.org/v1alpha1 --kind=Influxdb
$ cd influxdata-operator

- remove all directories and subdirectories inside influxdata-operator and clone it from our repo which contain all code logic and manifest.

git clone https://bitbucket.org/dev9-labs/influxdata-operator/src/master/

# Build and push the influxdata-operator image to a registry such as docker.io
$ operator-sdk build docker.io/example/influxdata-operator
$ docker push docker.io/example/influxdata-operator

- this image will be used in operator.yaml.

# Update the operator manifest to use the built image name
$ sed -i 's|REPLACE_IMAGE|docker.io/example/influxdata-operator|g' deploy/operator.yaml


# Deploy the Influxdata Operator
$ kubectl create -f deploy/sa.yaml
$ kubectl create -f deploy/rbac.yaml
$ kubectl create -f deploy/operator.yaml
$ kubectl create -f deploy/crd.yaml

# Deploy the Custom Resource for Influxdata Installation
$ kubectl create -f deploy/cr.yaml

# Verify that the deployment and pods are created

[root@kube-master influxdata-operator]# kubectl get deployment 
NAME                  DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
influxdata-operator   1         1         1            1           20h
influxdb              1         1         1            1           19h

[root@kube-master influxdata-operator]# kubectl get pods 
NAME                                   READY   STATUS    RESTARTS   AGE
influxdata-operator-7d764b76fb-g7s4h   1/1     Running   0          20h
influxdb-76c6b44bc8-mshsw              1/1     Running   0          20h

- get IP address from the below command
kubectl describe pod influxdb-76c6b44bc8-mshsw

[root@kube-master influxdata-operator]# curl -G 'http://10.36.0.1:8086/query' --data-urlencode 'q=SHOW DATABASES'
{"results":[{"statement_id":0,"series":[{"name":"databases","columns":["name"],"values":[["_internal"]]}]}]}

[root@kube-master influxdata-operator]# curl -i -XPOST http://10.36.0.1:8086/query --data-urlencode "q=CREATE DATABASE mydb"
HTTP/1.1 200 OK
Content-Type: application/json
Request-Id: 52b588c2-c735-11e8-8003-000000000000
X-Influxdb-Build: OSS
X-Influxdb-Version: 1.6.3
X-Request-Id: 52b588c2-c735-11e8-8003-000000000000
Date: Wed, 03 Oct 2018 17:54:06 GMT
Transfer-Encoding: chunked

{"results":[{"statement_id":0}]}

[root@kube-master influxdata-operator]# curl -G 'http://10.36.0.1:8086/query' --data-urlencode 'q=SHOW DATABASES'
{"results":[{"statement_id":0,"series":[{"name":"databases","columns":["name"],"values":[["_internal"],["mydb"]]}]}]}
[root@kube-master influxdata-operator]# 

# Cleanup
$ kubectl delete -f deploy/cr.yaml
$ kubectl delete -f deploy/crd.yaml
$ kubectl delete -f deploy/operator.yaml
$ kubectl delete -f deploy/rbac.yaml
$ kubectl delete -f deploy/sa.yaml