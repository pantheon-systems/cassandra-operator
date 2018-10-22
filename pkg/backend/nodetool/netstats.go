package nodetool

import (
	"bufio"
	corev1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
	"unicode"
)

const (
	na = "n/a"
)

//while true; do date; diff <(nodetool -h localhost netstats) <(sleep 5 && nodetool -h localhost netstats); done

// Netstats is the result of the nodetool netstats command
type Netstats struct {
	Mode NodeMode
	// The number of successfully completed read repair operations
	AttemptedReadRepairOps int
	// The number of read repair operations since server restart that blocked a query.
	MismatchBlockingReadRepairOps int
	// The number of read repair operations since server restart performed in the background.
	MismatchBgReadRepairOps int
	// Information about client read and write requests by thread pool.
	ThreadPoolNetstats []ThreadPoolNetstat
}

// ThreadPoolNetstat contains active, pending, and completed number of commands and responses for a threadpool
type ThreadPoolNetstat struct {
	Name      string
	Active    int
	Pending   int
	Completed int
	Dropped   int
}

// GetNetstats triggers nodetool netstats which provides information about the host
func (e *Executor) GetNetstats(node *corev1.Pod) (*Netstats, error) {
	out, err := e.run(node, "netstats", []string{})
	if err != nil || out == "" {
		return nil, err
	}

	scanner := bufio.NewScanner(strings.NewReader(out))

	netstat := &Netstats{}
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		splitLine := strings.Split(line, ":")
		if len(splitLine) == 2 {
			key := splitLine[0]
			value := strings.TrimSpace(splitLine[1])
			if key == "Mode" {
				netstat.Mode = NodeMode(value)
				continue
			}

			if key == "Attempted" {
				attempted, err := strconv.Atoi(value)
				if err != nil {
					return nil, err
				}

				netstat.AttemptedReadRepairOps = attempted
				continue
			}

			if key == "Mismatch (Blocking)" {
				blocking, err := strconv.Atoi(value)
				if err != nil {
					return nil, err
				}

				netstat.MismatchBlockingReadRepairOps = blocking
				continue
			}

			if key == "Mismatch (Background)" {
				bg, err := strconv.Atoi(value)
				if err != nil {
					return nil, err
				}

				netstat.MismatchBgReadRepairOps = bg
				continue
			}
		}

		if strings.Contains(line, "Pool Name") {
			netstat.ThreadPoolNetstats = []ThreadPoolNetstat{}
			for scanner.Scan() {
				stat, err := processThreadPool(scanner.Text())
				if err != nil {
					return nil, err
				}
				netstat.ThreadPoolNetstats = append(netstat.ThreadPoolNetstats, *stat)
			}
		}
	}

	return netstat, nil
}

func processThreadPool(line string) (*ThreadPoolNetstat, error) {
	var err error

	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '.' && c != '-' && c != '/'
	}
	fields := strings.FieldsFunc(line, f)

	poolNameParts := []string{}
	baseIdx := 0
	// gather multi-part pool name
	for i, v := range fields {
		_, err = strconv.Atoi(fields[i])
		if v == na || err == nil {
			baseIdx = i
			break
		}
		poolNameParts = append(poolNameParts, v)
	}

	poolName := strings.Join(poolNameParts, " ")

	var activeCount int
	if len(fields) > baseIdx && fields[baseIdx] != na {
		activeCount, err = strconv.Atoi(fields[baseIdx])
		if err != nil {
			return nil, err

		}
	}

	var pendingCount int
	pendingIdx := baseIdx + 1
	if len(fields) > pendingIdx && fields[pendingIdx] != na {
		pendingCount, err = strconv.Atoi(fields[pendingIdx])
		if err != nil {
			return nil, err
		}
	}

	var completedCount int
	completedIdx := baseIdx + 2
	if len(fields) > completedIdx && fields[completedIdx] != na {
		completedCount, err = strconv.Atoi(fields[completedIdx])
		if err != nil {
			return nil, err
		}
	}

	var droppedCount int
	droppedIdx := baseIdx + 3
	if len(fields) > droppedIdx && fields[droppedIdx] != na {
		droppedCount, err = strconv.Atoi(fields[droppedIdx])
		if err != nil {
			return nil, err
		}
	}

	tp := &ThreadPoolNetstat{
		Name:      poolName,
		Active:    activeCount,
		Pending:   pendingCount,
		Completed: completedCount,
		Dropped:   droppedCount,
	}

	return tp, nil
}
