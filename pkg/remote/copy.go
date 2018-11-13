package remote

import (
	"archive/tar"
	"errors"
	"fmt"
	"io"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type fileSpec struct {
	PodNamespace string
	PodName      string
	File         string
}

var (
	errFileSpecDoesntMatchFormat = errors.New("Filespec must match the canonical format: [[namespace/]pod:]file/path")
	errFileCannotBeEmpty         = errors.New("Filepath can not be empty")
)

func (client *K8sClient) CopyFromK8s(src, dest string) error {
	fmt.Printf("CopyFromK8s [%s] -> [%s]\n", src, dest)
	srcSpec, err := extractFileSpec(src)
	if err != nil {
		return err
	}

	fmt.Printf("srcSpec [%s] -> [%s]\n", src, dest)
	destSpec, err := extractFileSpec(dest)
	if err != nil {
		return err
	}

	fmt.Printf("Getting pods ...\n")
	pod, err := client.ClientSet.CoreV1().Pods(srcSpec.PodNamespace).Get(srcSpec.PodName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return fmt.Errorf("cannot exec into a container in a completed pod; current phase is %s", pod.Status.Phase)
	}

	cmd := []string{"tar", "cf", "-", srcSpec.File}
	output, stderr, err := client.Exec(srcSpec.PodNamespace, srcSpec.PodName, pod.Spec.Containers[0].Name, cmd, nil)

	stderrOut := stderr.String()
	if len(stderrOut) > 0 {
		fmt.Println("STDERR: " + stderrOut)
	}

	if err != nil {
		return err
	}

	prefix := path.Clean(srcSpec.File)

	return untarAll(output, destSpec.File, prefix)
}

func extractFileSpec(arg string) (fileSpec, error) {
	if i := strings.Index(arg, ":"); i == -1 {
		return fileSpec{File: arg}, nil
	} else if i > 0 {
		file := arg[i+1:]
		pod := arg[:i]
		pieces := strings.Split(pod, "/")
		if len(pieces) == 1 {
			return fileSpec{
				PodName: pieces[0],
				File:    file,
			}, nil
		}
		if len(pieces) == 2 {
			return fileSpec{
				PodNamespace: pieces[0],
				PodName:      pieces[1],
				File:         file,
			}, nil
		}
	}

	return fileSpec{}, errFileSpecDoesntMatchFormat
}

func untarAll(reader io.Reader, destFile, prefix string) error {
	entrySeq := -1

	// TODO: use compression here?
	tarReader := tar.NewReader(reader)
	for {
		header, err := tarReader.Next()
		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
		entrySeq++
		mode := header.FileInfo().Mode()
		outFileName := path.Join(destFile, clean(header.Name[len(prefix):]))
		baseName := path.Dir(outFileName)
		if err := os.MkdirAll(baseName, 0755); err != nil {
			return err
		}
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(outFileName, 0755); err != nil {
				return err
			}
			continue
		}

		// handle coping remote file into local directory
		if entrySeq == 0 && !header.FileInfo().IsDir() {
			exists, err := dirExists(outFileName)
			if err != nil {
				return err
			}
			if exists {
				outFileName = filepath.Join(outFileName, path.Base(clean(header.Name)))
			}
		}

		if mode&os.ModeSymlink != 0 {
			err := os.Symlink(header.Linkname, outFileName)
			if err != nil {
				return err
			}
		} else {
			outFile, err := os.Create(outFileName)
			if err != nil {
				return err
			}
			defer outFile.Close()
			if _, err := io.Copy(outFile, tarReader); err != nil {
				return err
			}
			if err := outFile.Close(); err != nil {
				return err
			}
		}
	}

	if entrySeq == -1 {
		//if no file was copied
		errInfo := fmt.Sprintf("error: %s no such file or directory", prefix)
		return errors.New(errInfo)
	}
	return nil
}

// dirExists checks if a path exists and is a directory.
func dirExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil && fi.IsDir() {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// clean prevents path traversals by stripping them out.
// This is adapted from https://golang.org/src/net/http/fs.go#L74
func clean(fileName string) string {
	return path.Clean(string(os.PathSeparator) + fileName)
}
