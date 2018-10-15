package providers

import (
	"fmt"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"strconv"
)

const (
	nfsDepName           = "nfs-provisioner"
	namespaceForNFS      = "NFS_NAMESPACE"
	ownerRefName         = "OWNER_REFERENCE_NAME"
	isRbacEnabled        = "RBAC_ENABLED"
	nfsServiceAccountEnv = "NFS_SERVICE_ACCOUNT_NAME"
)

// SetUpNfsProvisioner sets up a deployment a pvc and a service to handle nfs workload
func SetUpNfsProvisioner(pv *v1.PersistentVolumeClaim) error {
	logrus.Info("Creating new PersistentVolumeClaim for Nfs provisioner..")

	const volumeName = "nfs-prov-volume"

	nfsNamespace := os.Getenv(namespaceForNFS)
	plusStorage, _ := resource.ParseQuantity("2Gi")
	parsedStorageSize := pv.Spec.Resources.Requests["storage"]
	parsedStorageSize.Add(plusStorage)
	isLocalVolume := false

	ownerRef := make([]metav1.OwnerReference, 0)

	if os.Getenv(ownerRefName) != "" {
		ownerRef = []metav1.OwnerReference{asOwner(getOwner())}
	}

	if pv.Annotations["localvolume"] == "true" {
		isLocalVolume = true
	}
	if !isLocalVolume {
		nfsPvc := &v1.PersistentVolumeClaim{
			TypeMeta: metav1.TypeMeta{
				Kind:       "PersistentVolumeClaim",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%s-data", *pv.Spec.StorageClassName),
				Namespace: nfsNamespace,
			},
			Spec: v1.PersistentVolumeClaimSpec{
				AccessModes: []v1.PersistentVolumeAccessMode{
					v1.ReadWriteOnce,
				},
				Resources: v1.ResourceRequirements{
					Requests: v1.ResourceList{
						"storage": parsedStorageSize,
					},
				},
			},
		}

		if len(ownerRef) != 0 {
			nfsPvc.SetOwnerReferences(ownerRef)
		}

		err := sdk.Create(nfsPvc)
		if err != nil && !errors.IsAlreadyExists(err) {
			logrus.Errorf("Error happened during creating a PersistentVolumeClaim for Nfs %s", err.Error())
			return err
		}
	}
	logrus.Info("Creating new Service for Nfs provisioner..")
	nfsSvc := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: nfsDepName,
			Labels: map[string]string{
				"app": nfsDepName,
			},
			Namespace: nfsNamespace,
		},
		Spec: v1.ServiceSpec{
			Ports: []v1.ServicePort{
				{Name: "nfs", Port: 2049},
				{Name: "mountd", Port: 20048},
				{Name: "rpcbind", Port: 111},
				{Name: "rpcbind-udp", Port: 111, Protocol: "UDP"},
			},
			Selector: map[string]string{
				"app": nfsDepName,
			},
		},
	}
	if len(ownerRef) != 0 {
		nfsSvc.SetOwnerReferences(ownerRef)
	}

	err := sdk.Create(nfsSvc)
	if err != nil && !errors.IsAlreadyExists(err) {
		logrus.Errorf("Error happened during creating the Service for Nfs %s", err.Error())
		return err
	}
	logrus.Info("Creating new Deployment for Nfs provisioner..")
	replicas := int32(1)
	nfsDepl := &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfsDepName,
			Namespace: nfsNamespace,
		},
		Spec: v1beta1.DeploymentSpec{
			Replicas: &replicas,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": nfsDepName,
					},
				},
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Name:  nfsDepName,
							Image: "quay.io/kubernetes_incubator/nfs-provisioner:v1.0.9",
							Ports: []v1.ContainerPort{
								{Name: "nfs", ContainerPort: 2049},
								{Name: "mountd", ContainerPort: 20048},
								{Name: "rpcbind", ContainerPort: 111},
								{Name: "rpcbind-udp", ContainerPort: 111, Protocol: "UDP"},
							},
							SecurityContext: &v1.SecurityContext{
								Capabilities: &v1.Capabilities{
									Add: []v1.Capability{
										"DAC_READ_SEARCH",
										"SYS_RESOURCE",
									},
								},
							},
							Args: []string{
								"-provisioner=influxdata.com/nfs",
							},
							Env: []v1.EnvVar{
								{Name: "POD_IP", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "status.podIP"}}},
								{Name: "SERVICE_NAME", Value: nfsDepName},
								{Name: "POD_NAMESPACE", ValueFrom: &v1.EnvVarSource{FieldRef: &v1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}},
							},
							VolumeMounts: []v1.VolumeMount{
								{Name: volumeName, MountPath: "/export"},
							},
							Resources: v1.ResourceRequirements{
								Requests: v1.ResourceList{v1.ResourceCPU: resource.MustParse("1")},
							},
						},
					},
				},
			},
			Strategy: v1beta1.DeploymentStrategy{
				Type: v1beta1.RecreateDeploymentStrategyType,
			},
		},
	}
	if isLocalVolume {
		nfsDepl.Spec.Template.Spec.Volumes = []v1.Volume{{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: "/tmp",
				},
			},
		}}
	} else {
		nfsDepl.Spec.Template.Spec.Volumes = []v1.Volume{{
			Name: volumeName,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: fmt.Sprintf("%s-data", *pv.Spec.StorageClassName),
				}}}}
	}
	if len(ownerRef) != 0 {
		nfsDepl.SetOwnerReferences(ownerRef)
	}
	rbacEnabled, err := strconv.ParseBool(os.Getenv(isRbacEnabled))
	if err != nil {
		return err
	}
	if rbacEnabled {
		if serviceAcc := os.Getenv(nfsServiceAccountEnv); serviceAcc != "" {
			nfsDepl.Spec.Template.Spec.ServiceAccountName = serviceAcc
		}
	}

	err = sdk.Create(nfsDepl)
	if err != nil && !errors.IsAlreadyExists(err) {
		logrus.Errorf("Error happened during creating the Deployment for Nfs %s", err.Error())
		return err
	}
	logrus.Info("Creating new StorageClass for Nfs provisioner..")
	reclaimPolicy := v1.PersistentVolumeReclaimRetain
	nfsStorageClass := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageClass",
			APIVersion: "storage.k8s.io/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: *pv.Spec.StorageClassName,
		},
		ReclaimPolicy: &reclaimPolicy,
		Provisioner:   "influxdata.com/nfs",
	}
	if len(ownerRef) != 0 {
		nfsStorageClass.SetOwnerReferences(ownerRef)
	}
	err = sdk.Create(nfsStorageClass)
	if err != nil && !errors.IsAlreadyExists(err) {
		logrus.Errorf("Error happened during creating the StorageClass for Nfs %s", err.Error())
		return err
	}
	return nil
}

// CheckNfsServerExistence checks if the NFS deployment and all companion service exists
func CheckNfsServerExistence(name, namespace string) bool {
	if !CheckPersistentVolumeClaimExistence(fmt.Sprintf("%s-data", name), namespace) {
		logrus.Info("PersistentVolume claim for Nfs does not exists!")
		return false
	}
	if !checkNfsProviderDeployment() {
		logrus.Info("Nfs provider deployment does not exists!")
		return false
	}
	if !CheckStorageClassExistence(name) {
		logrus.Info("StorageClass for Nfs does not exist!")
		return false
	}
	return true

}

// checkNfsProviderDeployment checks if the NFS deployment exists
func checkNfsProviderDeployment() bool {
	nfsNamespace := os.Getenv(namespaceForNFS)
	deployment := &v1beta1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfsDepName,
			Namespace: nfsNamespace,
		},
		Spec: v1beta1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: []v1.Container{
						{
							Args: []string{
								"-provisioner=influxdata.com/nfs",
							},
						},
					},
				},
			},
		},
	}
	service := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfsDepName,
			Namespace: nfsNamespace,
		},
		Spec: v1.ServiceSpec{
			Selector: map[string]string{"app": nfsDepName},
		},
	}
	if err := sdk.Get(deployment); err != nil {
		logrus.Infof("Nfs provider deployment does not exists %s", err.Error())
		return false
	}
	if err := sdk.Get(service); err != nil {
		logrus.Infof("Nfs provider service does not exists %s", err.Error())
		return false
	}
	logrus.Info("Nfs provider exists!")
	return true
}
