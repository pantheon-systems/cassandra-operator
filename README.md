# Cassandra Operator
This repository contains the cassandra cluster kubernetes operator. The operator consists of the CustomResourceDefinition (CRD) and a Kubernetes Controller. It is a work in progress.

[![Unofficial](https://img.shields.io/badge/Pantheon-Unofficial-yellow?logo=pantheon&color=FFDC28)](https://pantheon.io/docs/oss-support-levels#unofficial)

The image is stored at: https://quay.io/repository/getpantheon/cassandra-operator

## Operator-SDK
We are using the `0.0.7` branch of the operator-sdk.
https://github.com/operator-framework/operator-sdk

This sdk is under heavy development.

## Current Capabilities
* Create a single node empty cluster
* Create a multi-node empty cluster
* Scale up a single node and down a single node
** Does not call cassandra lifecycle at this time
* Add ExternalSeeds to CRD to setup multi-dc
* Delete a cluster that has been created with the operator
** Persistant Volumes (data disk) is retained and must be manually deleted
** The system does not currently decommission the cluster before deleting

## Deploying the Operator
The operator comes in two parts. The Custom Resource Definition must be created first and is per cluster task. The
operator service runs in a kube pod and needs to be deployed to each cluster as well. The pod will run in the `kube-system`
namespace.

### CustomResourceDefinition (CRD)
>kubectl -n kube-system --context \<cluster\> create -f deploy/crd.yaml
### Operator Service
The operator must be deployed on any k8s cluster that is expected to run cassandra cluster resources. Rename the `./deploy/operator.yaml.example` and rename it `./deploy/operator.yaml`. Set the docker image tag for the operator service and run the following commands:

>KUBE_CONTEXT=gke_pantheon-internal_us-central1-a_cluster-02 make deploy
>KUBE_CONTEXT=gke_pantheon-internal_us-central1-b_cluster-01 make deploy

This will create a running container of the cassandra operator image on the cluster and register the CRD as well.

NOTE: you can specify the release (image tag) using the `$VERSION` and `$UNIQUE_TAG` enviornment variables. Our images are tagged with `v<version>-<unique-tag>` (eg `v0.0.1-20e37818-e3e2-4675-ab10-aa065045f753`) where the unique tag is either a git commit 
hash or a circle ci workflow id.

### Repairs
The cassandra operator can automatically manage repair jobs. To enable this feature you must set the values for the `v1alpha1.RepairPolicy`:

```yaml
apiVersion: "database.pantheon.io/v1alpha1"
kind: "CassandraCluster"
metadata:
  name: "example-application"
spec:
  size: 1 # ring size
  repair:
    schedule: "22 6 * * 0,4"
    image: "quay.io/getpantheon/cassandra-repair:11"
...
```

NOTE: The schedule is specified in Cron format. See [wikipedia](https://en.wikipedia.org/wiki/Cron#CRON_expression)

### Multi-DC Deployment
When `externalSeeds` is set in the v1alphaCassandraCluster.ClusterSpec section of the custom resource, the cluster that is created will be created as a second datacenter of the clusters that the external seeds are members. The comma-seperated list of external seeds are appeneded to the seed list created for the new ring, and auto-bootstrap is disabled for the new node. Currently we only support single node second datacenter creation. The workaround is to scale up the new datacenter after initial creation and `nodetool rebuild -- <name of other dc>` is completed.

If the other datacenter is not in the same physical network as the new ring being constructed, in the yaml set:
```
enablePublicPodServices: true
```

## The `v1alpha1.cassandracluster` Custom Resource
In the `./deploy` directory you will find a `sample.yaml` file which contains a sample cassandra cluster setup.

### API Documentation
The v1alpha1.CassandraCluster API documenation is located at:
[GoDoc Coming Soon]()

### JVM Agents
The cassandra operator uses JMX agents to get information from the JVM about how cassandra is running. It is also used in some areas instead of nodetool. The JMX agents are also used to feed the metrics system for cassandra. There are two supported options for how this can be configured:

#### Jolokia Agent Attched
```yaml
apiVersion: "database.pantheon.io/v1alpha1"
kind: "CassandraCluster"
metadata:
  name: "example-application"
spec:
    ...
    jvmAgent: "agent"
    jvmAgentConfigName: "<configmap name goes here>"
    ...
```

See Jolokia documenation [here](https://jolokia.org/documentation.html)

#### Telegraf Agent Sidecar
```yaml
apiVersion: "database.pantheon.io/v1alpha1"
kind: "CassandraCluster"
metadata:
  name: "example-application"
spec:
    ...
    jvmAgent: "sidecar"
    jvmAgentConfigName: "<configmap name goes here>"
    ...
```

##### Telegraf Configuration
A configmap should be created that has this file as the value and the key `telegraf.conf`

See Telegraf config documenation [here](https://github.com/influxdata/telegraf/blob/master/etc/telegraf.conf)

## The Cassandra Docker Image
The image provided to the operator for the cassandra image (which can be sepecified in the CRD) should meet the following:

### Exposed Ports
The following ports are expected to be exposed by the cassandra container:
* 7000 - intra-node communication
* 7001 - intra-node tls based communication
* 7199 - JMX port
* 9042 - CQL
* 9160 - Thrift
* 8778 - Metrics

### Enviornment Variables
The operator will pass configuration options to cassandra on startup through enviornment variables. These should be used to populate values in the `cassandra.yaml` file:

* CASSANDRA_DC: Name of datacenter, if not set lets snitch set the DC name
* POD_NAMESPACE: From the downward API passing in the namespace of the pod (metadata.namespace)
* POD_IP: From the downward API passing in the pod private IP address (status.podIP)
* CASSANDRA_CLUSTER_NAME: Name of the cluster 
* SERVICE_NAME: Name of the public service used as the LB for CQL/Thrift access
* CASSANDRA_ALLOCATE_TOKENS_FOR_KEYSPACE: Name of the keyspace to create on startup (defaults to cluster name)
* CASSANDRA_MAX_HEAP: Maximum heap size for the JVM
* CASSANDRA_MIN_HEAP: Minimum head size for the JVM
* CASSANDRA_SEEDS: Comma seperated seed list for the ring
* CASSANDRA_AUTO_BOOTSTRAP: Boolean if the node should auto-bootstrap from the rest of the cluster on startup

### Secrets

The certificates that cassandra uses should be in a secret called `test-cluster-cassandra-certs` where `test-cluster` is the name of the cluster specified in the CRD. These certificates in the secret will be attached to the container at a volume at the `/keystore` mount path.

### Configmaps

Depending on which JVM agent you choose you will need to provide a configuration. The configuration should be stored as a configmap resource in kube. The default name for the configmap is `test-cluster-prometheus-jvm-agent-config` where `test-cluster` is the cluster name. 

If you choose the JvmAgent is `sidecar` then the telegraf sidecar container will have this configmap mounted at the `/telegraf-config` mount point if JvmAgent is default or set to `jvm` then the jolokia sidecar is used and mounted in the primary cassandra container at the `/jvm-agent` mount point.

### Version Taint

Developers can run multiple operators in a single kubernetes cluster and not cross paths by using the `version-taint` command line option to the operator executable. Use `--version-taint=<something unique here>` to enable it, this will flag your clusters with a version tag and sets your operator to only operate on your clusters.

The annotation for the managing operator version is `database.panth.io/cassandra-operator-version`.

### Feature Flag

Feature flags have been implemented using annotations on the objects that they toggle features on.

#### Available Feature Flags

* `disable-pod-finalizer` disables finalizers on the pods representing cassandra nodes (added to corev1.Pod)
