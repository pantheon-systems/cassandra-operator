package nodetool

import (
	"errors"
	corev1 "k8s.io/api/core/v1"
)

// Decommission triggers a nodetool decommission on the node to
// begin the process of scaling down or replacing the node
func (e *Executor) Decommission(node *corev1.Pod) error {
	_, err := e.run(node, "decommission", []string{})
	if err != nil {
		return err
	}

	hostID, err := e.GetHostID(node)
	if err != nil {
		return err
	}

	statuses, err := e.GetStatus(node)
	if err != nil {
		return err
	}

	if _, ok := statuses[hostID]; !ok {
		return errors.New("node decommission failed")
	}

	return nil
}
