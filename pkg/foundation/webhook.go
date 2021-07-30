// Copyright (c) 2020 Red Hat, Inc.
// Copyright Contributors to the Open Cluster Management project

package foundation

import (
	v1alpha1 "github.com/open-cluster-management/backplane-operator/api/v1alpha1"
	"github.com/open-cluster-management/backplane-operator/pkg/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// WebhookName is the name of the foundation webhook deployment
const WebhookName string = "ocm-webhook"

// WebhookDeployment creates the deployment for the foundation webhook
func WebhookDeployment(bpc *v1alpha1.BackplaneConfig, overrides map[string]string) *appsv1.Deployment {
	replicas := utils.GetReplicaCount()
	mode := int32(420)

	dep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WebhookName,
			Namespace: bpc.Namespace,
			Labels:    defaultLabels(WebhookName),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: defaultLabels(WebhookName),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: defaultLabels(WebhookName),
				},
				Spec: corev1.PodSpec{
					// ImagePullSecrets:   []corev1.LocalObjectReference{{Name: bpc.Spec.ImagePullSecret}},
					ServiceAccountName: ServiceAccount,
					Tolerations:        defaultTolerations(),
					// NodeSelector:       bpc.Spec.NodeSelector,
					Affinity: utils.DistributePods("ocm-antiaffinity-selector", WebhookName),
					Volumes: []corev1.Volume{
						{
							Name: "webhook-cert",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{DefaultMode: &mode, SecretName: WebhookName},
							},
						},
					},
					Containers: []corev1.Container{{
						Image:           Image(overrides),
						ImagePullPolicy: utils.GetImagePullPolicy(bpc),
						Name:            WebhookName,
						Args: []string{
							"/webhook",
							"--tls-cert-file=/var/run/ocm-webhook/tls.crt",
							"--tls-private-key-file=/var/run/ocm-webhook/tls.key",
						},
						Ports: []corev1.ContainerPort{{ContainerPort: 8000}},
						LivenessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{"ls"},
								},
							},
							InitialDelaySeconds: 15,
							PeriodSeconds:       15,
						},
						ReadinessProbe: &corev1.Probe{
							Handler: corev1.Handler{
								Exec: &corev1.ExecAction{
									Command: []string{"ls"},
								},
							},
							InitialDelaySeconds: 15,
							PeriodSeconds:       15,
						},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("128Mi"),
								corev1.ResourceCPU:    resource.MustParse("50m"),
							},
							Limits: corev1.ResourceList{
								corev1.ResourceMemory: resource.MustParse("256Mi"),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{Name: "webhook-cert", MountPath: "/var/run/ocm-webhook"},
						},
					}},
				},
			},
		},
	}

	dep.SetOwnerReferences([]metav1.OwnerReference{
		*metav1.NewControllerRef(bpc, bpc.GetObjectKind().GroupVersionKind()),
	})
	return dep
}

// WebhookService creates a service object for the foundation webhook
func WebhookService(bpc *v1alpha1.BackplaneConfig) *corev1.Service {
	s := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      WebhookName,
			Namespace: bpc.Namespace,
			Labels:    defaultLabels(WebhookName),
			Annotations: map[string]string{
				"service.beta.openshift.io/serving-cert-secret-name": WebhookName,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: defaultLabels(WebhookName),
			Ports: []corev1.ServicePort{{
				Port:       443,
				TargetPort: intstr.FromInt(8000),
			}},
		},
	}

	s.SetOwnerReferences([]metav1.OwnerReference{
		*metav1.NewControllerRef(bpc, bpc.GetObjectKind().GroupVersionKind()),
	})
	return s
}