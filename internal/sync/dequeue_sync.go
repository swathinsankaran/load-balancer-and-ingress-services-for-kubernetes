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
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/rest"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

const (
	allowedNumOfRetries = 10
)

type checksumObject struct {
	name                string
	cloudConfigCheckSum string
}

type restOpWrapper struct {
	RealMethod utils.RestMethod
	RestOp     *utils.RestOp
}

// DequeueSync will be invoked by every sync layer workers and it does
// 1. GET from the controller with the object's uri.
// 2. Validates the object by comparing the checksum of the object with the GET response.
func DequeueSync(restOp *utils.RestOp) {
	utils.AviLog.Infof("key: %s, msg: start sync layer sync", restOp.ObjName)
	newRestOps := &restOpWrapper{
		RealMethod: restOp.Method,
		RestOp:     restOp,
	}
	newRestOps.RestOp.Method = utils.RestGet
	client := avicache.SharedAVIClients().AviClient[0]
	utils.AviLog.Debugf("Merged Rest Ops: %+v", newRestOps)

	// Do a GET of controller objects
	rest.AviRestOperateWrapper(client, []*utils.RestOp{newRestOps.RestOp})
	restlayer := rest.NewRestOperations(avicache.SharedAviObjCache(), avicache.SharedAVIClients())

	newRestOps.RestOp.Method = newRestOps.RealMethod

	// validate the checksum.
	isValid := validateChecksum(newRestOps.RestOp)
	if isValid {
		if restOp.Method == utils.RestDelete {
			// overwrite the err as nil, a real call to controller returns no error
			restOp.Err = nil
		}
		objKey := avicache.NamespaceName{Namespace: lib.GetTenant(), Name: newRestOps.RestOp.ObjName}
		restlayer.PopulateOneCache(newRestOps.RestOp, objKey, "from-sync-layer")
		return
	}
	// append the objects for retry here
	newRestOps.RestOp.RetryNum += 1
	if newRestOps.RestOp.RetryNum == allowedNumOfRetries {
		utils.AviLog.Warn("Retry limit reached for object %s, not pushing the object to sync", newRestOps.RestOp.ObjName)
		return
	}
	rest.PublishToSyncLayer([]*utils.RestOp{newRestOps.RestOp})
	utils.AviLog.Warn("Retry is required as the checksum is not the same")
}
