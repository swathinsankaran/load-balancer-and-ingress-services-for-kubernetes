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

package nodes

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

var (
	L4Rule = GraphSchema{
		Type:              lib.L4Rule,
		GetParentServices: L4RuleToSvc,
	}
)

func L4RuleToSvc(l4RuleName string, namespace string, key string) ([]string, bool) {
	var allSvcs []string

	// Get all services that are mapped to this L4Rule.
	services, err := utils.GetInformers().ServiceInformer.Informer().GetIndexer().ByIndex(lib.L4RuleToServicesIndex, l4RuleName)
	if err != nil {
		utils.AviLog.Errorf("key: %s, msg: failed to get the services mapped to L4Rule %s", key, l4RuleName)
		return allSvcs, false
	}

	for _, svc := range services {
		svcObj, isSvc := svc.(*corev1.Service)
		if isSvc {
			allSvcs = append(allSvcs, svcObj.Namespace+"/"+svcObj.Name)
		}
	}

	utils.AviLog.Debugf("key: %s, msg: total services retrieved from L4Rule: %s", key, allSvcs)
	return allSvcs, true
}
