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

package lib

import (
	"context"
	"os"
	"time"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

const (
	leaseDuration = 15 * time.Second
	renewDeadline = 10 * time.Second
	retryPeriod   = 2 * time.Second
	leaseLockName = "ako-lease-lock"
)

type leaderElector struct {
	identity      string
	leaderElector *leaderelection.LeaderElector
	readyCh       chan struct{}
}

type LeaderElector interface {
	Run(ctx context.Context) chan struct{}
}

func NewLeaderElector(cs *kubernetes.Clientset, populateCache func() error,
	syncStatus func()) (LeaderElector, error) {

	leaderElector := &leaderElector{
		identity: os.Getenv("POD_NAME"),
		readyCh:  make(chan struct{}),
	}
	leaseLock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseLockName,
			Namespace: os.Getenv("POD_NAMESPACE"),
		},
		Client: cs.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: leaderElector.identity,
		},
	}
	leaderCallBacks := leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			utils.AviLog.Debug("Current AKO is the leader")
			AKOControlConfig().SetIsLeaderFlag(true)
			AKOControlConfig().PodEventf(v1.EventTypeNormal, "LeaderElection", "AKO became the leader")
			if err := populateCache(); err != nil {
				ShutdownApi()
				utils.AviLog.Errorf("populate cache failed with error %v", err)
				return
			}
			AKOControlConfig().PodEventf(v1.EventTypeNormal, "ControllerCacheSync", "Controller cache population completed")

			// once the l3 cache is populated, we can call the updatestatus functions from here
			go syncStatus()
			leaderElector.readyCh <- struct{}{}
		},
		OnStoppedLeading: func() {
			AKOControlConfig().SetIsLeaderFlag(false)
			utils.AviLog.Info("Current AKO lost the leadership")
		},
		OnNewLeader: func(identity string) {
			if identity == leaderElector.identity {
				utils.AviLog.Info("Current AKO is still leading")
				return
			}
			utils.AviLog.Infof("AKO leader is now %s", identity)
			AKOControlConfig().SetIsLeaderFlag(false)
			AKOControlConfig().PodEventf(v1.EventTypeNormal, "LeaderElection", "AKO became a follower")
			leaderElector.readyCh <- struct{}{}
		},
	}
	leaderConfig := leaderelection.LeaderElectionConfig{
		Name:            leaderElector.identity,
		Lock:            leaseLock,
		ReleaseOnCancel: true,
		LeaseDuration:   leaseDuration,
		RenewDeadline:   renewDeadline,
		RetryPeriod:     retryPeriod,
		Callbacks:       leaderCallBacks,
	}
	var err error
	leaderElector.leaderElector, err = leaderelection.NewLeaderElector(leaderConfig)
	if err != nil {
		return nil, err
	}
	return leaderElector, nil
}

func (le *leaderElector) Run(ctx context.Context) chan struct{} {
	utils.AviLog.Debug("Leader election is in-progress")
	go le.leaderElector.Run(ctx)
	return le.readyCh
}
