package k8s

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

// MockClient implements a Mock of the sdk client
type MockClient struct {
	RunStdOut string
	RunStdErr string
	RunErr    error

	PatchCallback  func(object sdk.Object, pt types.PatchType, patch []byte) error
	GetCallback    func(into sdk.Object, opts ...sdk.GetOption) error
	CreateCallback func(object sdk.Object) error
	UpdateCallback func(object sdk.Object) error
	ListCallback   func(namespace string, into sdk.Object, opts ...sdk.ListOption) error
	RunCallback    func(pod *corev1.Pod, containerIdx int, command []string) (string, string, error)
}

// Patch returns mock value
func (c *MockClient) Patch(object sdk.Object, pt types.PatchType, patch []byte) error {
	if c.PatchCallback != nil {
		return c.PatchCallback(object, pt, patch)
	}
	return nil
}

// Get returns mock value
func (c *MockClient) Get(into sdk.Object, opts ...sdk.GetOption) error {
	if c.GetCallback != nil {
		return c.GetCallback(into, opts...)
	}
	return nil
}

// Create returns mock value
func (c *MockClient) Create(object sdk.Object) error {
	if c.CreateCallback != nil {
		return c.CreateCallback(object)
	}
	return nil
}

// Update returns mock value
func (c *MockClient) Update(object sdk.Object) error {
	if c.UpdateCallback != nil {
		return c.UpdateCallback(object)
	}
	return nil
}

// Run returns mock values
func (c *MockClient) Run(pod *corev1.Pod, containerIdx int, command []string) (string, string, error) {
	if c.RunCallback != nil {
		return c.RunCallback(pod, containerIdx, command)
	}
	return c.RunStdOut, c.RunStdErr, c.RunErr
}

// List returns mock values
func (c *MockClient) List(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
	if c.ListCallback != nil {
		return c.ListCallback(namespace, into, opts...)
	}
	return nil
}
