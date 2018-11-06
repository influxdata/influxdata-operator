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

package backup

import (
	"errors"
	"os"

	"github.com/dev9/prod/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	"github.com/dev9/prod/influxdata-operator/pkg/backup/executor"
)

const (
	backupDir  = "/var/lib/influxdb/backup"
	restoreDir = "/var/lib/influxdb/restore"
)

// Runner implementations can execute backups
type Runner interface {
	Backup(clusterName string) (string, error)
	Restore(key string) error
}

type runner struct {
	executor executor.Interface
}

// NewConfiguredRunner creates a runner configured with the Backup/Restore target executor
func NewConfiguredRunner(execConfig v1alpha1.BackupExecutor, execCreds map[string]string) (Runner, error) {
	exec, err := executor.New(execConfig, execCreds)
	if err != nil {
		return nil, err
	}

	return &runner{executor: exec}, nil
}

// Backup performs a backup using the executor
func (r *runner) Backup(clusterName string) (string, error) {
	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		if err := os.MkdirAll(backupDir, os.ModePerm); err != nil {
			return "", err
		}
	}
	defer os.RemoveAll(backupDir)

	_, key, err := r.executor.Backup(backupDir, clusterName)
	if err != nil {
		return "", err
	}

	return key, nil
}

// Restore performs a retrieve using the storage provider then a restore using
// the executor.
func (r *runner) Restore(key string) error {
	return errors.New("restore not implemented")
	//if _, err := os.Stat(restoreDir); os.IsNotExist(err) {
	//	if err := os.MkdirAll(restoreDir, os.ModePerm); err != nil {
	//		return err
	//	}
	//}
	//defer os.RemoveAll(restoreDir)
	//
	//reader, err := r.storage.Retrieve(key)
	//if err != nil {
	//	return err
	//}
	//
	//err = r.executor.Restore(reader)
	//if err != nil {
	//	return err
	//}
	//
	//return nil
}
