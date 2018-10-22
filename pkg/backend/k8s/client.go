package k8s

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

/*
NOTE: When the operator-sdk has been refactored to
allow for unit testing without mockable abstractions
this can all be deleted in favor of using the sdk directly
*/

// Client is currently an interface because this code should be
// totally replaced once the refactors of the operator-sdk is complete
type Client interface {
	Get(into sdk.Object, opts ...sdk.GetOption) error
	List(namespace string, into sdk.Object, opts ...sdk.ListOption) error
	Create(object sdk.Object) error
	Update(object sdk.Object) error
	Run(pod *corev1.Pod, containerIdx int, command []string) (string, string, error)
	Patch(object sdk.Object, pt types.PatchType, patch []byte) (err error)
}
