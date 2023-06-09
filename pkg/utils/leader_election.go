/*
 * Copyright 2022-2023 VMware, Inc.
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

package utils

import (
	"context"
	"os"
	"sync"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

type callBackFunc func()
type callBackFuncWithParams func(le LeaderElector)

type leaderElector struct {
	ctx           context.Context
	wg            *sync.WaitGroup
	identity      string
	leaderElector *leaderelection.LeaderElector
	readyCh       chan struct{}
}

type LeaderElector interface {
	Run() chan struct{}
}

func NewLeaderElector(ctx context.Context, clientset kubernetes.Interface,
	OnStartedLeadingCallback, OnNewLeaderCallback callBackFunc, OnStoppedLeadingCallback callBackFuncWithParams, wg *sync.WaitGroup) (LeaderElector, error) {

	leaderElector := &leaderElector{}
	leaderElector.identity = os.Getenv("POD_NAME")
	leaderElector.readyCh = make(chan struct{})
	leaderElector.ctx = ctx
	leaderElector.wg = wg

	leaderConfig := leaderElector.GetLeaderConfig(
		clientset,
		OnStartedLeadingCallback,
		OnNewLeaderCallback,
		OnStoppedLeadingCallback,
	)
	var err error
	leaderElector.leaderElector, err = leaderelection.NewLeaderElector(leaderConfig)
	if err != nil {
		return nil, err
	}
	return leaderElector, nil
}

func (le *leaderElector) GetLeaderConfig(clientset kubernetes.Interface, OnStartedLeadingCallback,
	OnNewLeaderCallback callBackFunc, OnStoppedLeadingCallback callBackFuncWithParams) leaderelection.LeaderElectionConfig {
	return leaderelection.LeaderElectionConfig{
		Name:            le.identity,
		Lock:            le.GetLeaseLock(clientset),
		ReleaseOnCancel: true,
		LeaseDuration:   leaseDuration,
		RenewDeadline:   renewDeadline,
		RetryPeriod:     retryPeriod,
		Callbacks:       le.GetLeaderCallbacks(OnStartedLeadingCallback, OnNewLeaderCallback, OnStoppedLeadingCallback),
	}
}

func (le *leaderElector) GetLeaseLock(clientset kubernetes.Interface) *resourcelock.LeaseLock {
	return &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      leaseLockName,
			Namespace: os.Getenv("POD_NAMESPACE"),
		},
		Client: clientset.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: le.identity,
		},
	}
}

func (le *leaderElector) GetLeaderCallbacks(OnStartedLeadingCallback,
	OnNewLeaderCallback callBackFunc, OnStoppedLeadingCallback callBackFuncWithParams) leaderelection.LeaderCallbacks {
	return leaderelection.LeaderCallbacks{
		OnStartedLeading: func(ctx context.Context) {
			OnStartedLeadingCallback()
			le.OnNewLeaderElected()
		},
		OnStoppedLeading: func() {
			OnStoppedLeadingCallback(le)
		},
		OnNewLeader: func(identity string) {
			if identity == le.identity {
				return
			}
			OnNewLeaderCallback()
			le.OnNewLeaderElected()
		},
	}
}

func (le *leaderElector) OnNewLeaderElected() {
	le.readyCh <- struct{}{}
}

func (le *leaderElector) Run() chan struct{} {
	AviLog.Debug("Leader election is in-progress")
	le.wg.Add(1)
	go func() {
		defer le.wg.Done()
		le.leaderElector.Run(le.ctx)
		AviLog.Debug("leader election goroutine exited")
	}()
	return le.readyCh
}
