package v1alpha1

import (
	"github.com/dev9/prod/influxdata-operator/pkg/constants"
	"github.com/dev9/prod/influxdata-operator/version"
)

// setOperatorVersionLabel sets the specified operator version label on the label map.
func setOperatorVersionLabel(labelMap map[string]string, label string) {
	labelMap[constants.InfluxdataOperatorVersionLabel] = label
}

// getOperatorVersionLabel get the specified operator version label on the label map.
func getOperatorVersionLabel(labelMap map[string]string) string {
	return labelMap[constants.InfluxdataOperatorVersionLabel]
}

// EnsureDefaults will ensure that if a user omits any fields in the
// spec that are required, we set some sensible defaults.


// EnsureDefaults can be invoked to ensure the default values are present.
func (b Backup) EnsureDefaults() *Backup {
	buildVersion := version.GetBuildVersion()
	if buildVersion != "" {
		if b.Labels == nil {
			b.Labels = make(map[string]string)
		}
		_, hasKey := b.Labels[constants.InfluxdataOperatorVersionLabel]
		if !hasKey {
			setOperatorVersionLabel(b.Labels, buildVersion)
		}
	}
	return &b
}

// Validate checks if the resource spec is valid.
func (b Backup) Validate() error {
	return validateBackup(&b).ToAggregate()
}

// EnsureDefaults can be invoked to ensure the default values are present.
func (r Restore) EnsureDefaults() *Restore {
	buildVersion := version.GetBuildVersion()
	if buildVersion != "" {
		if r.Labels == nil {
			r.Labels = make(map[string]string)
		}
		_, hasKey := r.Labels[constants.InfluxdataOperatorVersionLabel]
		if !hasKey {
			setOperatorVersionLabel(r.Labels, buildVersion)
		}
	}
	return &r
}

// Validate checks if the resource spec is valid.
func (r Restore) Validate() error {
	return validateRestore(&r).ToAggregate()
}
