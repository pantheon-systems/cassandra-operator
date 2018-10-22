package nodetool_test

import (
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	"testing"

	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
)

var (
	testInfoOutput = `ID                     : 3b920369-cd41-4b6b-8f5f-192f1202ee18
Gossip active          : true
Thrift active          : true
Native Transport active: true
Load                   : 43.16 GB
Generation No          : 1529817347
Uptime (seconds)       : 1871647
Heap Memory (MB)       : 1584.43 / 6104.00
Off Heap Memory (MB)   : 33.42
Data Center            : us-central1
Rack                   : us-central1-b
Exceptions             : 0
Key Cache              : entries 1867103, size 237.81 MB, capacity 256 MB, 45455336581 hits, 45903812808 requests, 0.990 recent hit rate, 14400 save period in seconds
Row Cache              : entries 0, size 0 bytes, capacity 0 bytes, 0 hits, 0 requests, NaN recent hit rate, 0 save period in seconds
Counter Cache          : entries 0, size 0 bytes, capacity 50 MB, 0 hits, 0 requests, NaN recent hit rate, 7200 save period in seconds
Token                  : (invoke with -T/--tokens to see all 256 tokens)
`
)

func TestGetHostID_Success(t *testing.T) {
	testPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "cassandra",
				},
			},
		},
	}
	mockClient := &k8s.MockClient{
		RunStdOut: testInfoOutput,
	}
	obj := nodetool.NewExecutor(mockClient)

	result, err := obj.GetHostID(testPod)

	assert.NoError(t, err)
	assert.Equal(t, "3b920369-cd41-4b6b-8f5f-192f1202ee18", result)
}
