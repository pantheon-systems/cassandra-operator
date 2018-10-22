package nodetool

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	corev1 "k8s.io/api/core/v1"
)

// Status represtes the results of the nodetool status command
type Status struct {
	Status     NodeStatus
	State      NodeState
	Address    string
	Load       string
	TokenCount int
	Owns       float32
	HostID     string
	Rack       string
	Datacenter string
}

// GetStatus retrieves the status of a node within the cassandra cluster (ring)
func (n *Executor) GetStatus(node *corev1.Pod) (map[string]*Status, error) {
	output, err := n.run(node, "status", []string{})
	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(output))

	vals := make(map[string]*Status)
	dc := ""
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if strings.Contains(line, "Datacenter") {
			dc = strings.TrimSpace(strings.Split(line, ":")[1])

			/*
				throw away 4 lines
				=======================
				Status=Up/Down
				|/ State=Normal/Leaving/Joining/Moving
				--  Address          Load       Tokens       Owns (effective)  Host ID                               Rack
			*/
			scanner.Scan()
			scanner.Scan()
			scanner.Scan()
			scanner.Scan()
			continue
		}

		nodeStatus, err := processNode(line, dc)
		if err != nil {
			return nil, err
		}
		vals[nodeStatus.HostID] = nodeStatus
	}

	return vals, nil
}

func processNode(line string, dc string) (*Status, error) {
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '.' && c != '-'
	}
	fields := strings.FieldsFunc(line, f)

	// as of building this, nodetool status outputs 8 cols for each node
	if len(fields) != 8 {
		return nil, fmt.Errorf("Invalid format for nodetool status output, had %d, expected %d", len(fields), 8)
	}

	tokenCount, err := strconv.Atoi(fields[4])
	if err != nil {
		return nil, err
	}

	ownsPercentage, err := strconv.ParseFloat(fields[5], 32)
	if err != nil {
		return nil, err
	}

	return &Status{
		Status:     getNodeStatus(fields[0][0]),
		State:      getNodeState(fields[0][1]),
		Address:    fields[1],
		Load:       fmt.Sprintf("%s %s", fields[2], fields[3]),
		TokenCount: tokenCount,
		Owns:       float32(ownsPercentage),
		HostID:     fields[6],
		Rack:       fields[7],
		Datacenter: dc,
	}, nil
}

func getNodeStatus(b byte) NodeStatus {
	switch b {
	case 85:
		return NodeStatusUp
	case 68:
		return NodeStatusDown

	}
	return NodeStatusUnknown
}

func getNodeState(b byte) NodeState {
	switch b {
	case 78:
		return NodeStateNormal
	case 77:
		return NodeStateMoving
	case 74:
		return NodeStateJoining
	case 76:
		return NodeStateLeaving
	}
	return NodeStateUnknown
}
