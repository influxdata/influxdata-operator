package remote

import (
	"bytes"
	"fmt"
	"io"
	core_v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"path/filepath"
)

type K8sClient struct {
	ClientSet *kubernetes.Clientset
	Config    *rest.Config
}

func NewK8sClient() (*K8sClient, error) {
	var kubeconfig string
	if kubeConfigPath := os.Getenv("KUBECONFIG"); kubeConfigPath != "" {
		kubeconfig = kubeConfigPath // CI process
	} else {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config") // Development environment
	}

	var config *rest.Config

	_, err := os.Stat(kubeconfig)
	if err != nil {
		// In cluster configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		// Out of cluster configuration
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	var client = &K8sClient{ClientSet: clientset, Config: config}
	return client, nil
}

// GetListOfFilesFromK8s gets list of files in path from Kubernetes (recursive)
//func (client *K8sClient) GetListOfFilesFromK8s(path, findType, findName string) ([]string, error) {
//	pSplit := strings.Split(path, "/")
//	if err := validateK8sPath(pSplit); err != nil {
//		return nil, err
//	}
//
//	namespace, podName, containerName, findPath := initK8sVariables(pSplit)
//	command := []string{"find", findPath, "-type", findType, "-name", findName}
//
//	attempts := 3
//	attempt := 0
//	for attempt < attempts {
//		attempt++
//
//		output, stderr, err := client.Exec(namespace, podName, containerName, command, nil)
//		if len(stderr) != 0 {
//			if attempt == attempts {
//				return nil, fmt.Errorf("STDERR: " + (string)(stderr))
//			}
//			sleep(attempt)
//			continue
//		}
//		if err != nil {
//			if attempt == attempts {
//				return nil, err
//			}
//			sleep(attempt)
//			continue
//		}
//
//		lines := strings.Split((string)(output), "\n")
//		var outLines []string
//		for _, line := range lines {
//			if line != "" {
//				outLines = append(outLines, strings.Replace(line, findPath, "", 1))
//			}
//		}
//
//		return outLines, nil
//	}
//
//	return nil, nil
//}

func (client *K8sClient) Exec(namespace, podName, containerName string, command []string, stdin io.Reader) (*bytes.Buffer, *bytes.Buffer, error) {
	clientset, config := client.ClientSet, client.Config

	req := clientset.Core().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	scheme := runtime.NewScheme()
	if err := core_v1.AddToScheme(scheme); err != nil {
		return nil, nil, fmt.Errorf("error adding to scheme: %v", err)
	}

	parameterCodec := runtime.NewParameterCodec(scheme)
	req.VersionedParams(&core_v1.PodExecOptions{
		Command:   command,
		Container: containerName,
		Stdin:     stdin != nil,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}, parameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return nil, nil, fmt.Errorf("error while creating Executor: %v", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("error in Stream: %v", err)
	}

	return &stdout, &stderr, nil
}

//func validateK8sPath(pathSplit []string) error {
//	if len(pathSplit) >= 3 {
//		return nil
//	}
//
//	return fmt.Errorf("illegal path: %s", filepath.Join(pathSplit...))
//}
//
//func initK8sVariables(split []string) (string, string, string, string) {
//	namespace := split[0]
//	pod := split[1]
//	container := split[2]
//	path := getAbsPath(split[3:]...)
//
//	return namespace, pod, container, path
//}
//
//func getAbsPath(path ...string) string {
//	return filepath.Join("/", filepath.Join(path...))
//}
//
//func sleep(seconds int) {
//	time.Sleep(time.Duration(seconds) * time.Second)
//}
