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

package models

import (
	"net/http"
	"sync"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

var Cache *CacheModel
var once sync.Once

// CacheModel implements ApiModel
type CacheModel struct {
	L2Cache         interface{}
	L3VsCache       interface{}
	L3PoolCache     interface{}
	L3PgCache       interface{}
	L3DSCache       interface{}
	L3L4PolicyCache interface{}
	Lock            sync.RWMutex
}

func (a *CacheModel) InitModel() {
	once.Do(func() {
		Cache = &CacheModel{}
	})
}

func (a *CacheModel) ApiOperationMap() []OperationMap {
	var operationMapList []OperationMap

	getL2Models := OperationMap{
		Route:  "/api/cache/l2",
		Method: "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			response := &Cache.L2Cache
			utils.Respond(w, response)
		},
	}

	getVsCache := OperationMap{
		Route:  "/api/cache/l3/vs",
		Method: "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			response := &Cache.L3VsCache
			utils.Respond(w, response)
		},
	}

	getPoolCache := OperationMap{
		Route:  "/api/cache/l3/pool",
		Method: "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			response := &Cache.L3PoolCache
			utils.Respond(w, response)
		},
	}

	getPgCache := OperationMap{
		Route:  "/api/cache/l3/pg",
		Method: "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			response := &Cache.L3PgCache
			utils.Respond(w, response)
		},
	}

	getDSCache := OperationMap{
		Route:  "/api/cache/l3/ds",
		Method: "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			response := &Cache.L3DSCache
			utils.Respond(w, response)
		},
	}

	getL4PolicyCache := OperationMap{
		Route:  "/api/cache/l3/l4p",
		Method: "GET",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			response := &Cache.L3L4PolicyCache
			utils.Respond(w, response)
		},
	}

	operationMapList = append(operationMapList, getL2Models, getVsCache, getPoolCache, getPgCache, getDSCache, getL4PolicyCache)
	return operationMapList
}
