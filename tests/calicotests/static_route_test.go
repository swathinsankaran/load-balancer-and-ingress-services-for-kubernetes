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

package calicotests

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/k8s"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	avinodes "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/nodes"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/objects"
	crdfake "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/client/v1alpha1/clientset/versioned/fake"
	v1alpha2crdfake "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/client/v1alpha2/clientset/versioned/fake"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/tests/integrationtest"

	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

var KubeClient *k8sfake.Clientset
var CRDClient *crdfake.Clientset
var v1alpha2CRDClient *v1alpha2crdfake.Clientset
var ctrl *k8s.AviController
var DynamicClient *dynamicfake.FakeDynamicClient

func TestMain(m *testing.M) {
	os.Setenv("VIP_NETWORK_LIST", `[{"networkName":"net123"}]`)
	os.Setenv("CLUSTER_NAME", "cluster")
	os.Setenv("CLOUD_NAME", "CLOUD_VCENTER")
	os.Setenv("SEG_NAME", "Default-Group")
	os.Setenv("NODE_NETWORK_LIST", `[{"networkName":"net123","cidrs":["10.79.168.0/22"]}]`)
	os.Setenv("SERVICE_TYPE", "ClusterIP")
	os.Setenv("AUTO_L4_FQDN", "disable")
	os.Setenv("POD_NAMESPACE", utils.AKO_DEFAULT_NS)
	os.Setenv("SHARD_VS_SIZE", "LARGE")
	os.Setenv("CNI_PLUGIN", "calico")

	akoControlConfig := lib.AKOControlConfig()
	KubeClient = k8sfake.NewSimpleClientset()
	CRDClient = crdfake.NewSimpleClientset()
	v1alpha2CRDClient = v1alpha2crdfake.NewSimpleClientset()

	gvrToKind := make(map[schema.GroupVersionResource]string)
	var testData unstructured.Unstructured
	testData.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "blockaffinities",
		"metadata": map[string]interface{}{
			"name":      "testblockaffinity",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"cidr":    "10.133.0.0/26",
			"deleted": "false",
			"node":    "testNodeCalico",
			"state":   "confirmed",
		},
	})
	gvrToKind[lib.CalicoBlockaffinityGVR] = "blockaffinitiesList"

	DynamicClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(), gvrToKind, &testData)
	//DynamicClient = dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	lib.SetDynamicClientSet(DynamicClient)
	akoControlConfig.SetCRDClientset(CRDClient)
	akoControlConfig.Setv1alpha2CRDClientset(v1alpha2CRDClient)
	akoControlConfig.SetAKOInstanceFlag(true)
	akoControlConfig.SetEventRecorder(lib.AKOEventComponent, KubeClient, true)
	data := map[string][]byte{
		"username": []byte("admin"),
		"password": []byte("admin"),
	}
	object := metav1.ObjectMeta{Name: "avi-secret", Namespace: utils.GetAKONamespace()}
	secret := &corev1.Secret{Data: data, ObjectMeta: object}
	KubeClient.CoreV1().Secrets(utils.GetAKONamespace()).Create(context.TODO(), secret, metav1.CreateOptions{})

	registeredInformers := []string{
		utils.ServiceInformer,
		utils.EndpointInformer,
		utils.IngressInformer,
		utils.IngressClassInformer,
		utils.SecretInformer,
		utils.NSInformer,
		utils.NodeInformer,
		utils.ConfigMapInformer,
	}
	utils.NewInformers(utils.KubeClientIntf{ClientSet: KubeClient}, registeredInformers)
	informers := k8s.K8sinformers{Cs: KubeClient, DynamicClient: DynamicClient}
	k8s.NewCRDInformers(CRDClient)

	integrationtest.InitializeFakeAKOAPIServer()

	integrationtest.NewAviFakeClientInstance(KubeClient)
	defer integrationtest.AviFakeClientInstance.Close()

	ctrl = k8s.SharedAviController()
	stopCh := utils.SetupSignalHandler()
	ctrlCh := make(chan struct{})
	quickSyncCh := make(chan struct{})
	waitGroupMap := make(map[string]*sync.WaitGroup)
	wgIngestion := &sync.WaitGroup{}
	waitGroupMap["ingestion"] = wgIngestion
	wgFastRetry := &sync.WaitGroup{}
	waitGroupMap["fastretry"] = wgFastRetry
	wgSlowRetry := &sync.WaitGroup{}
	waitGroupMap["slowretry"] = wgSlowRetry
	wgGraph := &sync.WaitGroup{}
	waitGroupMap["graph"] = wgGraph
	wgStatus := &sync.WaitGroup{}
	waitGroupMap["status"] = wgStatus
	wgLeaderElection := &sync.WaitGroup{}
	waitGroupMap["leaderElection"] = wgLeaderElection

	integrationtest.AddConfigMap(KubeClient)
	ctrl.SetSEGroupCloudNameFromNSAnnotations()
	integrationtest.PollForSyncStart(ctrl, 10)

	ctrl.HandleConfigMap(informers, ctrlCh, stopCh, quickSyncCh)
	integrationtest.KubeClient = KubeClient
	integrationtest.AddDefaultIngressClass()

	go ctrl.InitController(informers, registeredInformers, ctrlCh, stopCh, quickSyncCh, waitGroupMap)
	os.Exit(m.Run())
}

func TestBlockAffinity(t *testing.T) {
	g := gomega.NewGomegaWithT(t)
	modelName := "admin/global"
	nodeip := "10.1.1.2"
	nodeName := "testNodeCalico"
	objects.SharedAviGraphLister().Delete(modelName)

	// mimicking actual scenario where the node will have atleast one BlockAffinity object created from start
	var testData unstructured.Unstructured
	testData.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "blockaffinities",
		"metadata": map[string]interface{}{
			"name":      "testblockaffinity",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"cidr":    "10.133.0.0/26",
			"deleted": "false",
			"node":    nodeName,
			"state":   "confirmed",
		},
	})

	DynamicClient.Resource(lib.CalicoBlockaffinityGVR).Namespace("default").Create(context.TODO(), &testData, v1.CreateOptions{})

	nodeExample := (integrationtest.FakeNode{
		Name:    nodeName,
		Version: "1",
		NodeIP:  nodeip,
	}).NodeCalico()

	_, err := KubeClient.CoreV1().Nodes().Create(context.TODO(), nodeExample, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("error in adding Node: %v", err)
	}

	integrationtest.PollForCompletion(t, modelName, 5)

	g.Eventually(func() bool {
		found, _ := objects.SharedAviGraphLister().Get(modelName)
		return found
	}, 10*time.Second).Should(gomega.Equal(true))
	_, aviModel := objects.SharedAviGraphLister().Get(modelName)
	g.Expect(aviModel.(*avinodes.AviObjectGraph).IsVrf).To(gomega.Equal(true))
	nodes := aviModel.(*avinodes.AviObjectGraph).GetAviVRF()
	g.Expect(len(nodes)).To(gomega.Equal(1))

	g.Expect(len(nodes[0].StaticRoutes)).To(gomega.Equal(1))
	g.Expect(*(nodes[0].StaticRoutes[0].NextHop.Addr)).To(gomega.Equal(nodeip))
	g.Expect(*(nodes[0].StaticRoutes[0].Prefix.IPAddr.Addr)).To(gomega.Equal("10.133.0.0"))
	g.Expect(*(nodes[0].StaticRoutes[0].Prefix.Mask)).To(gomega.Equal(int32(26)))

	// creating a new BlockAffinity object for the node
	var testData2 unstructured.Unstructured
	testData2.SetUnstructuredContent(map[string]interface{}{
		"apiVersion": "crd.projectcalico.org/v1",
		"kind":       "blockaffinities",
		"metadata": map[string]interface{}{
			"name":      "testblockaffinity2",
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"cidr":    "10.134.0.0/26",
			"deleted": "false",
			"node":    nodeName,
			"state":   "confirmed",
		},
	})

	DynamicClient.Resource(lib.CalicoBlockaffinityGVR).Namespace("default").Create(context.TODO(), &testData2, v1.CreateOptions{})
	// waiting for routes to get populated after BlockAffinity object creation
	time.Sleep(5 * time.Second)

	_, aviModel = objects.SharedAviGraphLister().Get(modelName)
	g.Expect(aviModel.(*avinodes.AviObjectGraph).IsVrf).To(gomega.Equal(true))
	nodes = aviModel.(*avinodes.AviObjectGraph).GetAviVRF()
	g.Expect(len(nodes)).To(gomega.Equal(1))

	g.Expect(len(nodes[0].StaticRoutes)).To(gomega.Equal(2))
	g.Expect(*(nodes[0].StaticRoutes[1].NextHop.Addr)).To(gomega.Equal(nodeip))
	g.Expect(*(nodes[0].StaticRoutes[1].Prefix.IPAddr.Addr)).To(gomega.Equal("10.134.0.0"))
	g.Expect(*(nodes[0].StaticRoutes[1].Prefix.Mask)).To(gomega.Equal(int32(26)))

	time.Sleep(5 * time.Second)
	// deleting the second BlockAffinity object for the node
	DynamicClient.Resource(lib.CalicoBlockaffinityGVR).Namespace("default").Delete(context.TODO(), "testblockaffinity2", v1.DeleteOptions{})
	g.Eventually(func() int {
		num_routes := len(nodes[0].StaticRoutes)
		return num_routes
	}, 10*time.Second).Should(gomega.Equal(1))
}