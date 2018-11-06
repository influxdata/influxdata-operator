package v1alpha1

import (
	"fmt"
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/dev9/prod/influxdata-operator/pkg/constants"
)

func validateExecutor(executor BackupExecutor, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(executor.Config.Hostname) == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("config.hostname"), executor, ""))
	}

	if len(executor.Config.Database) == 0 {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("config.database"), executor, ""))
	}

	return allErrs
}

func validateBackup(backup *Backup) field.ErrorList {
	allErrs := field.ErrorList{}
	allErrs = append(allErrs, validateBackupLabels(backup.Labels, field.NewPath("labels"))...)
	allErrs = append(allErrs, validateBackupSpec(backup.Spec, field.NewPath("spec"))...)

	return allErrs
}

func validateBackupLabels(labels map[string]string, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}
	if labels[constants.InfluxdataOperatorVersionLabel] == "" {
		errorStr := fmt.Sprintf("no '%s' present.", constants.InfluxdataOperatorVersionLabel)
		allErrs = append(allErrs, field.Invalid(fldPath, labels, errorStr))
	}
	return allErrs
}

func validateBackupSpec(spec BackupSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateExecutor(spec.Executor, field.NewPath("executor"))...)

	return allErrs
}

func validateRestore(restore *Restore) field.ErrorList {
	allErrs := field.ErrorList{}

	allErrs = append(allErrs, validateRestoreSpec(restore.Spec, field.NewPath("spec"))...)

	value, ok := restore.Labels[constants.InfluxdataOperatorVersionLabel]
	if !ok {
		errorStr := fmt.Sprintf("no '%s' present.", constants.InfluxdataOperatorVersionLabel)
		allErrs = append(allErrs, field.Invalid(field.NewPath("labels"), restore.Labels, errorStr))
	}
	if value == "" {
		errorStr := fmt.Sprintf("empty '%s' present.", constants.InfluxdataOperatorVersionLabel)
		allErrs = append(allErrs, field.Invalid(field.NewPath("labels"), restore.Labels, errorStr))
	}

	return allErrs
}

func validateRestoreSpec(s RestoreSpec, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if s.Cluster == nil || s.Cluster.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("cluster").Child("name"), "a cluster to restore into is required"))
	}

	if s.Backup == nil || s.Backup.Name == "" {
		allErrs = append(allErrs, field.Required(fldPath.Child("backup").Child("name"), "a backup to restore is required"))
	}

	return allErrs
}

