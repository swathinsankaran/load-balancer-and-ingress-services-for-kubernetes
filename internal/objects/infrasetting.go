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

package objects

import (
	"sync"
)

var infral7lister *AviInfraSettingL7Lister
var infraonce sync.Once

func InfraSettingL7Lister() *AviInfraSettingL7Lister {
	infraonce.Do(func() {
		infral7lister = &AviInfraSettingL7Lister{
			IngRouteInfraSettingStore:  NewObjectMapStore[string](),
			InfraSettingShardSizeStore: NewObjectMapStore[string](),
		}
	})
	return infral7lister
}

type AviInfraSettingL7Lister struct {
	InfraSettingIngRouteLock sync.RWMutex

	// namespaced ingress/route -> infrasetting
	IngRouteInfraSettingStore *ObjectMapStore[string]

	// infrasetting -> shardSize
	InfraSettingShardSizeStore *ObjectMapStore[string]
}

func (v *AviInfraSettingL7Lister) GetIngRouteToInfraSetting(ingrouteNsName string) (bool, string) {
	found, infraSettingName := v.IngRouteInfraSettingStore.Get(ingrouteNsName)
	if !found {
		return false, ""
	}
	return true, infraSettingName
}

func (v *AviInfraSettingL7Lister) UpdateIngRouteInfraSettingMappings(ingrouteNsName, infraSettingName, shardSize string) {
	v.InfraSettingIngRouteLock.Lock()
	defer v.InfraSettingIngRouteLock.Unlock()
	v.IngRouteInfraSettingStore.AddOrUpdate(ingrouteNsName, infraSettingName)
	v.InfraSettingShardSizeStore.AddOrUpdate(infraSettingName, shardSize)
}

func (v *AviInfraSettingL7Lister) RemoveIngRouteInfraSettingMappings(ingrouteNsName string) bool {
	v.InfraSettingIngRouteLock.Lock()
	defer v.InfraSettingIngRouteLock.Unlock()
	if found, infraSettingName := v.GetIngRouteToInfraSetting(ingrouteNsName); found {
		v.InfraSettingShardSizeStore.Delete(infraSettingName)
	}
	return v.IngRouteInfraSettingStore.Delete(ingrouteNsName)
}

func (v *AviInfraSettingL7Lister) GetInfraSettingToShardSize(infraSettingName string) (bool, string) {
	found, shardSize := v.InfraSettingShardSizeStore.Get(infraSettingName)
	if !found {
		return false, ""
	}
	return true, shardSize
}
