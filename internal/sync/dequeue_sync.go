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
	"fmt"

	avicache "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/cache"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/rest"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"

	avimodels "github.com/vmware/alb-sdk/go/models"
)

type checksumObject struct {
	name                string
	cloudConfigCheckSum string
}

type restOpWrapper struct {
	RealMethod utils.RestMethod
	RestOp     *utils.RestOp
}

// ProcessAndPublishToSyncLayer will be called from the rest layer with list of restOp objects
// It does the pre-processing of objects before pushing it to the sync layer workers.
func ProcessAndPublishToSyncLayer(restOps []*utils.RestOp) {
	objToRestOpMap := make(map[string]*restOpWrapper)
	for _, restOp := range restOps {
		key := fmt.Sprintf("%s:%s", restOp.Model, restOp.ObjName)
		objToRestOpMap[key] = &restOpWrapper{
			RealMethod: restOp.Method,
			RestOp:     restOp,
		}
	}
	mergedRestOps := make([]*utils.RestOp, 0)
	for _, restOpWrapper := range objToRestOpMap {
		restOpWrapper.RestOp.Method = utils.RestGet
		mergedRestOps = append(mergedRestOps, restOpWrapper.RestOp)
	}
	client := avicache.SharedAVIClients().AviClient[0]
	utils.AviLog.Debugf("Merged Rest Ops: %+v", mergedRestOps)

	// Do a GET of controller objects
	rest.AviRestOperateWrapper(client, mergedRestOps)
	restlayer := rest.NewRestOperations(avicache.SharedAviObjCache(), avicache.SharedAVIClients())

	for _, restOp := range mergedRestOps {
		key := fmt.Sprintf("%s:%s", restOp.Model, restOp.ObjName)
		restOpWrapper, _ := objToRestOpMap[key]
		restOpWrapper.RestOp.Method = restOpWrapper.RealMethod

		// code to validate the checksum.
		cachingRequired := DequeueSync(restOpWrapper.RestOp)
		if cachingRequired {
			if restOp.Method == utils.RestDelete {
				// overwrite the err as nil, in a real call to controller returns with no error
				restOp.Err = nil
			}
			objKey := avicache.NamespaceName{Namespace: lib.GetTenant(), Name: restOpWrapper.RestOp.ObjName}
			restlayer.PopulateOneCache(restOpWrapper.RestOp, objKey, "from-sync-layer")
			continue
		}
		// append the objects for retry here
		utils.AviLog.Warn("Retry is required as the checksum is not the same")
		return
	}
}

// Dequeues the objects coming to the sync layer and does the GET operation from the controller
// and validates the operations.
func DequeueSync(restOp *utils.RestOp) bool {
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
		checksumFromRestOpObj, checksumFromController := getPoolCheckSum(restOp)
		if !isSameChecksum(checksumFromRestOpObj, checksumFromController) {
			utils.AviLog.Warnf("Checksum is not matching for pool object for restOp %v", restOp)
			return false
		}
	} else if restOp.Model == "Virtualservice" {
		checksumFromRestOpObj, checksumFromController := getVSChecksum(restOp)
		if !isSameChecksum(checksumFromRestOpObj, checksumFromController) {
			utils.AviLog.Warnf("Checksum is not matching for vs object for restOp %v", restOp)
			return false
		}
	} else {
		utils.AviLog.Warnf("model not implemented")
		return false
	}
	return true
}

func getPoolCheckSum(restOp *utils.RestOp) (checksumFromRestOpObj *checksumObject, checksumFromController *checksumObject) {
	key := fmt.Sprintf("%s/%s", lib.GetTenant(), restOp.ObjName)
	resp_elems := rest.RestRespArrToObjByType(restOp, "pool", key)
	utils.AviLog.Debugf("key: %s, msg: the pool object response %v", key, restOp.Response)
	if resp_elems == nil {
		utils.AviLog.Warnf("key: %s, msg: unable to find pool obj in resp %v", key, restOp.Response)
		return nil, nil
	}

	for _, resp := range resp_elems {
		name, ok := resp["name"].(string)
		if !ok {
			utils.AviLog.Warnf("key: %s, msg: Name not present in response %v", key, resp)
			return nil, nil
		}

		cksum := resp["cloud_config_cksum"].(string)
		checksumFromController = &checksumObject{
			name:                name,
			cloudConfigCheckSum: cksum,
		}
	}

	pool, ok := restOp.Obj.(avimodels.Pool)
	if !ok {
		utils.AviLog.Warnf("pool is not proper in the restOp %v", restOp)
		return nil, nil
	}

	checksumFromRestOpObj = &checksumObject{
		name:                *pool.Name,
		cloudConfigCheckSum: *pool.CloudConfigCksum,
	}
	return checksumFromRestOpObj, checksumFromController
}

func getVSChecksum(restOp *utils.RestOp) (checksumFromRestOpObj *checksumObject, checksumFromController *checksumObject) {
	key := fmt.Sprintf("%s/%s", lib.GetTenant(), restOp.ObjName)
	resp_elems := rest.RestRespArrToObjByType(restOp, "virtualservice", key)
	if resp_elems == nil {
		utils.AviLog.Warnf("key: %s, msg: unable to find vs obj in resp %v", key, restOp.Response)
		return nil, nil
	}

	for _, resp := range resp_elems {
		name, ok := resp["name"].(string)
		if !ok {
			utils.AviLog.Warnf("key: %s, msg: name not present in response %v", key, resp)
			return nil, nil
		}

		cksum := resp["cloud_config_cksum"].(string)
		checksumFromController = &checksumObject{
			name:                name,
			cloudConfigCheckSum: cksum,
		}
	}

	vs, ok := restOp.Obj.(avimodels.VirtualService)
	if !ok {
		utils.AviLog.Warnf("Vs is not proper in the restOp %v", restOp)
		return nil, nil
	}

	checksumFromRestOpObj = &checksumObject{
		name:                *vs.Name,
		cloudConfigCheckSum: *vs.CloudConfigCksum,
	}
	return checksumFromRestOpObj, checksumFromController
}
