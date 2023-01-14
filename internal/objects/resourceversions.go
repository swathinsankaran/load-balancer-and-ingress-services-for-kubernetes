/*
 * Copyright 2020-2021 VMware, Inc.
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

// APIs to access AVI graph from and to memory. The VS should have a uuid and the corresponding model.

package objects

import (
	"sync"
)

var resouceVerInstance *ResourceVersionLister[string]
var resourceVerOnce sync.Once

func SharedResourceVerInstanceLister() *ResourceVersionLister[string] {
	resourceVerOnce.Do(func() {
		ResourceVerStore := NewObjectMapStore[string]()
		resouceVerInstance = &ResourceVersionLister[string]{}
		resouceVerInstance.ResourceVerStore = ResourceVerStore
	})
	return resouceVerInstance
}

type ResourceVersionLister[T SupportedTypes] struct {
	ResourceVerStore *ObjectMapStore[T]
}

func (a *ResourceVersionLister[T]) Save(vsName string, resVer T) {
	a.ResourceVerStore.AddOrUpdate(vsName, resVer)
}

func (a *ResourceVersionLister[T]) Get(resName string) (bool, T) {
	ok, obj := a.ResourceVerStore.Get(resName)
	return ok, obj
}

func (a *ResourceVersionLister[T]) GetAll() map[string]T {
	obj := a.ResourceVerStore.GetAllObjectNames()
	return obj
}

func (a *ResourceVersionLister[T]) Delete(resName string) {
	a.ResourceVerStore.Delete(resName)

}
