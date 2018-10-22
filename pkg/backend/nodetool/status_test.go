package nodetool_test

import (
	"fmt"
	"github.com/pantheon-systems/cassandra-operator/pkg/backend/nodetool"
	"testing"

	"github.com/pantheon-systems/cassandra-operator/pkg/backend/k8s"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

var (
	TestStatusOutput = `
Datacenter: us-central1
=======================
Status=Up/Down
|/ State=Normal/Leaving/Joining/Moving
--  Address          Load       Tokens       Owns (effective)  Host ID                               Rack
UJ  104.197.117.166  44.9 GB    256          36.5%             30bfd332-9113-4e0f-b453-0e90d9a00bdc  us-central1-c
UL  104.197.168.13   36.52 GB   256          37.0%             379874b8-3d69-4dce-a3a9-692fff8acd33  us-central1-f
DN  35.232.245.147   35.27 GB   256          38.9%             55808991-d091-4509-9cee-698d20b7685e  us-central1-f
UM  35.202.20.228    43.67 GB   256          37.8%             a0d7f4a1-cda7-43da-b6eb-305d08236486  us-central1-c
Datacenter: us-central1-2
=========================
Status=Up/Down
|/ State=Normal/Leaving/Joining/Moving
--  Address          Load       Tokens       Owns (effective)  Host ID                               Rack
UN  35.224.81.85     128 GB     256          100.0%            f9484164-e778-4a7f-8540-d6885cc7574b  us-central1-b
UJ  104.198.187.232  117.34 GB  256          100.0%            b38803dc-3216-4355-a22c-b986b3c970bc  us-central1-a
XX  35.224.58.223    138.41 GB  256          100.0%            368908b2-ee4b-4c86-a323-5d92e7ceaa19  us-central1-c

`
	testExtraColStatusOutput = `
Datacenter: us-central1
=======================
Status=Up/Down
|/ State=Normal/Leaving/Joining/Moving
--  Address          Load       Tokens       Owns (effective)  Host ID                               Rack           SomeNewField
UJ  104.197.117.166  44.9 GB    256          36.5%             30bfd332-9113-4e0f-b453-0e90d9a00bdc  us-central1-c  testValue1
`
	testInvalidValueTokensStatusOutput = `
Datacenter: us-central1
=======================
Status=Up/Down
|/ State=Normal/Leaving/Joining/Moving
--  Address          Load       Tokens       Owns (effective)  Host ID                               Rack
UJ  104.197.117.166  44.9 GB    NaN          36.5%             30bfd332-9113-4e0f-b453-0e90d9a00bdc  us-central1-c
`

	// 	testQuestionMarkInOwnsColumn = `
	// Datacenter: us-central1
	// =======================
	// Status=Up/Down
	// |/ State=Normal/Leaving/Joining/Moving
	// --  Address          Load       Tokens       Owns (effective)  Host ID                               Rack
	// UJ  104.197.117.166  44.9 GB    32           ?                 30bfd332-9113-4e0f-b453-0e90d9a00bdc  us-central1-c
	// `

	testInvalidValueOwnsStatusOutput = `
Datacenter: us-central1
=======================
Status=Up/Down
|/ State=Normal/Leaving/Joining/Moving
--  Address          Load       Tokens       Owns (effective)  Host ID                               Rack
UJ  104.197.117.166  44.9 GB    256          Something         30bfd332-9113-4e0f-b453-0e90d9a00bdc  us-central1-c
`

	testMissingColStatusOutput = `
Datacenter: us-central1
=======================
Status=Up/Down
|/ State=Normal/Leaving/Joining/Moving
--  Address          Load       Tokens       Owns (effective)  Host ID
UJ  104.197.117.166  44.9 GB    256          36.5%             30bfd332-9113-4e0f-b453-0e90d9a00bdc
`
)

func TestGetStatus_PodNil(t *testing.T) {
	mockClient := &k8s.MockClient{}
	obj := nodetool.NewExecutor(mockClient)
	statuses, err := obj.GetStatus(nil)

	assert.Error(t, err)
	assert.Nil(t, statuses)
}

func TestGetStatus_NoCassandraContainer(t *testing.T) {
	testPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "NotCassandra1",
				},
				{
					Name: "NotCassandra2",
				},
			},
		},
	}

	mockClient := &k8s.MockClient{}
	obj := nodetool.NewExecutor(mockClient)
	statuses, err := obj.GetStatus(testPod)

	assert.Error(t, err)
	assert.Nil(t, statuses)
}

func TestGetStatus_ExecutorError(t *testing.T) {
	testPod := &corev1.Pod{}

	mockClient := &k8s.MockClient{
		RunErr: fmt.Errorf("Some fake error"),
	}
	obj := nodetool.NewExecutor(mockClient)
	statuses, err := obj.GetStatus(testPod)

	assert.Error(t, err)
	assert.Nil(t, statuses)
}

func TestGetStatus_ExecutorStdError(t *testing.T) {
	testPod := &corev1.Pod{}

	mockClient := &k8s.MockClient{
		RunStdErr: "Some fake error",
	}
	obj := nodetool.NewExecutor(mockClient)
	statuses, err := obj.GetStatus(testPod)

	assert.Error(t, err)
	assert.Nil(t, statuses)
}

func TestGetStatus_InvalidFieldCount(t *testing.T) {
	testPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "cassandra",
				},
			},
		},
		Status: corev1.PodStatus{},
	}

	mockClient := &k8s.MockClient{
		RunStdOut: testExtraColStatusOutput,
		RunStdErr: "",
		RunErr:    nil,
	}
	obj := nodetool.NewExecutor(mockClient)
	statuses, err := obj.GetStatus(testPod)

	assert.Error(t, err)
	assert.Nil(t, statuses)

	mockClient.RunStdOut = testMissingColStatusOutput
	statuses, err = obj.GetStatus(testPod)

	assert.Error(t, err)
	assert.Nil(t, statuses)
}

func TestGetStatus_InvalidValues(t *testing.T) {
	testPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "cassandra",
				},
			},
		},
		Status: corev1.PodStatus{},
	}

	mockClient := &k8s.MockClient{
		RunStdOut: testInvalidValueTokensStatusOutput,
		RunStdErr: "",
		RunErr:    nil,
	}
	obj := nodetool.NewExecutor(mockClient)
	statuses, err := obj.GetStatus(testPod)

	assert.Error(t, err)
	assert.Nil(t, statuses)

	mockClient.RunStdOut = testInvalidValueOwnsStatusOutput
	statuses, err = obj.GetStatus(testPod)

	assert.Error(t, err)
	assert.Nil(t, statuses)
}

func TestGetStatus_SuccessWithNewLine(t *testing.T) {
	testPod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "cassandra",
				},
			},
		},
		Status: corev1.PodStatus{},
	}

	mockClient := &k8s.MockClient{
		RunStdOut: TestStatusOutput,
		RunStdErr: "",
		RunErr:    nil,
	}
	obj := nodetool.NewExecutor(mockClient)
	statuses, err := obj.GetStatus(testPod)

	assert.NoError(t, err)
	assert.Len(t, statuses, 7)

	expected := map[string]nodetool.Status{
		"30bfd332-9113-4e0f-b453-0e90d9a00bdc": {
			Status:     nodetool.NodeStatusUp,
			State:      nodetool.NodeStateJoining,
			Address:    "104.197.117.166",
			Load:       "44.9 GB",
			TokenCount: 256,
			Owns:       36.5,
			HostID:     "30bfd332-9113-4e0f-b453-0e90d9a00bdc",
			Rack:       "us-central1-c",
			Datacenter: "us-central1",
		},
		"379874b8-3d69-4dce-a3a9-692fff8acd33": {
			Status:     nodetool.NodeStatusUp,
			State:      nodetool.NodeStateLeaving,
			Address:    "104.197.168.13",
			Load:       "36.52 GB",
			TokenCount: 256,
			Owns:       37.0,
			HostID:     "379874b8-3d69-4dce-a3a9-692fff8acd33",
			Rack:       "us-central1-f",
			Datacenter: "us-central1",
		},
		"55808991-d091-4509-9cee-698d20b7685e": {
			Status:     nodetool.NodeStatusDown,
			State:      nodetool.NodeStateNormal,
			Address:    "35.232.245.147",
			Load:       "35.27 GB",
			TokenCount: 256,
			Owns:       38.9,
			HostID:     "55808991-d091-4509-9cee-698d20b7685e",
			Rack:       "us-central1-f",
			Datacenter: "us-central1",
		},
		"a0d7f4a1-cda7-43da-b6eb-305d08236486": {
			Status:     nodetool.NodeStatusUp,
			State:      nodetool.NodeStateMoving,
			Address:    "35.202.20.228",
			Load:       "43.67 GB",
			TokenCount: 256,
			Owns:       37.8,
			HostID:     "a0d7f4a1-cda7-43da-b6eb-305d08236486",
			Rack:       "us-central1-c",
			Datacenter: "us-central1",
		},
		"f9484164-e778-4a7f-8540-d6885cc7574b": {
			Status:     nodetool.NodeStatusUp,
			State:      nodetool.NodeStateNormal,
			Address:    "35.224.81.85",
			Load:       "128 GB",
			TokenCount: 256,
			Owns:       100.0,
			HostID:     "f9484164-e778-4a7f-8540-d6885cc7574b",
			Rack:       "us-central1-b",
			Datacenter: "us-central1-2",
		},
		"b38803dc-3216-4355-a22c-b986b3c970bc": {
			Status:     nodetool.NodeStatusUp,
			State:      nodetool.NodeStateJoining,
			Address:    "104.198.187.232",
			Load:       "117.34 GB",
			TokenCount: 256,
			Owns:       100.0,
			HostID:     "b38803dc-3216-4355-a22c-b986b3c970bc",
			Rack:       "us-central1-a",
			Datacenter: "us-central1-2",
		},
		"368908b2-ee4b-4c86-a323-5d92e7ceaa19": {
			Status:     nodetool.NodeStatusUnknown,
			State:      nodetool.NodeStateUnknown,
			Address:    "35.224.58.223",
			Load:       "138.41 GB",
			TokenCount: 256,
			Owns:       100.0,
			HostID:     "368908b2-ee4b-4c86-a323-5d92e7ceaa19",
			Rack:       "us-central1-c",
			Datacenter: "us-central1-2",
		},
	}

	for hostID, expectedStatus := range expected {
		actualStatus := statuses[hostID]
		assert.EqualValues(t, expectedStatus, *actualStatus)
	}
}
