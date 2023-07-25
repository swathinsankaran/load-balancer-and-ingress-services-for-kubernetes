/*
 * Copyright 2023-2024 VMware, Inc.
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

package k8s

import (
	"fmt"
	"regexp"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	gatewayv1beta1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	akogatewayapilib "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/ako-gateway-api/lib"
	akogatewayapiobjects "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/ako-gateway-api/objects"
	akogatewayapistatus "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/ako-gateway-api/status"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

func IsValidGateway(key string, gateway *gatewayv1beta1.Gateway) bool {
	spec := gateway.Spec

	defaultCondition := akogatewayapistatus.NewCondition().
		Type(string(gatewayv1beta1.GatewayConditionAccepted)).
		Reason(string(gatewayv1beta1.GatewayReasonInvalid)).
		Status(metav1.ConditionFalse).
		ObservedGeneration(gateway.ObjectMeta.Generation)

	gatewayStatus := gateway.Status.DeepCopy()

	// has 1 or more listeners
	if len(spec.Listeners) == 0 {
		utils.AviLog.Errorf("key %s, msg: no listeners found in gateway %+v", key, gateway.Name)
		defaultCondition.
			Message("No listeners found").
			SetIn(&gatewayStatus.Conditions)
		akogatewayapistatus.Record(key, gateway, &akogatewayapistatus.Status{GatewayStatus: *gatewayStatus})
		return false
	}

	// has 1 or none addresses
	if len(spec.Addresses) > 1 {
		utils.AviLog.Errorf("key: %s, msg: more than 1 gateway address found in gateway %+v", key, gateway.Name)
		defaultCondition.
			Message("More than one address is not supported").
			SetIn(&gatewayStatus.Conditions)
		akogatewayapistatus.Record(key, gateway, &akogatewayapistatus.Status{GatewayStatus: *gatewayStatus})
		return false
	}

	if len(spec.Addresses) == 1 && *spec.Addresses[0].Type != "IPAddress" {
		utils.AviLog.Errorf("gateway address is not of type IPAddress %+v", gateway.Name)
		defaultCondition.
			Message("Only IPAddress as AddressType is supported").
			SetIn(&gatewayStatus.Conditions)
		akogatewayapistatus.Record(key, gateway, &akogatewayapistatus.Status{GatewayStatus: *gatewayStatus})
		return false
	}

	gatewayStatus.Listeners = make([]gatewayv1beta1.ListenerStatus, len(gateway.Spec.Listeners))

	var invalidListenerCount int
	for index := range spec.Listeners {
		if !isValidListener(key, gateway, gatewayStatus, index) {
			invalidListenerCount++
		}
	}

	if invalidListenerCount > 0 {
		utils.AviLog.Errorf("key: %s, msg: Gateway %s contains %d invalid listeners", key, gateway.Name, invalidListenerCount)
		defaultCondition.
			Type(string(gatewayv1beta1.GatewayReasonAccepted)).
			Reason(string(gatewayv1beta1.GatewayReasonListenersNotValid)).
			Message(fmt.Sprintf("Gateway contains %d invalid listener(s)", invalidListenerCount)).
			SetIn(&gatewayStatus.Conditions)
		akogatewayapistatus.Record(key, gateway, &akogatewayapistatus.Status{GatewayStatus: *gatewayStatus})
		return false
	}

	defaultCondition.
		Reason(string(gatewayv1beta1.GatewayReasonAccepted)).
		Status(metav1.ConditionTrue).
		Message("Gateway configuration is valid").
		SetIn(&gatewayStatus.Conditions)
	akogatewayapistatus.Record(key, gateway, &akogatewayapistatus.Status{GatewayStatus: *gatewayStatus})
	utils.AviLog.Infof("key: %s, msg: Gateway %s is valid", key, gateway.Name)
	return true
}

func isValidListener(key string, gateway *gatewayv1beta1.Gateway, gatewayStatus *gatewayv1beta1.GatewayStatus, index int) bool {

	listener := gateway.Spec.Listeners[index]
	gatewayStatus.Listeners[index].Name = gateway.Spec.Listeners[index].Name
	gatewayStatus.Listeners[index].SupportedKinds = akogatewayapilib.SupportedKinds[listener.Protocol]
	gatewayStatus.Listeners[index].AttachedRoutes = akogatewayapilib.ZeroAttachedRoutes

	defaultCondition := akogatewayapistatus.NewCondition().
		Type(string(gatewayv1beta1.GatewayConditionAccepted)).
		Reason(string(gatewayv1beta1.GatewayReasonListenersNotValid)).
		Status(metav1.ConditionFalse).
		ObservedGeneration(gateway.ObjectMeta.Generation)

	// hostname is not nil or wildcard
	if listener.Hostname == nil || *listener.Hostname == "*" {
		utils.AviLog.Errorf("key: %s, msg: hostname with wildcard found in listener %s", key, listener.Name)
		defaultCondition.
			Message("Hostname not found or Hostname has invalid configuration").
			SetIn(&gatewayStatus.Listeners[index].Conditions)
		return false
	}

	// protocol validation
	if listener.Protocol != gatewayv1beta1.HTTPProtocolType &&
		listener.Protocol != gatewayv1beta1.HTTPSProtocolType {
		utils.AviLog.Errorf("key: %s, msg: protocol is not supported for listener %s", key, listener.Name)
		defaultCondition.
			Reason(string(gatewayv1beta1.ListenerReasonUnsupportedProtocol)).
			Message("Unsupported protocol").
			SetIn(&gatewayStatus.Listeners[index].Conditions)
		gatewayStatus.Listeners[index].SupportedKinds = akogatewayapilib.SupportedKinds[gatewayv1beta1.HTTPSProtocolType]
		return false
	}

	// has valid TLS config
	if listener.TLS != nil {
		if (listener.TLS.Mode != nil && *listener.TLS.Mode != gatewayv1beta1.TLSModeTerminate) || len(listener.TLS.CertificateRefs) == 0 {
			utils.AviLog.Errorf("key: %s, msg: tls mode/ref not valid %+v/%+v", key, gateway.Name, listener.Name)
			defaultCondition.
				Reason(string(gatewayv1beta1.ListenerReasonInvalidCertificateRef)).
				Message("TLS mode or reference not valid").
				SetIn(&gatewayStatus.Listeners[index].Conditions)
			return false
		}
		for _, certRef := range listener.TLS.CertificateRefs {
			//only secret is allowed
			if (certRef.Group != nil && string(*certRef.Group) != "") ||
				certRef.Kind != nil && string(*certRef.Kind) != utils.Secret {
				utils.AviLog.Errorf("CertificateRef is not valid %+v/%+v, must be Secret", gateway.Name, listener.Name)
				defaultCondition.
					Reason(string(gatewayv1beta1.ListenerReasonInvalidCertificateRef)).
					Message("TLS mode or reference not valid").
					SetIn(&gatewayStatus.Listeners[index].Conditions)
				return false
			}

		}
	}

	// Valid listener
	defaultCondition.
		Reason(string(gatewayv1beta1.GatewayReasonAccepted)).
		Status(metav1.ConditionTrue).
		Message("Listener is valid").
		SetIn(&gatewayStatus.Listeners[index].Conditions)
	utils.AviLog.Infof("key: %s, msg: Listener %s/%s is valid", key, gateway.Name, listener.Name)
	return true
}

func IsHTTPRouteValid(key string, obj *gatewayv1beta1.HTTPRoute) bool {

	httpRoute := obj.DeepCopy()
	if len(httpRoute.Spec.ParentRefs) == 0 {
		utils.AviLog.Errorf("key: %s, msg: Parent Reference is empty for the HTTPRoute %s", key, httpRoute.Name)
		return false
	}

	for _, hostname := range httpRoute.Spec.Hostnames {
		if strings.Contains(string(hostname), "*") {
			utils.AviLog.Errorf("key: %s, msg: Wildcard in hostname is not supported for the HTTPRoute %s", key, httpRoute.Name)
			return false
		}
	}

	httpRouteStatus := obj.Status.DeepCopy()
	httpRouteStatus.Parents = make([]gatewayv1beta1.RouteParentStatus, 0, len(httpRoute.Spec.ParentRefs))
	var invalidParentRefCount int
	for index := range httpRoute.Spec.ParentRefs {
		err := validateParentReference(key, httpRoute, httpRouteStatus, index)
		if err != nil {
			invalidParentRefCount++
			parentRefName := httpRoute.Spec.ParentRefs[index].Name
			utils.AviLog.Warnf("key: %s, msg: Parent Reference %s of HTTPRoute object %s is not valid, err: %v", key, parentRefName, httpRoute.Name, err)
		}
	}
	akogatewayapistatus.Record(key, httpRoute, &akogatewayapistatus.Status{HTTPRouteStatus: *httpRouteStatus})

	// No valid attachment, we can't proceed with this HTTPRoute object.
	if invalidParentRefCount == len(httpRoute.Spec.ParentRefs) {
		utils.AviLog.Errorf("key: %s, msg: HTTPRoute object %s is not valid", key, httpRoute.Name)
		return false
	}
	utils.AviLog.Infof("key: %s, msg: HTTPRoute object %s is valid", key, httpRoute.Name)
	return true
}

func validateParentReference(key string, httpRoute *gatewayv1beta1.HTTPRoute, httpRouteStatus *gatewayv1beta1.HTTPRouteStatus, index int) error {

	name := string(httpRoute.Spec.ParentRefs[index].Name)
	namespace := httpRoute.Namespace
	if httpRoute.Spec.ParentRefs[index].Namespace != nil {
		namespace = string(*httpRoute.Spec.ParentRefs[index].Namespace)
	}

	obj, err := akogatewayapilib.AKOControlConfig().GatewayApiInformers().GatewayInformer.Lister().Gateways(namespace).Get(name)
	if err != nil {
		utils.AviLog.Errorf("key: %s, msg: unable to get the gateway object. err: %s", key, err)
		return err
	}
	gateway := obj.DeepCopy()

	gwClass := string(gateway.Spec.GatewayClassName)
	_, isAKOCtrl := akogatewayapiobjects.GatewayApiLister().IsGatewayClassControllerAKO(gwClass)
	if !isAKOCtrl {
		utils.AviLog.Warnf("key: %s, msg: controller for the parent reference %s of HTTPRoute object %s is not ako", key, name, httpRoute.Name)
		return fmt.Errorf("controller for the parent reference %s of HTTPRoute object %s is not ako", name, httpRoute.Name)
	}
	// creates the Parent status only when the AKO is the gateway controller
	httpRouteStatus.Parents = append(httpRouteStatus.Parents, gatewayv1beta1.RouteParentStatus{})
	httpRouteStatus.Parents[index].ControllerName = akogatewayapilib.GatewayController
	httpRouteStatus.Parents[index].ParentRef.Name = gatewayv1beta1.ObjectName(name)
	httpRouteStatus.Parents[index].ParentRef.Namespace = (*gatewayv1beta1.Namespace)(&namespace)

	defaultCondition := akogatewayapistatus.NewCondition().
		Type(string(gatewayv1beta1.GatewayConditionAccepted)).
		Reason(string(gatewayv1beta1.GatewayReasonInvalid)).
		Status(metav1.ConditionFalse).
		ObservedGeneration(httpRoute.ObjectMeta.Generation)

	if httpRoute.Spec.ParentRefs[index].SectionName == nil {
		// can't attach to any so update the httproute status
		utils.AviLog.Errorf("key: %s, msg: Section Name in Parent Reference %s is empty, HTTPRoute object %s cannot be attached to a listener", key, name, httpRoute.Name)
		err := fmt.Errorf("Listener not specified")
		defaultCondition.
			Message(err.Error()).
			SetIn(&httpRouteStatus.Parents[index].Conditions)
		return err
	}

	listenerName := *httpRoute.Spec.ParentRefs[index].SectionName
	httpRouteStatus.Parents[index].ParentRef.SectionName = &listenerName
	i := akogatewayapilib.FindListenerByName(string(listenerName), gateway.Spec.Listeners)
	if i == -1 {
		// listener is not present in gateway
		utils.AviLog.Errorf("key: %s, msg: not able to find the listener from the Section Name %s in Parent Reference %s", key, name, listenerName)
		err := fmt.Errorf("Invalid listener name provided")
		defaultCondition.
			Message(err.Error()).
			SetIn(&httpRouteStatus.Parents[index].Conditions)
		return err
	}

	hostsInListener := gateway.Spec.Listeners[i].Hostname

	// replace the wildcard character with a regex
	replacedHostname := strings.Replace(string(*hostsInListener), "*", "([a-zA-Z0-9-]{1,})", 1)

	// create the expression for pattern matching
	pattern := fmt.Sprintf("^%s$", replacedHostname)
	expr, err := regexp.Compile(pattern)
	if err != nil {
		utils.AviLog.Errorf("key: %s, msg: unable to match the hostname with listener hostname. err: %s", key, err)
		err := fmt.Errorf("Invalid hostname in parent reference")
		defaultCondition.
			Message(err.Error()).
			SetIn(&httpRouteStatus.Parents[index].Conditions)
		return err
	}
	var matched bool
	for _, host := range httpRoute.Spec.Hostnames {
		matched = matched || expr.MatchString(string(host))
	}
	if !matched {
		utils.AviLog.Errorf("key: %s, msg: Gateway object %s don't have any listeners that matches the hostnames in HTTPRoute %s", key, gateway.Name, httpRoute.Name)
		err := fmt.Errorf("Hostname in Gateway Listener doesn't match with any of the hostnames in HTTPRoute")
		defaultCondition.
			Message(err.Error()).
			SetIn(&httpRouteStatus.Parents[index].Conditions)
		return err
	}

	// Increment the attached routes of the listener in the Gateway object
	gatewayStatus := gateway.Status.DeepCopy()
	i = akogatewayapilib.FindListenerStatusByName(string(listenerName), gatewayStatus.Listeners)
	if i == -1 {
		utils.AviLog.Errorf("key: %s, msg: Gateway status is missing for the listener with name %s", key, listenerName)
		err := fmt.Errorf("Couldn't find the listener %s in the Gateway status", listenerName)
		defaultCondition.
			Message(err.Error()).
			SetIn(&httpRouteStatus.Parents[index].Conditions)
		return err
	}

	gatewayStatus.Listeners[index].AttachedRoutes += 1
	akogatewayapistatus.Record(key, gateway, &akogatewayapistatus.Status{GatewayStatus: *gatewayStatus})

	defaultCondition.
		Reason(string(gatewayv1beta1.GatewayReasonAccepted)).
		Status(metav1.ConditionTrue).
		Message("Parent reference is valid").
		SetIn(&httpRouteStatus.Parents[index].Conditions)
	utils.AviLog.Infof("key: %s, msg: Parent Reference %s of HTTPRoute object %s is valid", key, name, httpRoute.Name)
	return nil
}
