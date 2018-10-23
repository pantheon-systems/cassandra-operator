# Lifecycle Events

Nodetool commands can be run using: 
`kubectl exec <podname> -- nodetool <command> <options>`

If they are interactive then you must create the tty:
`kubectl exec -it <podname> -- nodetool <command> <options>`

## Scale Up

_NOTE: Increasing replicas by more than one will serialize the creation process. The operator creates the first node and only allows the next node to be created after the previous node fully joins the ring and has bootstrapped all data. During this process it is recommended to not modify any cassandra related resources or the CRD till it is complete._

1. Modify CRD

```yaml
Spec:
  size: __REPLICAS__
```

2. CI/DI Pipeline runs `kubectl apply -f <crd yaml file>`
3. Run `kubectl create job --from=cronjob/__APP__-__CASSANDRA_CLUSTER__-repair <give your job a name>`

See: [pantheon-systems/cassandra-repair-cron](https://github.com/pantheon-systems/cassandra-repair-cron)

4. For each pod that is a cassandra node:
`kubectl exec <cassandra pod name> -- nodetool cleanup`

## Scale Down

>This is a Work in progress and must be done manually for now. It could be added with a small amount of effort.

_NOTE: Decreasing replicas by more than 1 will not work properly at this time due to the need to manually run decommission for each node that is removed from the ring before the next one is removed. The persistent volume used to back the pod will not be deleted from GCP. It will remain accessible and be retained till it is manually deleted._

1. For the node being removed run: `kubectl exec <pod being removed name> -- nodetool decommission`
2. Modify CRD (1 node at a time)

```yaml
Spec:
  size: __REPLICAS__
```

3. CI/DI Pipeline runs `kubectl apply -f <crd yaml file>` _NOTE: Will ERROR -> the automation will try to run nodetool drain and nodetool stop at this point and fail, the finalizer will be removed in the next step_
4. Remove finalizer from pod `kubectl patch pod <pod being removed name> -p '{"metadata":{"finalizers":null}}'`

## Create Empty Cluster

_NOTE: The creation process will serialize the creation of nodes. Wait for all nodes to be created/bootstrapped before utilizing the cluster or making changes to the cluster._

1. Create CRD. See reference CRD below 
2. `kubectl create -f CRD.yaml`

## Create New DC on Existing Cluster (TODO/WIP)

_NOTE: New DCs must be created one node at a time as the operator does not allow calculate auto_bootstrap correctly for a multi-node new DC cluster. This can be fixed by adding logic to the statefulset method that calculates the auto_boostrap value based on CRD options. This would be fairly low effort work to automate._

1. Create CRD with `externalSeeds` set to a comma seperated list of the seeds in the other datacenter and `size` set to 1.
2. `kubectl create -f CRD.yaml`
3. The single node will come up with auto_bootstrap disabled, then run the following: `kubectl exec <podname> -- nodetool rebuild -- <name of other DC>`
4. After first node is fully joined, scale up to desired state using the Scale Up instructions

## Restarting a Node (Or All Nodes)

_NOTE: This handles the cassandra lifecycle process for restarting a node. To restart all nodes, do a single node at a time and wait for the restarting node to come back fully online._

1. `kubectl delete pod <pod-name>`

## Deleting a Cluster

_Note: The persistent volumes for the pods will be retained after deletion of the cluster._

1. `kubectl delete cassandracluster <clustername>`

## Reference CRD:

```yaml
apiVersion: "database.pantheon.io/v1alpha1"
kind: "CassandraCluster"
metadata:
name: __CASSANDRA_CLUSTER__
labels:
    app: __APP__
    component: "datastore"
spec:
    size: __REPLICAS__
    repair:
    schedule: "22 6 * * 0,4"
    image: "quay.io/getpantheon/cassandra-repair:11"
    node:
    image: __IMAGE__
    fileMountPath: /var/lib/cassandra
    resources:
        limits:
        cpu: __CPU_LIMIT__
        memory: __MEMORY_LIMIT__
        requests:
        cpu: __CPU_REQUEST__
        memory: __MEMORY_REQUEST__
        persistentVolume:
        resources:
            storage: __DISK_SIZE__
        storageClassName: "ssd"
    keyspaceName: somekeyspace
    secretName: "__APP__-cassandra-certs"
    configMapName: "__APP__-config"
    jvmAgentConfigName: "__APP__-prometheus-jvm-agent-config"
    jvmAgent: agent
    datacenter: override-dc-name
    externalSeeds: “other-dc-seed.cassandra.somedomain.net,third-dc-seed.cassandra.somedomain.net”
    enablePublicPodServices: true
    exposePublicLB: true
    enablePodDisruptionBudget: true
    affinity: <same as kube affinity/anti-affinity structure>
```

_NOTE: For affinity structure see [https://kubernetes.io/blog/2017/03/advanced-scheduling-in-kubernetes/](https://kubernetes.io/blog/2017/03/advanced-scheduling-in-kubernetes/)_

_NOTE: If repair is not set then the repair cronjob is disabled_



