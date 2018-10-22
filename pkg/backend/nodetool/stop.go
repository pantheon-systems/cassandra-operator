package nodetool

import (
	corev1 "k8s.io/api/core/v1"
)

// Stop triggers the nodetool stop operation on a cassandra node
// in preparation for restart
func (e *Executor) Stop(node *corev1.Pod) error {
	_, err := e.run(node, "stop", []string{})
	return err
}
