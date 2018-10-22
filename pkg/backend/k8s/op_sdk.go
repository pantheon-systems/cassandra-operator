package k8s

import (
	"bytes"
	"fmt"
	"k8s.io/apimachinery/pkg/types"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// OperatorSdkClient is a class that allows for the operator-sdk to be injected for unit testing purposes
type OperatorSdkClient struct {
	client *kubernetes.Clientset
	config *rest.Config
}

// NewOperatorSdkClient returns a new client for interacting with the operator-sdk/kubernetes
func NewOperatorSdkClient() *OperatorSdkClient {
	k8sCfg := k8sclient.GetKubeConfig()
	k8sClient := kubernetes.NewForConfigOrDie(k8sCfg)
	return &OperatorSdkClient{
		config: k8sCfg,
		client: k8sClient,
	}
}

// Patch resource in kube
func (c *OperatorSdkClient) Patch(object sdk.Object, pt types.PatchType, patch []byte) error {
	return sdk.Patch(object, pt, patch)
}

// Get resource in kube
func (c *OperatorSdkClient) Get(into sdk.Object, opts ...sdk.GetOption) error {
	err := sdk.Get(into, opts...)
	if k8serrors.IsNotFound(err) {
		return nil
	}
	return err
}

// Create resource in kube
func (c *OperatorSdkClient) Create(object sdk.Object) error {
	return sdk.Create(object)
}

// Update resource in kube
func (c *OperatorSdkClient) Update(object sdk.Object) error {
	return sdk.Update(object)
}

// List resources from kube
func (c *OperatorSdkClient) List(namespace string, into sdk.Object, opts ...sdk.ListOption) error {
	return sdk.List(namespace, into, opts...)
}

// Run executes a command inside a container inside a pod
func (c *OperatorSdkClient) Run(pod *corev1.Pod, containerIdx int, command []string) (string, string, error) {
	containerName := pod.Spec.Containers[containerIdx].Name
	fullCommand := strings.Join(command, " ")
	logrus.Debugf("Executing command `%s` on pod %s/%s:%s",
		fullCommand,
		pod.ObjectMeta.Namespace,
		pod.ObjectMeta.Name,
		containerName)

	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		return "", "", fmt.Errorf("error adding to scheme: %v", err)
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	request := c.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(pod.Name).
		Namespace(pod.Namespace).
		SubResource("exec").
		Param("container", containerName)
	request.VersionedParams(&corev1.PodExecOptions{
		Container: containerName,
		Command:   command,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	//c.config.Timeout = 15 * time.Second
	exec, err := remotecommand.NewSPDYExecutor(c.config, "POST", request.URL())
	if err != nil {
		return "", "", fmt.Errorf("Could not execute command on pod: %v", err)
	}

	formattedString := strings.Join(command[1:], "\n")
	stdin := strings.NewReader(formattedString)
	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	return stdout.String(), stderr.String(), err
}
