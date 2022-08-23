/*
 * Copyright 2021-2022 VMware, Inc.
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
package sync

import (
	avicache "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/cache"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"

	avimodels "github.com/vmware/alb-sdk/go/models"
)

type checksumObject struct {
	name                string
	cloudConfigCheckSum string
}

// ProcessAndPublishToSyncLayer will be called from the rest layer with list of restOp objects
// It does the pre-processing of objects before pushing it to the sync layer workers.
func ProcessAndPublishToSyncLayer(restOps *utils.RestOp) {
	// TODO
}

// PublishToSyncLayer figures out the worker and push the restOp objects to it.
func PublishToSyncLayer(key string, restOps []*utils.RestOp) {
	sharedQueue := utils.SharedWorkQueue().GetQueueByName(utils.SyncLayer)
	for _, restOp := range restOps {
		bkt := utils.Bkt(restOp.ObjName, sharedQueue.NumWorkers)
		sharedQueue.Workqueue[bkt].AddRateLimited(restOp)
		utils.AviLog.Infof("key: %s, msg: Published key to sync layer with modelName: %s", key, restOp.ObjName)
	}
}

// Dequeues the objects coming to the sync layer and does the GET operation from the controller
// and validates the operations.
func DequeueSync(restOp *utils.RestOp) {
	utils.AviLog.Infof("key: %s, msg: start sync layer sync.", restOp.ObjName)
	//TODO
	/*
		1. get checksum of restop (all objects)
		2. get data for obj type from controller using "path" variable from restop
		3. Extract out fields from the above controller data which are there in restop
		4. get checksum of the extracted data from controller.
		5. validate checksum from 1 and 4th setup
		6. if validation passed, update CONTROLLER CACHE on AKO.
		7. If validation failed, retry 3 times. After 3rd time, add it to fAILED QUEUE.
		8. After dequeuing is done, execute fAILED QUEUE.
	*/
	if restOp.Model == "Pool" {
		pool, ok := restOp.Obj.(avimodels.Pool)
		if !ok {
			utils.AviLog.Warnf("pool is not proper in the restOp %v", restOp)
			return
		}
		if restOp.Method == "Delete" {
			k := avicache.NamespaceName{Namespace: lib.GetTenant(), Name: *pool.Name}
			avicache.SharedAviObjCache().PoolCache.AviCacheDelete(k)
			return
		}
		checksumFromRestOpObj, checksumFromController := processPoolObj(restOp.Path, &pool)
		if !validateChecksums(checksumFromRestOpObj, checksumFromController) {
			utils.AviLog.Warnf("Checksum is not matching for pool object with name %v", pool.Name)
			return
		}
		avicache.SharedAviObjCache().AviPopulateOnePoolCache(avicache.SharedAVIClients().AviClient[0], utils.CloudName, *pool.Name)
	} else if restOp.Model == "Virtualservice" {
		vs, ok := restOp.Obj.(avimodels.VirtualService)
		if !ok {
			utils.AviLog.Warnf("Vs is not proper in the restOp %v", restOp)
			return
		}
		if restOp.Method == "Delete" {
			k := avicache.NamespaceName{Namespace: lib.GetTenant(), Name: *vs.Name}
			avicache.SharedAviObjCache().VsCacheMeta.AviCacheDelete(k)
			return
		}
		checksumFromRestOpObj, checksumFromController := processVsObj(restOp.Path, &vs)
		if !validateChecksums(checksumFromRestOpObj, checksumFromController) {
			utils.AviLog.Warnf("Checksum is not matching for vs object with name %v", vs.Name)
			return
		}
		avicache.SharedAviObjCache().AviObjOneVSCachePopulate(avicache.SharedAVIClients().AviClient[0], utils.CloudName, *vs.Name)
	} else {
		utils.AviLog.Warnf("model not implemented")
		return
	}
}

func validateChecksums(checksumFromRestOpObj, checksumFromController *checksumObject) bool {
	h1 := utils.Hash(utils.Stringify(checksumFromRestOpObj))
	h2 := utils.Hash(utils.Stringify(checksumFromController))
	return h1 == h2
}

func processPoolObj(uri string, pool *avimodels.Pool) (checksumFromRestOpObj *checksumObject, checksumFromController *checksumObject) {
	checksumFromRestOpObj = &checksumObject{
		name:                *pool.Name,
		cloudConfigCheckSum: *pool.CloudConfigCksum,
	}
	// TODO
	return
}

func processVsObj(uri string, vs *avimodels.VirtualService) (checksumFromRestOpObj *checksumObject, checksumFromController *checksumObject) {
	checksumFromRestOpObj = &checksumObject{
		name:                *vs.Name,
		cloudConfigCheckSum: *vs.CloudConfigCksum,
	}

	client := avicache.SharedAVIClients().AviClient[0]
	var restResponse interface{}
	err := lib.AviGet(client, uri, &restResponse)
	if err != nil {
		utils.AviLog.Warnf("Vs Get uri %v returned err %v", uri, err)
		return nil, nil
	} else {
		resp, ok := restResponse.(map[string]interface{})
		if !ok {
			utils.AviLog.Warnf("Vs Get uri %v returned %v type %T not as map[string]interface{}", uri, restResponse, restResponse)
			return nil, nil
		}
		utils.AviLog.Debugf("Vs Get uri %v returned %v vses", uri, resp["count"])
		results, ok := resp["results"].([]interface{})
		if !ok {
			utils.AviLog.Warnf("results not of type []interface{} Instead of type %T", resp["results"])
			return nil, nil
		}
		for _, vs_intf := range results {
			vsObj, ok := vs_intf.(map[string]interface{})
			if !ok {
				utils.AviLog.Warnf("vs_intf is not of type map[string]interface{} Instead of type %T", vs_intf)
				return nil, nil
			}
			checksumFromController = &checksumObject{
				name:                vsObj["name"].(string),
				cloudConfigCheckSum: vsObj["cloud_config_cksum"].(string),
			}
			break
		}
	}
	return checksumFromRestOpObj, checksumFromController
}
