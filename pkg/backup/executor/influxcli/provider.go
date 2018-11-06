// Copyright 2018 Oracle and/or its affiliates. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package influxcli

import (
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	utilexec "k8s.io/utils/exec"

	"github.com/dev9/prod/influxdata-operator/pkg/apis/influxdata/v1alpha1"
)

const (
	influxdbCmd = "influxdb"
)

// Executor creates backups using mysqldump.
type Executor struct {
	executor *v1alpha1.InfluxdbBackupExecutor
	execCreds map[string]string
}

// NewExecutor creates a provider capable of creating and restoring backups
func NewExecutor(executor *v1alpha1.InfluxdbBackupExecutor, execCreds map[string]string) (*Executor, error) {
	if len(executor.Database) == 0 {
		return nil, errors.New("No database specified")
	}

	if len(executor.Hostname) == 0 {
		return nil, errors.New("No database specified")
	}

	return &Executor{executor: executor, execCreds: execCreds}, nil
}

// Backup performs a full cluster backup using the mysqldump tool.
func (ex *Executor) Backup(backupDir string, clusterName string) (io.ReadCloser, string, error) {
	exec := utilexec.New()
	mysqldumpPath, err := exec.LookPath(influxdbCmd)
	if err != nil {
		return nil, "", fmt.Errorf("influxdb path: %v", err)
	}

	args := []string{
		"backup",
		"-portable",
		"-database " + ex.executor.Database,
		"-host " + ex.executor.Hostname,
		backupDir,
	}

	cmd := exec.Command(influxdbCmd, args...)

	var mu sync.Mutex
	mu.Lock()
	defer mu.Unlock()

	backupName := fmt.Sprintf("%s.%s.tar.gz", clusterName, time.Now().UTC().Format("20060102150405"))

	pr, pw := io.Pipe()
	zw := gzip.NewWriter(pw)
	cmd.SetStdout(zw)

	go func() {
		glog.V(4).Infof("running cmd: '%s %s'", mysqldumpPath, SanitizeArgs(args, "password"))
		err = cmd.Run()
		zw.Close()
		if err != nil {
			pw.CloseWithError(errors.Wrap(err, "executing backup"))
		} else {
			pw.Close()
		}
	}()

	return pr, backupName, nil
}

// Restore a cluster from a mysqldump.
func (ex *Executor) Restore(content io.ReadCloser) error {
	defer content.Close()

	glog.V(4).Infof("TODO: Restore functionality not implemented. Please Implement.")

	//exec := utilexec.New()
	//mysqlPath, err := exec.LookPath(influxdbCmd)
	//if err != nil {
	//	return fmt.Errorf("mysql path: %v", err)
	//}
	//
	//args := []string{
	//	"-u" + ex.config.username,
	//	"-p" + ex.config.password,
	//}
	//cmd := exec.Command(mysqlPath, args...)
	//
	//var mu sync.Mutex
	//mu.Lock()
	//defer mu.Unlock()
	//
	//zr, err := gzip.NewReader(content)
	//if err != nil {
	//	return err
	//}
	//defer zr.Close()
	//cmd.SetStdin(zr)
	//
	//glog.V(4).Infof("running cmd: '%s %s'", mysqlPath, SanitizeArgs(args, ex.config.password))
	//_, err = cmd.CombinedOutput()
	//return err

	return nil
}

// SanitizeArgs takes a slice, redacts all occurrences of a given string and
// returns a single string concatenated with spaces
func CombineArgs(args []string) string {
	return strings.Join(args, " ")
}
