package remote
//
//import (
//	"archive/tar"
//	"bytes"
//	"errors"
//	"fmt"
//	"io"
//	corev1 "k8s.io/api/core/v1"
//	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
//	"k8s.io/apimachinery/pkg/runtime"
//	"k8s.io/client-go/kubernetes"
//	restclient "k8s.io/client-go/rest"
//	"k8s.io/client-go/tools/remotecommand"
//	"log"
//	"net/url"
//	"os"
//	"path"
//	"path/filepath"
//	"strings"
//)
//
//var (
//	errFileSpecDoesntMatchFormat = errors.New("Filespec must match the canonical format: [[namespace/]pod:]file/path")
//	errFileCannotBeEmpty         = errors.New("Filepath can not be empty")
//)
//
//type RemoteCopyOptions struct {
//	Namespace  string
//	PodName    string
//	Command    []string
//
//	Stdin      io.Reader
//	Stdout     io.Writer
//	Stderr     io.Writer
//
//	ClientConfig *restclient.Config
//	Clientset    kubernetes.Interface
//}
//
//func CopyFromPod(src, dest string) error {
//	srcSpec, err := extractFileSpec(src)
//	if err != nil {
//		return err
//	}
//
//	destSpec, err := extractFileSpec(dest)
//	if err != nil {
//		return err
//	}
//
//	errOut := &bytes.Buffer{}
//	reader, outStream := io.Pipe()
//
//	options := &RemoteCopyOptions{
//		Namespace: srcSpec.PodNamespace,
//		PodName:   srcSpec.PodName,
//
//		Stdin: nil,
//		Stdout: outStream,
//		Stderr: errOut,
//
//		Command:  []string{"tar", "cf", "-", srcSpec.File},
//	}
//
//	// run in separate thread... only because that's what they do in the k8s source.
//	go func() {
//		defer outStream.Close()
//		log.Println("Starting tar execution...")
//		err := options.execute()
//		if err != nil {
//			panic(err)
//		}
//	}()
//
//	prefix := path.Clean(srcSpec.File)
//
//	return untarAll(reader, destSpec.File, prefix)
//}
//
//
//func (o *RemoteCopyOptions) execute() error {
//	// Run executes a validated remote execution against a pod.
//	pod, err := o.Clientset.CoreV1().Pods(o.Namespace).Get(o.PodName, metav1.GetOptions{})
//	if err != nil {
//		return err
//	}
//
//	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
//		return fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
//	}
//
//	containerName := pod.Spec.Containers[0].Name
//
//	fn := func() error {
//		scheme := runtime.NewScheme()
//		if err := corev1.AddToScheme(scheme); err != nil {
//			return err
//		}
//
//		restClient, err := restclient.RESTClientFor(o.ClientConfig)
//		if err != nil {
//			return err
//		}
//
//		req := restClient.Post().
//			Resource("pods").
//			Name(pod.Name).
//			Namespace(pod.Namespace).
//			SubResource("exec").
//			Param("container", containerName)
//
//		req.VersionedParams(&corev1.PodExecOptions{
//				Container: containerName,
//				Command:   o.Command,
//				Stdin:     o.Stdin != nil,
//				Stdout:    o.Stdout != nil,
//				Stderr:    o.Stderr != nil,
//				TTY:       false,
//		}, runtime.NewParameterCodec(scheme))
//
//		log.Println("Executing POST to stream TAR data back...")
//		return remoteExecute("POST", req.URL(), o.ClientConfig, o.Stdin, o.Stdout, o.Stderr)
//	}
//
//	if err := fn(); err != nil {
//		return err
//	}
//
//	return nil
//}
//
//func remoteExecute(method string, url *url.URL, config *restclient.Config, stdin io.Reader, stdout, stderr io.Writer) error {
//	exec, err := remotecommand.NewSPDYExecutor(config, method, url)
//	if err != nil {
//		return err
//	}
//	return exec.Stream(remotecommand.StreamOptions{
//		Stdin:             stdin,
//		Stdout:            stdout,
//		Stderr:            stderr,
//		Tty:               false,
//	})
//}
//
//func extractFileSpec(arg string) (fileSpec, error) {
//	if i := strings.Index(arg, ":"); i == -1 {
//		return fileSpec{File: arg}, nil
//	} else if i > 0 {
//		file := arg[i+1:]
//		pod := arg[:i]
//		pieces := strings.Split(pod, "/")
//		if len(pieces) == 1 {
//			return fileSpec{
//				PodName: pieces[0],
//				File:    file,
//			}, nil
//		}
//		if len(pieces) == 2 {
//			return fileSpec{
//				PodNamespace: pieces[0],
//				PodName:      pieces[1],
//				File:         file,
//			}, nil
//		}
//	}
//
//	return fileSpec{}, errFileSpecDoesntMatchFormat
//}
//
//func untarAll(reader io.Reader, destFile, prefix string) error {
//	entrySeq := -1
//
//	// TODO: use compression here?
//	tarReader := tar.NewReader(reader)
//	for {
//		header, err := tarReader.Next()
//		if err != nil {
//			if err != io.EOF {
//				return err
//			}
//			break
//		}
//		entrySeq++
//		mode := header.FileInfo().Mode()
//		outFileName := path.Join(destFile, clean(header.Name[len(prefix):]))
//		baseName := path.Dir(outFileName)
//		if err := os.MkdirAll(baseName, 0755); err != nil {
//			return err
//		}
//		if header.FileInfo().IsDir() {
//			if err := os.MkdirAll(outFileName, 0755); err != nil {
//				return err
//			}
//			continue
//		}
//
//		// handle coping remote file into local directory
//		if entrySeq == 0 && !header.FileInfo().IsDir() {
//			exists, err := dirExists(outFileName)
//			if err != nil {
//				return err
//			}
//			if exists {
//				outFileName = filepath.Join(outFileName, path.Base(clean(header.Name)))
//			}
//		}
//
//		if mode&os.ModeSymlink != 0 {
//			err := os.Symlink(header.Linkname, outFileName)
//			if err != nil {
//				return err
//			}
//		} else {
//			outFile, err := os.Create(outFileName)
//			if err != nil {
//				return err
//			}
//			defer outFile.Close()
//			if _, err := io.Copy(outFile, tarReader); err != nil {
//				return err
//			}
//			if err := outFile.Close(); err != nil {
//				return err
//			}
//		}
//	}
//
//	if entrySeq == -1 {
//		//if no file was copied
//		errInfo := fmt.Sprintf("error: %s no such file or directory", prefix)
//		return errors.New(errInfo)
//	}
//	return nil
//}
//
//// clean prevents path traversals by stripping them out.
//// This is adapted from https://golang.org/src/net/http/fs.go#L74
//func clean(fileName string) string {
//	return path.Clean(string(os.PathSeparator) + fileName)
//}
//
//// dirExists checks if a path exists and is a directory.
//func dirExists(path string) (bool, error) {
//	fi, err := os.Stat(path)
//	if err == nil && fi.IsDir() {
//		return true, nil
//	}
//	if os.IsNotExist(err) {
//		return false, nil
//	}
//	return false, err
//}
