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

package executor

import (
	"io"

	"github.com/dev9/prod/influxdata-operator/pkg/apis/influxdata/v1alpha1"
	"github.com/dev9/prod/influxdata-operator/pkg/backup/executor/influxcli"
)

// Interface will execute backup operations via a tool
type Interface interface {
	// Backup runs a backup operation using the given credentials, returning the content.
	Backup(backupDir string, clusterName string) (io.ReadCloser, string, error)
	// Restore restores the given content to the node
	Restore(content io.ReadCloser) error
}

// New builds a new backup executor.
func New(executor v1alpha1.BackupExecutor, execCreds map[string]string) (Interface, error) {
	return influxcli.NewExecutor(executor.Config, execCreds)
}
