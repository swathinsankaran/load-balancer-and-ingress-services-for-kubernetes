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
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

// ProcessAndPublishToSyncLayer will be called from the rest layer with list of restOp objects
// It does the pre-processing of objects before pushing it to the sync layer workers.
func ProcessAndPublishToSyncLayer(restOp []*utils.RestOp) {
	// TODO
}

// PublishToSyncLayer figures out the worker and push the restOp objects to it.
func PublishToSyncLayer(key string, restOp *utils.RestOp, sharedQueue *utils.WorkerQueue) {
	// TOCHECK: number of workers? rest layer has 8 so here also 8?
	bkt := utils.Bkt(restOp.ObjName, sharedQueue.NumWorkers)
	sharedQueue.Workqueue[bkt].AddRateLimited(restOp)
	utils.AviLog.Infof("key: %s, msg: Published key to sync layer with modelName: %s", key, restOp.ObjName)

}

// Dequeues the objects coming to the sync layer and does the GET operation from the controller
// and validates the operations.
func DequeueSync(restOp *utils.RestOp) {
	utils.AviLog.Infof("key: %s, msg: start rest layer sync.", restOp.ObjName)
	//TODO
}
