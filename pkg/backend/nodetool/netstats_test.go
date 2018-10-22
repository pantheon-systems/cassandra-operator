package nodetool_test

import (
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

var (
	valid1 = `
Mode: NORMAL
Not sending any streams.
Read Repair Statistics:
Attempted: 1177089787
Mismatch (Blocking): 377073
Mismatch (Background): 331734
Pool Name                    Active   Pending      Completed   Dropped
Large messages                  n/a         0        3545464         0
Small messages                  n/a         1   125497835413     76041
Gossip messages                 n/a         0        7281116         0
`

	valid1Result = &nodetool.Netstats{
		Mode:                          nodetool.NodeModeNormal,
		AttemptedReadRepairOps:        1177089787,
		MismatchBlockingReadRepairOps: 377073,
		MismatchBgReadRepairOps:       331734,
		ThreadPoolNetstats: []nodetool.ThreadPoolNetstat{
			{
				Name:      "Large messages",
				Active:    0,
				Pending:   0,
				Completed: 3545464,
				Dropped:   0,
			},
			{
				Name:      "Small messages",
				Active:    0,
				Pending:   1,
				Completed: 125497835413,
				Dropped:   76041,
			},
			{
				Name:      "Gossip messages",
				Active:    0,
				Pending:   0,
				Completed: 7281116,
				Dropped:   0,
			},
		},
	}
)

func TestExecutor_GetNetstats(t *testing.T) {
	type args struct {
		node   *corev1.Pod
		retVal string
	}
	tests := []struct {
		name    string
		args    args
		want    *nodetool.Netstats
		wantErr bool
	}{
		{
			name:    "NoContainers",
			args:    args{node: &corev1.Pod{}},
			want:    nil,
			wantErr: true,
		},
		// {
		// 	name: "InvalidMode",
		// 	args: args{
		// 		retVal: invalidMode,
		// 		node: &corev1.Pod{
		// 			Spec: corev1.PodSpec{
		// 				Containers: []corev1.Container{
		// 					corev1.Container{
		// 						Name: "cassandra",
		// 					},
		// 				},
		// 			},
		// 		},
		// 	},
		// 	want:    nil,
		// 	wantErr: true,
		// },
		{
			name: "Test1",
			args: args{
				retVal: valid1,
				node: &corev1.Pod{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "cassandra",
							},
						},
					},
				},
			},
			want:    valid1Result,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &k8s.MockClient{
				RunStdOut: tt.args.retVal,
			}
			e := nodetool.NewExecutor(mockClient)
			got, err := e.GetNetstats(tt.args.node)
			if (err != nil) != tt.wantErr {
				t.Errorf("Executor.GetNetstats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Executor.GetNetstats() = %v, want %v", got, tt.want)
			}
		})
	}
}
