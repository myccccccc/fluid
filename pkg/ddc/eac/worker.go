/*
  Copyright 2022 The Fluid Authors.

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package eac

import (
	"context"
	"encoding/json"
	"fmt"
	datav1alpha1 "github.com/fluid-cloudnative/fluid/api/v1alpha1"
	"github.com/fluid-cloudnative/fluid/pkg/common"
	"github.com/fluid-cloudnative/fluid/pkg/ctrl"
	"github.com/fluid-cloudnative/fluid/pkg/ddc/base"
	"github.com/fluid-cloudnative/fluid/pkg/utils"
	"github.com/fluid-cloudnative/fluid/pkg/utils/kubeclient"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	podutil "k8s.io/kubernetes/pkg/api/v1/pod"
	"reflect"
	"strconv"
)

// getWorkerSelectors gets the selector of the worker
func (e *EACEngine) getWorkerSelectors() string {
	labels := map[string]string{
		"release":   e.name,
		PodRoleType: WOKRER_POD_ROLE,
		"app":       common.EACRuntime,
	}
	labelSelector := &metav1.LabelSelector{
		MatchLabels: labels,
	}

	selectorValue := ""
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		e.Log.Error(err, "Failed to parse the labelSelector of the runtime", "labels", labels)
	} else {
		selectorValue = selector.String()
	}
	return selectorValue
}

// ShouldSetupWorkers checks if we need setup the workers
func (e *EACEngine) ShouldSetupWorkers() (should bool, err error) {
	runtime, err := e.getRuntime()
	if err != nil {
		return
	}

	switch runtime.Status.WorkerPhase {
	case datav1alpha1.RuntimePhaseNone:
		should = true
	default:
		should = false
	}

	return
}

// SetupWorkers checks the desired and current replicas of workers and makes an update
// over the status by setting phases and conditions. The function
// calls for a status update and finally returns error if anything unexpected happens.
func (e *EACEngine) SetupWorkers() (err error) {
	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		workers, err := ctrl.GetWorkersAsStatefulset(e.Client,
			types.NamespacedName{Namespace: e.namespace, Name: e.getWorkerName()})
		if err != nil {
			return err
		}
		runtime, err := e.getRuntime()
		if err != nil {
			return err
		}
		runtimeToUpdate := runtime.DeepCopy()
		if runtimeToUpdate.Replicas() != 0 {
			return e.Helper.SetupWorkers(runtimeToUpdate, runtimeToUpdate.Status, workers)
		} else {
			return e.setupDisabledWorkers(runtimeToUpdate, runtimeToUpdate.Status, workers)
		}
	})
	if err != nil {
		_ = utils.LoggingErrorExceptConflict(e.Log, err, "Failed to setup workers", types.NamespacedName{Namespace: e.namespace, Name: e.name})
		return err
	}
	return
}

func (e *EACEngine) setupDisabledWorkers(runtime base.RuntimeInterface,
	currentStatus datav1alpha1.RuntimeStatus,
	workers *appsv1.StatefulSet) (err error) {

	statusToUpdate := runtime.GetStatus()
	statusToUpdate.WorkerPhase = datav1alpha1.RuntimePhaseNotReady

	statusToUpdate.DesiredWorkerNumberScheduled = runtime.Replicas()
	statusToUpdate.CurrentWorkerNumberScheduled = statusToUpdate.DesiredWorkerNumberScheduled

	if len(statusToUpdate.Conditions) == 0 {
		statusToUpdate.Conditions = []datav1alpha1.RuntimeCondition{}
	}
	cond := utils.NewRuntimeCondition(datav1alpha1.RuntimeWorkersInitialized, datav1alpha1.RuntimeWorkersInitializedReason,
		"The workers are initialized.", corev1.ConditionTrue)
	statusToUpdate.Conditions =
		utils.UpdateRuntimeCondition(statusToUpdate.Conditions,
			cond)

	status := *statusToUpdate
	if !reflect.DeepEqual(status, currentStatus) {
		return e.Client.Status().Update(context.TODO(), runtime)
	}

	return
}

// are the workers ready
func (e *EACEngine) CheckWorkersReady() (ready bool, err error) {
	workers, err := ctrl.GetWorkersAsStatefulset(e.Client,
		types.NamespacedName{Namespace: e.namespace, Name: e.getWorkerName()})
	if err != nil {
		return ready, err
	}

	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		runtime, err := e.getRuntime()
		if err != nil {
			return err
		}
		runtimeToUpdate := runtime.DeepCopy()
		if runtimeToUpdate.Replicas() != 0 {
			ready, err = e.Helper.CheckWorkersReady(runtimeToUpdate, runtimeToUpdate.Status, workers)
		} else {
			ready, err = e.checkDisabledWorkersReady(runtimeToUpdate, runtimeToUpdate.Status, workers)
		}
		if err != nil {
			_ = utils.LoggingErrorExceptConflict(e.Log, err, "Failed to check worker ready", types.NamespacedName{Namespace: e.namespace, Name: e.name})
		}
		return err
	})

	return
}

func (e *EACEngine) checkDisabledWorkersReady(runtime base.RuntimeInterface,
	currentStatus datav1alpha1.RuntimeStatus,
	workers *appsv1.StatefulSet) (ready bool, err error) {
	var (
		phase datav1alpha1.RuntimePhase     = datav1alpha1.RuntimePhaseReady
		cond  datav1alpha1.RuntimeCondition = datav1alpha1.RuntimeCondition{}
	)
	ready = true

	statusToUpdate := runtime.GetStatus()
	statusToUpdate.WorkerPhase = phase
	if len(statusToUpdate.Conditions) == 0 {
		statusToUpdate.Conditions = []datav1alpha1.RuntimeCondition{}
	}
	cond = utils.NewRuntimeCondition(datav1alpha1.RuntimeWorkersReady, datav1alpha1.RuntimeWorkersReadyReason,
		"The workers are ready.", corev1.ConditionTrue)
	statusToUpdate.Conditions =
		utils.UpdateRuntimeCondition(statusToUpdate.Conditions,
			cond)

	if !reflect.DeepEqual(currentStatus, statusToUpdate) {
		err = e.Client.Status().Update(context.TODO(), runtime)
	}
	return
}

func (e *EACEngine) syncWorkersEndpoints() (err error) {
	configMapName := fmt.Sprintf("%s-worker-endpoints", e.name)
	configMap, err := kubeclient.GetConfigmapByName(e.Client, configMapName, e.namespace)
	if err != nil {
		return err
	}

	workerPods, err := e.getWorkerPods()
	if err != nil {
		return err
	}

	workersEndpoints := WorkerEndPoints{}
	for _, pod := range workerPods {
		if !podutil.IsPodReady(&pod) {
			continue
		}
		for _, container := range pod.Spec.Containers {
			if container.Name == "eac-worker" {
				for _, port := range container.Ports {
					if port.Name == "rpc" {
						workersEndpoints.ContainerEndPoints = append(workersEndpoints.ContainerEndPoints, pod.Status.PodIP+":"+strconv.Itoa(int(port.ContainerPort)))
					}
				}
			}
		}
	}

	b, _ := json.Marshal(workersEndpoints)
	e.Log.Info("Sync worker endpoints", "worker-endpoints", string(b))

	err = retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		configMapToUpdate := configMap.DeepCopy()
		configMapToUpdate.Data["eac-worker-endpoints.json"] = string(b)
		if !reflect.DeepEqual(configMapToUpdate, configMap) {
			err = e.Client.Update(context.TODO(), configMapToUpdate)
			if err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
