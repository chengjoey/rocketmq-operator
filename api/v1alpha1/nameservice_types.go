/*
 * Copyright 2022.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *     http://www.apache.org/licenses/LICENSE-2.0
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package v1alpha1

import corev1 "k8s.io/api/core/v1"

// NameServiceSpec defines the desired state of NameService
// +k8s:openapi-gen=true
type NameServiceSpec struct {
	// +kubebuilder:validation:Required
	// Name of the NameService
	Name string `json:"name"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// Size is the number of the name service Pod
	Size int32 `json:"size"`
	// Env of the NameService
	Env []corev1.EnvVar `json:"env,omitempty"`
	//Image is the name service image
	Image string `json:"image"`
	// ImagePullPolicy defines how the image is pulled.
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// HostNetwork can be true or false
	HostNetwork bool `json:"hostNetwork,omitempty"`
	// dnsPolicy defines how a pod's DNS will be configured
	DNSPolicy corev1.DNSPolicy `json:"dnsPolicy,omitempty"`
	// Resources describes the compute resource requirements
	Resources corev1.ResourceRequirements `json:"resources"`
	// StorageMode can be EmptyDir, HostPath, StorageClass
	StorageMode string `json:"storageMode"`
	// HostPath is the local path to store data
	HostPath *string `json:"hostPath,omitempty"`
	// VolumeClaimTemplates defines the StorageClass
	VolumeClaimTemplates []corev1.PersistentVolumeClaim `json:"volumeClaimTemplates"`
	// Pod Security Context
	PodSecurityContext *corev1.PodSecurityContext `json:"securityContext,omitempty"`
	// Container Security Context
	ContainerSecurityContext *corev1.SecurityContext `json:"containerSecurityContext,omitempty"`
	// The secrets used to pull image from private registry
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Affinity the pod's scheduling constraints
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
	// Tolerations the pod's tolerations.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// NodeSelector is a selector which must be true for the pod to fit on a node
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`
	// PriorityClassName indicates the pod's priority
	PriorityClassName string `json:"priorityClassName,omitempty"`
	// ServiceAccountName
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// NameServiceStatus defines the observed state of NameService
// +k8s:openapi-gen=true
type NameServiceStatus struct {
	// NameServers is the name service ip list
	NameServers []string `json:"nameServers"`
	// Running is the number of the running name service Pod
	Running int32 `json:"running"`
	// Status describes the status of the name service
	Status ConditionStatus `json:"status,omitempty"`
}
