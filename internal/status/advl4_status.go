/*
 * Copyright 2019-2020 VMware, Inc.
 * All Rights Reserved.
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*   http://www.apache.org/licenses/LICENSE-2.0
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*/

package status

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	advl4v1alpha1pre1 "github.com/vmware-tanzu/service-apis/apis/v1alpha1pre1"
	avicache "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/cache"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"

	core "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
)

type UpdateGWStatusConditionOptions struct {
	Type    string               // to be casted to the appropriate conditionType
	Status  core.ConditionStatus // True/False/Unknown
	Message string               // extended condition message
	Reason  string               // reason for transition
}

// TODO: handle bulk during bootup
func UpdateGatewayStatusAddress(options []UpdateStatusOptions, bulk bool) {
	for _, option := range options {
		gatewayNSName := strings.Split(option.ServiceMetadata.Gateway, "/")
		gw, err := lib.GetAdvL4Clientset().NetworkingV1alpha1pre1().Gateways(gatewayNSName[0]).Get(gatewayNSName[1], metav1.GetOptions{})
		if err != nil {
			utils.AviLog.Infof("key: %s, msg: unable to find gateway object %s", option.Key, option.ServiceMetadata.Gateway)
			DeleteL4LBStatus(avicache.ServiceMetadataObj{
				NamespaceServiceName: option.ServiceMetadata.NamespaceServiceName,
			}, option.Key)
			continue
		}

		// assuming 1 IP per gateway
		gwStatus := gw.Status.DeepCopy()
		if len(gwStatus.Addresses) > 0 && gwStatus.Addresses[0].Value == option.Vip {
			continue
		}

		gwStatus.Addresses = []advl4v1alpha1pre1.GatewayAddress{{
			Value: option.Vip,
			Type:  advl4v1alpha1pre1.IPAddressType,
		}}
		UpdateGatewayStatusGWCondition(gwStatus, &UpdateGWStatusConditionOptions{
			Type:   "Ready",
			Status: corev1.ConditionTrue,
		})
		UpdateGatewayStatusObject(gw, gwStatus)

		utils.AviLog.Debugf("key: %s, msg: Updating corresponding service %v statuses for gateway %s",
			option.Key, option.ServiceMetadata.NamespaceServiceName, option.ServiceMetadata.Gateway)

		UpdateL4LBStatus([]UpdateStatusOptions{{
			Vip: option.Vip,
			Key: option.Key,
			ServiceMetadata: avicache.ServiceMetadataObj{
				NamespaceServiceName: option.ServiceMetadata.NamespaceServiceName,
			},
		}}, false)
	}

	return
}

func DeleteGatewayStatusAddress(svcMetadataObj avicache.ServiceMetadataObj, key string) error {
	gwNSName := strings.Split(svcMetadataObj.Gateway, "/")
	gw, err := lib.GetAdvL4Clientset().NetworkingV1alpha1pre1().Gateways(gwNSName[0]).Get(gwNSName[1], metav1.GetOptions{})
	if err != nil {
		utils.AviLog.Warnf("key: %s, msg: there was a problem in resetting the gateway address status: %s", key, err)
		return err
	}

	if len(gw.Status.Addresses) == 0 ||
		(len(gw.Status.Addresses) > 0 && gw.Status.Addresses[0].Value == "") {
		return nil
	}

	// assuming 1 IP per gateway
	gwStatus := gw.Status.DeepCopy()
	gwStatus.Addresses = []advl4v1alpha1pre1.GatewayAddress{}
	UpdateGatewayStatusGWCondition(gwStatus, &UpdateGWStatusConditionOptions{
		Type:   "Pending",
		Status: corev1.ConditionTrue,
		Reason: "virtualservice deleted/notfound",
	})
	UpdateGatewayStatusObject(gw, gwStatus)

	utils.AviLog.Infof("key: %s, msg: Successfully reset the address status of gateway: %s", key, svcMetadataObj.Gateway)
	return nil
}

// supported GatewayConditionTypes
// InvalidListeners, InvalidAddress, *Serviceable
func UpdateGatewayStatusGWCondition(gwStatus *advl4v1alpha1pre1.GatewayStatus, updateStatus *UpdateGWStatusConditionOptions) {
	utils.AviLog.Debugf("Updating Gateway status gateway condition %v", utils.Stringify(updateStatus))
	for i, _ := range gwStatus.Conditions {
		if string(gwStatus.Conditions[i].Type) == updateStatus.Type {
			gwStatus.Conditions[i].Status = updateStatus.Status
			gwStatus.Conditions[i].Message = updateStatus.Message
			gwStatus.Conditions[i].Reason = updateStatus.Reason
			gwStatus.Conditions[i].LastTransitionTime = metav1.Now()
		}

		if (updateStatus.Type == "Pending" && string(gwStatus.Conditions[i].Type) == "Ready") ||
			(updateStatus.Type == "Ready" && string(gwStatus.Conditions[i].Type) == "Pending") {
			// if Pending true, mark Ready as false automatically
			// if Ready true, mark Pending as false automatically
			gwStatus.Conditions[i].Status = corev1.ConditionFalse
			gwStatus.Conditions[i].LastTransitionTime = metav1.Now()
			gwStatus.Conditions[i].Message = ""
			gwStatus.Conditions[i].Reason = ""
		}

		if updateStatus.Type == "Ready" {
			UpdateGatewayStatusListenerConditions(gwStatus, "", &UpdateGWStatusConditionOptions{
				Type:   "Ready",
				Status: corev1.ConditionTrue,
			})
		}
	}
}

// supported ListenerConditionType
// PortConflict, InvalidRoutes, UnsupportedProtocol, *Serviceable
// pass portString as empty string for updating status in all ports
func UpdateGatewayStatusListenerConditions(gwStatus *advl4v1alpha1pre1.GatewayStatus, portString string, updateStatus *UpdateGWStatusConditionOptions) {
	utils.AviLog.Debugf("Updating Gateway status listener condition port: %s %v", portString, utils.Stringify(updateStatus))
	for port, condition := range gwStatus.Listeners {
		notFound := true
		if condition.Port == portString || portString == "" {
			for i, portCondition := range condition.Conditions {
				if updateStatus.Type == "Ready" && updateStatus.Type != string(portCondition.Type) {
					gwStatus.Listeners[port].Conditions[i].Status = corev1.ConditionFalse
					gwStatus.Listeners[port].Conditions[i].Message = ""
					gwStatus.Listeners[port].Conditions[i].Reason = ""
				}

				if string(portCondition.Type) == updateStatus.Type {
					gwStatus.Listeners[port].Conditions[i].Status = updateStatus.Status
					gwStatus.Listeners[port].Conditions[i].Message = updateStatus.Message
					gwStatus.Listeners[port].Conditions[i].Reason = updateStatus.Reason
					gwStatus.Listeners[port].Conditions[i].LastTransitionTime = metav1.Now()
					notFound = false
				}
			}

			if notFound {
				gwStatus.Listeners[port].Conditions = append(gwStatus.Listeners[port].Conditions, advl4v1alpha1pre1.ListenerCondition{
					Type:               advl4v1alpha1pre1.ListenerConditionType(updateStatus.Type),
					Status:             updateStatus.Status,
					Reason:             updateStatus.Reason,
					LastTransitionTime: metav1.Now(),
				})
			}
		}
	}

	// in case of a positive error listenerCondition Update we need to mark the
	// gateway Condition back from Ready to Pending
	badTypes := []string{"PortConflict", "InvalidRoutes", "UnsupportedProtocol"}
	if utils.HasElem(badTypes, updateStatus.Type) {
		UpdateGatewayStatusGWCondition(gwStatus, &UpdateGWStatusConditionOptions{
			Type:   "Pending",
			Status: corev1.ConditionTrue,
			Reason: fmt.Sprintf("port %s error %s", portString, updateStatus.Type),
		})
	}
}

func UpdateGatewayStatusObject(gw *advl4v1alpha1pre1.Gateway, updateStatus *advl4v1alpha1pre1.GatewayStatus, retryNum ...int) error {
	retry := 0
	if len(retryNum) > 0 {
		retry = retryNum[0]
		if retry >= 4 {
			return errors.New("msg: UpdateGatewayStatus retried 5 times, aborting")
		}
	}

	if reflect.DeepEqual(gw.Status, *updateStatus) {
		return nil
	}

	patchPayload, _ := json.Marshal(map[string]interface{}{
		"status": updateStatus,
	})

	_, err := lib.GetAdvL4Clientset().NetworkingV1alpha1pre1().Gateways(gw.Namespace).Patch(gw.Name, types.MergePatchType, patchPayload, "status")
	if err != nil {
		utils.AviLog.Warnf("msg: %d there was an error in updating the gateway status: %+v", retry, err)
		updatedGW, err := lib.GetAdvL4Clientset().NetworkingV1alpha1pre1().Gateways(gw.Namespace).Get(gw.Name, metav1.GetOptions{})
		if err != nil {
			utils.AviLog.Warnf("gateway not found %v", err)
			return err
		}
		return UpdateGatewayStatusObject(updatedGW, updateStatus, retry+1)
	}

	utils.AviLog.Infof("msg: Successfully updated the gateway %s/%s status %+v", gw.Namespace, gw.Name, utils.Stringify(updateStatus))
	return nil
}

func InitializeGatewayConditions(gw *advl4v1alpha1pre1.Gateway) error {
	gwStatus := gw.Status.DeepCopy()
	if len(gwStatus.Conditions) > 0 {
		// already initialised
		return nil
	}

	gwStatus.Conditions = []advl4v1alpha1pre1.GatewayCondition{{
		Type:               "Pending",
		Status:             corev1.ConditionTrue,
		LastTransitionTime: metav1.Now(),
	}, {
		Type:               "Ready",
		Status:             corev1.ConditionFalse,
		LastTransitionTime: metav1.Now(),
	}}

	var listenerStatuses []advl4v1alpha1pre1.ListenerStatus
	for _, listener := range gw.Spec.Listeners {
		listenerStatuses = append(listenerStatuses, advl4v1alpha1pre1.ListenerStatus{
			Port: strconv.Itoa(int(listener.Port)),
			Conditions: []advl4v1alpha1pre1.ListenerCondition{{
				Type:               "Ready",
				Status:             corev1.ConditionFalse,
				LastTransitionTime: metav1.Now(),
			}},
		})
	}
	gwStatus.Listeners = listenerStatuses
	if len(gwStatus.Addresses) == 0 {
		gwStatus.Addresses = []advl4v1alpha1pre1.GatewayAddress{}
	}

	return UpdateGatewayStatusObject(gw, gwStatus)
}
