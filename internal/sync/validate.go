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

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/rest"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"

	avimodels "github.com/vmware/alb-sdk/go/models"
)

func isSameChecksum(checksumFromRestOpObj, checksumFromController *checksumObject) bool {
	h1 := utils.Hash(utils.Stringify(checksumFromRestOpObj))
	h2 := utils.Hash(utils.Stringify(checksumFromController))
	utils.AviLog.Infof("Got checksum from controller and rest layer as %d and %d respectively", h2, h1)
	return h1 == h2
}

// validateChecksum calculates the checksum of the objects coming from controller with the object from
// the rest layer.
func validateChecksum(restOp *utils.RestOp) bool {

	if restOp.Model == "Pool" {
		checksumFromRestOpObj, checksumFromController := getPoolCheckSum(restOp)
		if !isSameChecksum(checksumFromRestOpObj, checksumFromController) {
			utils.AviLog.Warnf("Checksum is not matching for pool object for restOp %v", restOp)
			return false
		}
		utils.AviLog.Warnf("Checksum is the same for pool object with name %s", restOp.ObjName)
	} else if restOp.Model == "VirtualService" {
		checksumFromRestOpObj, checksumFromController := getVSChecksum(restOp)
		if !isSameChecksum(checksumFromRestOpObj, checksumFromController) {
			utils.AviLog.Warnf("Checksum is not matching for vs object for restOp %v", restOp)
			return false
		}
		utils.AviLog.Warnf("Checksum is the same for vs object with name %s", restOp.ObjName)
	} else {
		utils.AviLog.Warnf("model not implemented")
	}
	return true
}

func getPoolCheckSum(restOp *utils.RestOp) (checksumFromRestOpObj *checksumObject, checksumFromController *checksumObject) {
	key := fmt.Sprintf("%s/%s", lib.GetTenant(), restOp.ObjName)
	resp_elems := rest.RestRespArrToObjByType(restOp, "pool", key)
	utils.AviLog.Info("key: %s, msg: the pool object response %v", key, restOp.Response)
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
