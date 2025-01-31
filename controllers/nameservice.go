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

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	rocketmqv1alpha1 "erda.cloud/rocketmq/api/v1alpha1"
	"erda.cloud/rocketmq/pkg/constants"
	"github.com/google/uuid"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *RocketMQReconciler) reconcileNameService(ctx context.Context, rocketMQ *rocketmqv1alpha1.RocketMQ) error {
	logger := log.FromContext(ctx)

	// Check if the statefulSet already exists, if not create a new one
	found := &appsv1.StatefulSet{}
	sts := r.statefulSetForNameService(rocketMQ)
	err := r.Client.Get(ctx, types.NamespacedName{Name: sts.Name, Namespace: rocketMQ.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		err = r.Client.Create(ctx, sts)
		if err != nil {
			logger.Error(err, "Failed to create new StatefulSet for NameService", "StatefulSet.Namespace", sts.Namespace, "StatefulSet.Name", sts.Name)
			return err
		}
		return nil
	} else if err != nil {
		return err
	}

	svc := r.serviceForNameService(rocketMQ)
	err = r.Client.Get(ctx, types.NamespacedName{Name: svc.Name, Namespace: rocketMQ.Namespace}, svc)
	if err != nil && errors.IsNotFound(err) {
		err = r.Client.Create(ctx, svc)
		if err != nil {
			logger.Error(err, "Failed to create new Service for NameService", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
			return err
		}
	} else if err != nil {
		return err
	}

	size := rocketMQ.Spec.NameServiceSpec.Size
	if *found.Spec.Replicas != size {
		found.Spec.Replicas = &size
		err = r.Client.Update(ctx, found)
		logger.Info("Update NameService StatefulSet", "StatefulSet.Namespace", found.Namespace, "StatefulSet.Name", found.Name)
		if err != nil {
			logger.Error(err, "Failed to update StatefulSet for NameService", "StatefulSet.Namespace", found.Namespace, "StatefulSet.Name", found.Name)
			return err
		}
	}
	return r.updateNameServiceStatus(ctx, rocketMQ, found)
}

func (r *RocketMQReconciler) updateNameServiceStatus(ctx context.Context, rocketMQ *rocketmqv1alpha1.RocketMQ, sts *appsv1.StatefulSet) error {
	logger := log.FromContext(ctx)
	logger.Info("Check the NameServers status")

	nameService := rocketMQ.Spec.NameServiceSpec
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labelsForNameService(nameService.Name))
	listOps := &client.ListOptions{Namespace: rocketMQ.Namespace, LabelSelector: labelSelector}
	err := r.Client.List(ctx, podList, listOps)
	if err != nil {
		logger.Error(err, "Failed to list pods", "RocketMQ.Namespace", rocketMQ.Namespace, "RocketMQ.Name", rocketMQ.Name)
		return err
	}

	nameServers := getNameServers(nameService.Name, rocketMQ.Namespace, nameService.Size)
	runningNameServer := getRunningNameServersNum(podList.Items)
	status := getStatusFromSts(sts)
	if !reflect.DeepEqual(nameServers, rocketMQ.Status.NameServiceStatus.NameServers) ||
		runningNameServer != rocketMQ.Status.NameServiceStatus.Running ||
		status != rocketMQ.Status.NameServiceStatus.Status {
		rocketMQ.Status.NameServiceStatus.NameServers = nameServers
		rocketMQ.Status.NameServiceStatus.Running = runningNameServer
		rocketMQ.Status.NameServiceStatus.Status = status
		err = r.Client.Status().Update(ctx, rocketMQ)
		if err != nil {
			logger.Error(err, "Failed to update RocketMQ status")
			return err
		}
	}
	return nil
}

func getRunningNameServersNum(pods []corev1.Pod) int32 {
	var num int32 = 0
	for _, pod := range pods {
		if reflect.DeepEqual(pod.Status.Phase, corev1.PodRunning) {
			num++
		}
	}
	return num
}

func getNameServers(name string, namespace string, size int32) []string {
	var nameServers []string
	for i := int32(0); i < size; i++ {
		nameServers = append(nameServers, fmt.Sprintf("%s-%d.%s.%s.svc.cluster.local:9876", name, i, name, namespace))
	}
	return nameServers
}

func labelsForNameService(name string) map[string]string {
	return map[string]string{"app": name, "name_service_cr": name}
}

func (r *RocketMQReconciler) serviceForNameService(rocketMQ *rocketmqv1alpha1.RocketMQ) *corev1.Service {
	labels := labelsForNameService(rocketMQ.Spec.NameServiceSpec.Name)
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rocketMQ.Spec.NameServiceSpec.Name,
			Namespace: rocketMQ.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name: "namesrv",
					Port: constants.NameServiceMainContainerPort,
					TargetPort: intstr.IntOrString{
						Type:   intstr.Int,
						IntVal: constants.NameServiceMainContainerPort,
					},
				},
			},
			Selector: labels,
		},
	}
	ctrl.SetControllerReference(rocketMQ, svc, r.Scheme)
	return svc
}

func (r *RocketMQReconciler) statefulSetForNameService(rocketMQ *rocketmqv1alpha1.RocketMQ) *appsv1.StatefulSet {
	nameService := rocketMQ.Spec.NameServiceSpec
	labels := labelsForNameService(nameService.Name)

	if strings.EqualFold(nameService.VolumeClaimTemplates[0].Name, "") {
		nameService.VolumeClaimTemplates[0].Name = uuid.New().String()
	}

	dep := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      nameService.Name,
			Namespace: rocketMQ.Namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			Replicas: &nameService.Size,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			ServiceName: nameService.Name,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: nameService.ServiceAccountName,
					Affinity:           nameService.Affinity,
					NodeSelector:       nameService.NodeSelector,
					PriorityClassName:  nameService.PriorityClassName,
					HostNetwork:        nameService.HostNetwork,
					DNSPolicy:          nameService.DNSPolicy,
					ImagePullSecrets:   nameService.ImagePullSecrets,
					Containers: []corev1.Container{
						{
							Resources:       nameService.Resources,
							Image:           nameService.Image,
							Name:            nameService.Name,
							ImagePullPolicy: nameService.ImagePullPolicy,
							Env:             nameService.Env,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: constants.NameServiceMainContainerPort,
									Name:          constants.NameServiceMainContainerPortName,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									MountPath: constants.LogMountPath,
									Name:      nameService.VolumeClaimTemplates[0].Name,
									SubPath:   constants.LogSubPathName,
								},
							},
							SecurityContext: getContainerSecurityContext(&nameService),
						},
					},
				},
			},
			VolumeClaimTemplates: getVolumeClaimTemplates(&nameService),
		},
	}
	// Set RocketMQ instance as the owner and controller
	ctrl.SetControllerReference(rocketMQ, dep, r.Scheme)

	return dep
}

func getVolumeClaimTemplates(nameService *rocketmqv1alpha1.NameServiceSpec) []corev1.PersistentVolumeClaim {
	switch nameService.StorageMode {
	case constants.StorageModeStorageClass:
		return nameService.VolumeClaimTemplates
	case constants.StorageModeEmptyDir, constants.StorageModeHostPath:
		fallthrough
	default:
		return nil
	}
}

func getContainerSecurityContext(nameService *rocketmqv1alpha1.NameServiceSpec) *corev1.SecurityContext {
	var securityContext = corev1.SecurityContext{}
	if nameService.ContainerSecurityContext != nil {
		securityContext = *nameService.ContainerSecurityContext
	}
	return &securityContext
}
