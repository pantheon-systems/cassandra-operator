package nodetool

import (
	"bufio"
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// GetHostID returns the cassandra internal host ID
func (n *Executor) GetHostID(node *corev1.Pod) (string, error) {
	output, err := n.run(node, "info", []string{})
	if err != nil {
		return "", err
	}

	nodeID := ""
	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		line := strings.Split(scanner.Text(), ":")
		if strings.TrimSpace(line[0]) == "ID" {
			nodeID = strings.TrimSpace(line[1])
			break
		}
	}

	return nodeID, nil
}
