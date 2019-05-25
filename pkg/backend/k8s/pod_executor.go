package k8s

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

// PodExecutor executes a command inside the specified pod and container
type PodExecutor struct {
	client *kubernetes.Clientset
	config *rest.Config
}

// NewPodExecutor returns a new client for executing commands inside
// containers running in kube pods
func NewPodExecutor(k8sCfg *rest.Config) *PodExecutor {
	k8sClient := kubernetes.NewForConfigOrDie(k8sCfg)
	return &PodExecutor{
		config: k8sCfg,
		client: k8sClient,
	}
}

// Run executes a command inside a container inside a pod
func (p *PodExecutor) Run(pod *corev1.Pod, containerIdx int, command []string) (string, string, error) {
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
	request := p.client.CoreV1().RESTClient().Post().
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

	//p.config.Timeout = 15 * time.Second
	exec, err := remotecommand.NewSPDYExecutor(p.config, "POST", request.URL())
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
