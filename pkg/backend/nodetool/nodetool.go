package nodetool

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

const (
	cassandraPodName = "cassandra"
	nodetoolFullPath = "/usr/bin/nodetool"
)

// PodExecutor implements logic for executing commands inside pods
// allows for constraint based arguments to allow unit testing
type podExecutor interface {
	Run(pod *corev1.Pod, containerIdx int, command []string) (string, string, error)
}

// Executor implements the logic for executing nodetool commands inside
// kubernetes pods
type Executor struct {
	executor podExecutor
}

// NewExecutor creates a new Nodetool for running commands on cassandra cluster
func NewExecutor(executor podExecutor) *Executor {
	return &Executor{
		executor: executor,
	}
}

// run executes a nodetool command on a specified pod(node), if pod is nil, the first found ready
// pod will have the nodetool command executed instead
func (n *Executor) run(execPod *corev1.Pod, command string, options []string) (string, error) {
	if execPod == nil {
		return "", fmt.Errorf("NodetoolExecutor requires a pod to execute on")
	}

	containerIdx := n.getCassandraContainerIdx(execPod)
	if containerIdx == -1 {
		return "", fmt.Errorf("No container named %s in pod %s", cassandraPodName, execPod.GetName())
	}

	outputStdOut, outputStdErr, err := n.executor.Run(execPod, containerIdx, append([]string{nodetoolFullPath, command}, options...))
	if err != nil {
		return "", err
	}

	if len(outputStdErr) > 0 {
		return "", fmt.Errorf(outputStdErr)
	}

	return outputStdOut, nil
}

// getCassandraContainerIdx find the container named cassandra, if no
// container in the pod is named cassandra, returns -1
func (n *Executor) getCassandraContainerIdx(pod *corev1.Pod) int {
	containerIdx := -1
	for idx, container := range pod.Spec.Containers {
		if container.Name == cassandraPodName {
			containerIdx = idx
			break
		}
	}

	return containerIdx
}
