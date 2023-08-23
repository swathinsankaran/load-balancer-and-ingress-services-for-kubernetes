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

package objects

import (
	"strings"
	"sync"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/objects"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

var gwLister *GWLister
var gwonce sync.Once

func GatewayApiLister() *GWLister {
	gwonce.Do(func() {
		gwLister = &GWLister{
			gatewayClassStore:          objects.NewObjectMapStore(),
			gatewayToGatewayClassStore: objects.NewObjectMapStore(),
			gatewayClassToGatewayStore: objects.NewObjectMapStore(),
			routeToGateway:             objects.NewObjectMapStore(),
			gatewayToRoute:             objects.NewObjectMapStore(),
			serviceToGateway:           objects.NewObjectMapStore(),
			gatewayToService:           objects.NewObjectMapStore(),
			serviceToRoute:             objects.NewObjectMapStore(),
			routeToService:             objects.NewObjectMapStore(),
			secretToListener:           objects.NewObjectMapStore(),
			gatewayToSecret:            objects.NewObjectMapStore(),
			routeToChildVS:             objects.NewObjectMapStore(),
			gatewayToListenerStore:     objects.NewObjectMapStore(),
			gatewayListenerToHostname:  objects.NewObjectMapStore(),
			gatewayListenerToRoute:     objects.NewObjectMapStore(),
			routeToGatewayListener:     objects.NewObjectMapStore(),
			gatewayRouteToHostname:     objects.NewObjectMapStore(),
		}
	})
	return gwLister
}

type GWLister struct {
	gwLock sync.RWMutex

	//Gateways with AKO as controller
	gatewayClassStore *objects.ObjectMapStore

	//Namespace/Gateway -> GatewayClass
	gatewayToGatewayClassStore *objects.ObjectMapStore

	//GatewayClass -> [ns1/gateway1, ns2/gateway2, ...]
	gatewayClassToGatewayStore *objects.ObjectMapStore

	// routeType/routeNs/routeName -> [namespace/gateway, ...]
	routeToGateway *objects.ObjectMapStore

	// namespace/gateway -> [routeType/routeNs/routeName, ...]
	gatewayToRoute *objects.ObjectMapStore

	// serviceNs/serviceName -> [namespace/gateway, ...]
	serviceToGateway *objects.ObjectMapStore

	// namespace/gateway -> [serviceNs/serviceName, ...]
	gatewayToService *objects.ObjectMapStore

	// serviceNs/serviceName -> [routeType/routeNs/routeName, ...]
	serviceToRoute *objects.ObjectMapStore

	// routeType/routeNs/routeName -> [serviceNs/serviceName, ...]
	routeToService *objects.ObjectMapStore

	// secretNs/secretName -> [namespace/gateway: listener, ...]
	secretToListener *objects.ObjectMapStore

	// namespace/gateway/listener -> [secretNs/secretName, ...]
	gatewayToSecret *objects.ObjectMapStore

	// routeType/routeNs/routeName -> [childvs, ...]
	routeToChildVS *objects.ObjectMapStore

	// Namespace/Gateway -> [listener1, listener2, ...]
	gatewayToListenerStore *objects.ObjectMapStore

	// Namespace/Gateway/Listner -> hostname
	gatewayListenerToHostname *objects.ObjectMapStore

	// namespace/gateway/listener -> routeType/routeNs/routeName
	gatewayListenerToRoute *objects.ObjectMapStore

	// routeType/routeNs/routeName -> [namespace/gateway/listener, ...]
	routeToGatewayListener *objects.ObjectMapStore

	//gatewayNS/gatewayName -> [hostname, ...]
	gatewayRouteToHostname *objects.ObjectMapStore

	//svc -> gw
	//route <-> gw
	//secret -> gw
}

func (g *GWLister) GetGatewayToRoutes(gwNsName string) []string {
	var routes []string
	_, listeners := g.gatewayToListenerStore.Get(gwNsName)
	for _, listener := range listeners.([]string) {
		listenerSlice := strings.Split(listener, "/")

		found, route := g.gatewayListenerToRoute.Get(gwNsName + "/" + listenerSlice[0])
		if found {
			if !utils.HasElem(routes, route) {
				routes = append(routes, route.(string))
			}
		}
	}
	return routes
}

func (g *GWLister) UpdateGatewayRouteToHostname(ns, gw string, hostnames []string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	key := getKeyForGateway(ns, gw)
	g.gatewayRouteToHostname.AddOrUpdate(key, hostnames)

}
func (g *GWLister) GetGatewayRouteToHostname(ns, gw string) (bool, []string) {
	key := getKeyForGateway(ns, gw)
	found, hostnames := g.gatewayRouteToHostname.Get(key)
	if found {
		return true, hostnames.([]string)
	}
	return false, make([]string, 0)
}

func (g *GWLister) UpdateRouteToGateway(routeTypeNsName string, gateways []string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.routeToGateway.AddOrUpdate(routeTypeNsName, gateways)

}
func (g *GWLister) IsGatewayClassControllerAKO(gwClass string) (bool, bool) {
	g.gwLock.RLock()
	defer g.gwLock.RUnlock()

	found, isAkoCtrl := g.gatewayClassStore.Get(gwClass)
	if found {
		return true, isAkoCtrl.(bool)
	}
	return false, false
}

func (g *GWLister) UpdateGatewayClass(gwClass string, isAkoCtrl bool) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()
	found, _ := g.gatewayClassToGatewayStore.Get(gwClass)
	if !found {
		g.gatewayClassToGatewayStore.AddOrUpdate(gwClass, make([]string, 0))
	}
	g.gatewayClassStore.AddOrUpdate(gwClass, isAkoCtrl)
}

func (g *GWLister) DeleteGatewayClass(gwClass string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.gatewayClassStore.Delete(gwClass)
}

func (g *GWLister) IsGatewayInStore(gwNsName string) bool {
	gateways := g.gatewayToListenerStore.GetAllKeys()
	return utils.HasElem(gateways, gwNsName)
}

func (g *GWLister) GetGatewayClassToGateway(gwClass string) []string {

	found, gatewayList := g.gatewayClassToGatewayStore.Get(gwClass)
	if !found {
		return make([]string, 0)
	}
	return gatewayList.([]string)
}

func (g *GWLister) GetGatewayToGatewayClass(ns, gw string) string {

	key := getKeyForGateway(ns, gw)
	_, gwClass := g.gatewayToGatewayClassStore.Get(key)
	return gwClass.(string)
}

func (g *GWLister) UpdateGatewayToGatewayClass(ns, gw, gwClass string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	key := getKeyForGateway(ns, gw)

	//remove gateway from old class list
	if found, oldGwClass := g.gatewayToGatewayClassStore.Get(key); found {
		oldGwClassObj := oldGwClass.(string)
		if ok, gatewayList := g.gatewayClassToGatewayStore.Get(oldGwClassObj); ok {
			gatewayListObj := gatewayList.([]string)
			gatewayListObj = utils.Remove(gatewayListObj, gw)
			g.gatewayClassToGatewayStore.AddOrUpdate(oldGwClassObj, gatewayListObj)
		}
	}
	g.gatewayToGatewayClassStore.AddOrUpdate(key, gwClass)
	found, gatewayList := g.gatewayClassToGatewayStore.Get(gwClass)
	if !found {
		gatewayList = make([]string, 0)
	}
	gatewayListObj := gatewayList.([]string)
	if !utils.HasElem(gatewayListObj, key) {
		gatewayListObj = append(gatewayListObj, key)
		g.gatewayClassToGatewayStore.AddOrUpdate(gwClass, gatewayListObj)
	}
}

func (g *GWLister) UpdateGatewayToListener(gwNsName string, listeners []string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.gatewayToListenerStore.AddOrUpdate(gwNsName, listeners)
}
func (g *GWLister) GetGatewayListenerToHostname(ns, gw, listner string) string {
	key := getKeyForGateway(ns, gw) + "/" + listner
	_, obj := g.gatewayListenerToHostname.Get(key)

	return obj.(string)
}
func (g *GWLister) UpdateGatewayListenerToHostname(gwListenerNsName, hostname string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.gatewayListenerToHostname.AddOrUpdate(gwListenerNsName, hostname)
}

func (g *GWLister) GetGatewayToListeners(ns, gw string) []string {
	key := getKeyForGateway(ns, gw)

	_, listenerList := g.gatewayToListenerStore.Get(key)
	return listenerList.([]string)

}

func (g *GWLister) DeleteGatewayToGatewayClass(ns, gw string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	key := getKeyForGateway(ns, gw)
	found, gwClass := g.gatewayToGatewayClassStore.Get(key)
	if found {
		g.gatewayToGatewayClassStore.Delete(key)
		g.gatewayToListenerStore.Delete(key)
		_, gatewayList := g.gatewayClassToGatewayStore.Get(gwClass.(string))
		gatewayListObj := gatewayList.([]string)
		gatewayListObj = utils.Remove(gatewayListObj, key)
		g.gatewayClassToGatewayStore.AddOrUpdate(gwClass.(string), gatewayListObj)
	}
}

func getKeyForGateway(ns, gw string) string {
	return ns + "/" + gw
}

//=====All route <-> gateway mappings go here.

func (g *GWLister) GetRouteToGateway(routeTypeNsName string) (bool, []string) {
	if found, obj := g.routeToGateway.Get(routeTypeNsName); found {
		return true, obj.([]string)
	}
	return false, []string{}
}

func (g *GWLister) DeleteRouteToGateway(routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.routeToGateway.Delete(routeTypeNsName)
}

func (g *GWLister) GetGatewayToRoute(gwNsName string) (bool, []string) {
	if found, obj := g.gatewayToRoute.Get(gwNsName); found {
		return true, obj.([]string)
	}
	return false, []string{}
}

func (g *GWLister) DeleteGatewayToRoute(gwNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.gatewayToRoute.Delete(gwNsName)
}

func (g *GWLister) DeleteGatewayListenerToRoute(gwNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.gatewayListenerToRoute.Delete(gwNsName)
}

func (g *GWLister) UpdateGatewayRouteMappings(gwNsName, routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// update route to gateway mapping
	_, gwNsNameList := g.GetRouteToGateway(routeTypeNsName)
	if !utils.HasElem(gwNsNameList, gwNsName) {
		gwNsNameList = append(gwNsNameList, gwNsName)
		g.routeToGateway.AddOrUpdate(routeTypeNsName, gwNsNameList)
	}

	// update gateway to route mapping
	_, routeTypeNsNameList := g.GetGatewayToRoute(gwNsName)
	if !utils.HasElem(routeTypeNsNameList, routeTypeNsName) {
		routeTypeNsNameList = append(routeTypeNsNameList, routeTypeNsName)
		g.gatewayToRoute.AddOrUpdate(gwNsName, routeTypeNsNameList)
	}
}

func (g *GWLister) DeleteGatewayRouteMappings(gwNsName, routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// delete gateway to route mapping
	_, routeTypeNsNameList := g.GetGatewayToRoute(gwNsName)
	routeTypeNsNameList = utils.Remove(routeTypeNsNameList, routeTypeNsName)
	g.gatewayToRoute.AddOrUpdate(gwNsName, routeTypeNsNameList)

	// delete route to gateway mapping
	_, gwNsNameList := g.GetRouteToGateway(routeTypeNsName)
	gwNsNameList = utils.Remove(gwNsNameList, gwNsName)
	g.routeToGateway.AddOrUpdate(routeTypeNsName, gwNsNameList)
}

//=====All gateway <-> service mappings go here.

func (g *GWLister) GetGatewayToService(gwNsName string) (bool, []string) {
	if found, obj := g.gatewayToService.Get(gwNsName); found {
		return true, obj.([]string)
	}
	return false, []string{}
}

func (g *GWLister) DeleteGatewayToService(gwNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.gatewayToService.Delete(gwNsName)
}

func (g *GWLister) GetServiceToGateway(svcNsName string) (bool, []string) {
	if found, obj := g.serviceToGateway.Get(svcNsName); found {
		return true, obj.([]string)
	}
	return false, []string{}
}

func (g *GWLister) DeleteServiceToGateway(svcNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.serviceToGateway.Delete(svcNsName)
}

func (g *GWLister) UpdateGatewayServiceMappings(gwNsName, svcNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// update gateway to service mapping
	_, svcNsNameList := g.GetGatewayToService(gwNsName)
	if !utils.HasElem(svcNsNameList, svcNsName) {
		svcNsNameList = append(svcNsNameList, svcNsName)
		g.gatewayToService.AddOrUpdate(gwNsName, svcNsNameList)
	}
	// update service to gateway mapping
	_, gwNsNameList := g.GetServiceToGateway(gwNsName)
	if !utils.HasElem(gwNsNameList, gwNsName) {
		gwNsNameList = append(gwNsNameList, gwNsName)
		g.serviceToGateway.AddOrUpdate(svcNsName, gwNsNameList)
	}
}

func (g *GWLister) DeleteGatewayToServiceMappings(gwNsName, svcNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// delete service to gateway mapping
	_, gwNsNameList := g.GetServiceToGateway(gwNsName)
	gwNsNameList = utils.Remove(gwNsNameList, gwNsName)
	g.serviceToGateway.AddOrUpdate(svcNsName, gwNsNameList)

	// delete gateway to service mapping
	_, svcNsNameList := g.GetGatewayToService(gwNsName)
	svcNsNameList = utils.Remove(svcNsNameList, svcNsName)
	g.gatewayToService.AddOrUpdate(gwNsName, svcNsNameList)
}

//=====All route <-> service mappings go here.

func (g *GWLister) GetRouteToService(routeTypeNsName string) (bool, []string) {
	if found, obj := g.routeToService.Get(routeTypeNsName); found {
		return true, obj.([]string)
	}
	return false, []string{}
}

func (g *GWLister) DeleteRouteToService(routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.routeToService.Delete(routeTypeNsName)
}

func (g *GWLister) GetServiceToRoute(svcNsName string) (bool, []string) {
	if found, obj := g.serviceToRoute.Get(svcNsName); found {
		return true, obj.([]string)
	}
	return false, []string{}
}

func (g *GWLister) DeleteServiceToRoute(svcNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.serviceToRoute.Delete(svcNsName)
}

func (g *GWLister) UpdateRouteServiceMappings(routeTypeNsName, svcNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// update route to service mapping
	_, svcNsNameList := g.GetRouteToService(routeTypeNsName)
	if !utils.HasElem(svcNsNameList, svcNsName) {
		svcNsNameList = append(svcNsNameList, svcNsName)
		g.routeToService.AddOrUpdate(routeTypeNsName, svcNsNameList)
	}

	// update service to route mapping
	_, routeTypeNsNameList := g.GetServiceToRoute(svcNsName)
	if !utils.HasElem(routeTypeNsNameList, routeTypeNsName) {
		routeTypeNsNameList = append(routeTypeNsNameList, routeTypeNsName)
		g.serviceToRoute.AddOrUpdate(svcNsName, routeTypeNsNameList)
	}
}

func (g *GWLister) DeleteRouteToServiceMappings(routeTypeNsName, svcNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// delete service to route mapping
	_, routeTypeNsNameList := g.GetServiceToRoute(svcNsName)
	routeTypeNsNameList = utils.Remove(routeTypeNsNameList, routeTypeNsName)
	g.serviceToRoute.AddOrUpdate(svcNsName, routeTypeNsNameList)

	// delete route to service mapping
	_, svcNsNameList := g.GetRouteToService(routeTypeNsName)
	svcNsNameList = utils.Remove(svcNsNameList, svcNsName)
	g.routeToService.AddOrUpdate(routeTypeNsName, svcNsNameList)
}

//=====All route <-> child vs go here.

func (g *GWLister) GetRouteToChildVS(routeTypeNsName string) (bool, []string) {
	if found, obj := g.routeToChildVS.Get(routeTypeNsName); found {
		return true, obj.([]string)
	}
	return false, []string{}
}

func (g *GWLister) DeleteRouteToChildVS(routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.routeToChildVS.Delete(routeTypeNsName)
}

func (g *GWLister) UpdateRouteChildVSMappings(routeTypeNsName, childVS string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// update route to child vs mapping
	_, childVSList := g.GetRouteToChildVS(routeTypeNsName)
	if !utils.HasElem(childVSList, childVS) {
		childVSList = append(childVSList, childVS)
		g.routeToChildVS.AddOrUpdate(routeTypeNsName, childVSList)
	}
}

func (g *GWLister) DeleteRouteChildVSMappings(routeTypeNsName, childVS string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// delete route to child vs mapping
	_, childVSList := g.GetRouteToChildVS(routeTypeNsName)
	childVSList = utils.Remove(childVSList, childVS)
	g.routeToChildVS.AddOrUpdate(routeTypeNsName, childVSList)
}

//=====All route function go here.

func (g *GWLister) DeleteRouteGatewayMappings(routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// delete route to gateway mapping
	found, obj := g.routeToGateway.Get(routeTypeNsName)
	if !found {
		return
	}
	gwNsNameList := obj.([]string)
	g.routeToGateway.Delete(routeTypeNsName)

	for _, gwNsName := range gwNsNameList {
		if found, obj := g.gatewayToRoute.Get(gwNsName); found {
			routeTypeNsNameList := obj.([]string)
			routeTypeNsNameList = utils.Remove(routeTypeNsNameList, routeTypeNsName)
			g.gatewayToRoute.AddOrUpdate(gwNsName, routeTypeNsNameList)
		}
	}
}

func (g *GWLister) DeleteRouteServiceMappings(routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	found, obj := g.routeToService.Get(routeTypeNsName)
	if !found {
		return
	}
	svcNsNameList := obj.([]string)
	g.routeToService.Delete(routeTypeNsName)

	for _, svcNsName := range svcNsNameList {
		if found, obj := g.serviceToRoute.Get(svcNsName); found {
			routeTypeNsNameList := obj.([]string)
			routeTypeNsNameList = utils.Remove(routeTypeNsNameList, routeTypeNsName)
			g.gatewayToRoute.AddOrUpdate(svcNsName, routeTypeNsNameList)
		}
	}
}

//=====All service function go here.

func (g *GWLister) DeleteServiceGatewayMappings(svcNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	// delete service to gateway mapping
	found, obj := g.serviceToGateway.Get(svcNsName)
	if !found {
		return
	}
	gwNsNameList := obj.([]string)
	g.serviceToGateway.Delete(svcNsName)

	for _, gwNsName := range gwNsNameList {
		if found, obj := g.gatewayToService.Get(gwNsName); found {
			svcNsNameList := obj.([]string)
			svcNsNameList = utils.Remove(svcNsNameList, svcNsName)
			g.gatewayToService.AddOrUpdate(gwNsName, svcNsNameList)
		}
	}
}

func (g *GWLister) DeleteServiceRouteMappings(svcNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	found, obj := g.serviceToRoute.Get(svcNsName)
	if !found {
		return
	}
	routeTypeNsNameList := obj.([]string)
	g.serviceToRoute.Delete(svcNsName)

	for _, routeTypeNsName := range routeTypeNsNameList {
		if found, obj := g.routeToService.Get(routeTypeNsName); found {
			svcNsNameList := obj.([]string)
			svcNsNameList = utils.Remove(svcNsNameList, svcNsName)
			g.routeToService.AddOrUpdate(svcNsName, svcNsNameList)
		}
	}
}

func (g *GWLister) GetRouteToGatewayListener(routeType, routeNsName string) []string {
	routeKey := routeType + "/" + routeNsName
	_, obj := g.routeToGatewayListener.Get(routeKey)
	return obj.([]string)

}
func (g *GWLister) UpdateGatewayListenerRouteMappings(gwListener, routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	g.gatewayListenerToRoute.AddOrUpdate(gwListener, routeTypeNsName)

	// update route to gw mapping
	if found, obj := g.routeToGatewayListener.Get(routeTypeNsName); found {
		gwListeners := obj.([]string)
		if !utils.HasElem(gwListeners, gwListener) {
			gwListeners = append(gwListeners, gwListener)
		}
		g.routeToGatewayListener.AddOrUpdate(routeTypeNsName, gwListeners)
	} else {
		gwListeners := []string{gwListener}
		g.routeToGatewayListener.AddOrUpdate(routeTypeNsName, gwListeners)
	}
}

func (g *GWLister) DeleteGatewayListenerRouteMappings(routeTypeNsName string) {
	g.gwLock.Lock()
	defer g.gwLock.Unlock()

	if found, obj := g.routeToGatewayListener.Get(routeTypeNsName); found {
		gatewayListenerList := obj.([]string)
		for _, gatewayListener := range gatewayListenerList {
			g.gatewayListenerToRoute.Delete(gatewayListener)
		}
		g.routeToGatewayListener.Delete(routeTypeNsName)
	}
}
