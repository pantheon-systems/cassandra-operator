package nodetool

import (
	"errors"
	corev1 "k8s.io/api/core/v1"
)

// Drain triggers the nodetool drain operation on a cassandra node
// in preparation for restart
func (e *Executor) Drain(node *corev1.Pod) error {
	_, err := e.run(node, "drain", []string{})
	if err != nil {
		return err
	}

	statusthrift, err := e.run(node, "statusthrift", []string{})
	if err != nil {
		return err
	}

	statusbinary, err := e.run(node, "statusbinary", []string{})
	if err != nil {
		return err
	}

	if statusthrift == "running" || statusbinary == "running" {
		return errors.New("node drain failed")
	}

	return nil
}
