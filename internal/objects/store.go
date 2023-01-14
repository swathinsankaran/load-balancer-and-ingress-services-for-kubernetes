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

// Construct in memory database that populates updates from both kubernetes and MCP
// The format is: namespace:[object_name: obj]

package objects

import (
	"sync"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
	v1 "k8s.io/api/core/v1"
)

type ObjectStore[T SupportedTypes] struct {
	NSObjectMap map[string]*ObjectMapStore[T]
	NSLock      sync.RWMutex
}

func NewObjectStore[T SupportedTypes]() *ObjectStore[T] {
	objectStore := &ObjectStore[T]{}
	objectStore.NSObjectMap = make(map[string]*ObjectMapStore[T])
	return objectStore
}

func (store *ObjectStore[T]) GetNSStore(nsName string) *ObjectMapStore[T] {
	store.NSLock.Lock()
	defer store.NSLock.Unlock()
	val, ok := store.NSObjectMap[nsName]
	if ok {
		return val
	} else {
		// This namespace is not initialized, let's initialze it
		nsObjStore := NewObjectMapStore[T]()
		// Update the store.
		store.NSObjectMap[nsName] = nsObjStore
		return nsObjStore
	}
}

func (store *ObjectStore[T]) DeleteNSStore(nsName string) bool {
	// Deletes the key for a namespace. Wipes off the entire NS. So use with care.
	store.NSLock.Lock()
	defer store.NSLock.Unlock()
	_, ok := store.NSObjectMap[nsName]
	if ok {
		delete(store.NSObjectMap, nsName)
		return true
	}
	utils.AviLog.Warnf("Namespace: %s not found, nothing to delete returning false", nsName)
	return false

}

func (store *ObjectStore[T]) GetAllNamespaces() []string {
	// Take a read lock on the store and write lock on NS object
	store.NSLock.RLock()
	defer store.NSLock.RUnlock()
	var allNamespaces []string
	for ns := range store.NSObjectMap {
		allNamespaces = append(allNamespaces, ns)
	}
	return allNamespaces

}

type SupportedTypes interface {
	[]string | map[string][]string | string | *v1.Node | []utils.NPLAnnotation | interface{} | utils.PodsWithTargetPort
}

type ObjectMapStore[T SupportedTypes] struct {
	ObjectMap map[string]T
	ObjLock   sync.RWMutex
}

func NewObjectMapStore[T SupportedTypes]() *ObjectMapStore[T] {
	nsObjStore := &ObjectMapStore[T]{}
	nsObjStore.ObjectMap = make(map[string]T)
	return nsObjStore
}

func (o *ObjectMapStore[T]) AddOrUpdate(objName string, obj T) {
	o.ObjLock.Lock()
	defer o.ObjLock.Unlock()
	o.ObjectMap[objName] = obj
}

func (o *ObjectMapStore[T]) Delete(objName string) bool {
	o.ObjLock.Lock()
	defer o.ObjLock.Unlock()
	_, ok := o.ObjectMap[objName]
	if ok {
		delete(o.ObjectMap, objName)
		return true
	}
	return false

}

func (o *ObjectMapStore[T]) Get(objName string) (bool, T) {
	o.ObjLock.RLock()
	defer o.ObjLock.RUnlock()
	val, ok := o.ObjectMap[objName]
	if ok {
		return true, val
	}
	return false, val

}

func (o *ObjectMapStore[T]) GetAllObjectNames() map[string]T {
	o.ObjLock.RLock()
	defer o.ObjLock.RUnlock()
	// TODO (sudswas): Pass a copy instead of the reference
	return o.ObjectMap

}

func (o *ObjectMapStore[T]) GetAllKeys() []string {
	o.ObjLock.RLock()
	defer o.ObjLock.RUnlock()
	allKeys := []string{}
	for k := range o.ObjectMap {
		allKeys = append(allKeys, k)
	}
	return allKeys
}

func (o *ObjectMapStore[T]) CopyAllObjects() map[string]T {
	o.ObjLock.RLock()
	defer o.ObjLock.RUnlock()
	CopiedObjMap := make(map[string]T)
	for k, v := range o.ObjectMap {
		CopiedObjMap[k] = v
	}
	return CopiedObjMap
}
