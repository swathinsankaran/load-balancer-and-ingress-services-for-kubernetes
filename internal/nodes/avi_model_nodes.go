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
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	avicache "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/cache"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/internal/lib"
	akov1alpha1 "github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1alpha1"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"

	avimodels "github.com/vmware/alb-sdk/go/models"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
)

/*
DELIM : String Delimiter for when concatenating strings before hashing

(short) distinguish ab+c from a+bc, (a,b,c are strings)

(long) Concatenating strings and calculating hash once instead of hashing individual
strings and adding the resultant hashes should reduce hash collisions
But simply concatenating brings up its own issues.
For eg. two pairs of strings ("abcde", "fgh") and ("abc", "defgh")
give the same result when concatenated ie "abcdefgh" and thus hash collision
Therefore we add a delimiter when concatenating to distinguish "abcde:fgh" from "abc:defgh"
*/
const delim = ":"

type AviModelNode interface {
	//Each AVIModelNode represents a AVI API object.
	GetCheckSum() uint32
	CalculateCheckSum()
	GetNodeType() string
	CopyNode() AviModelNode
}

type AviObjectGraphIntf interface {
	GetOrderedNodes() []AviModelNode
}

type AviObjectGraph struct {
	modelNodes    []AviModelNode
	Name          string
	GraphChecksum uint32
	IsVrf         bool
	RetryCount    int
	Validator     *Validator
	Lock          sync.RWMutex
}

// GetCopy : Create a copy the model generated by graph layer after acquiring a ReadLock.
// The copy would be used by rest layer. This would ensure that any subsequent chages
// made in the model by made graph layer would not impact the rest layer. For all such subsequent changes,
// a new key would be published, which would be processed by the graph layer later.
func (v *AviObjectGraph) GetCopy(key string) (*AviObjectGraph, bool) {
	v.Lock.RLock()
	defer v.Lock.RUnlock()
	// Decrement the counter value before copying.
	v.DecrementRetryCounter()
	newModel := AviObjectGraph{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Errorf("key: %s, Unable to marshal: %s", key, err)
		return nil, false
	}
	err = json.Unmarshal(bytes, &newModel)
	if err != nil {
		utils.AviLog.Errorf("key: %s, Unable to Unmarshal src: %s", key, err)
		return nil, false
	}
	for _, node := range v.GetOrderedNodes() {
		newModel.AddModelNode(node.CopyNode())
	}
	newModel.SetRetryCounter(v.RetryCount)
	utils.AviLog.Debugf("key: %s, nodes copied from model: %d", key, len(newModel.modelNodes))
	return &newModel, true
}

func (v *AviObjectGraph) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.GraphChecksum
}

func (v *AviObjectGraph) SetRetryCounter(num ...int) {
	// Overwrite the retry counter value.
	v.Lock.RLock()
	defer v.Lock.RUnlock()
	if len(num) > 0 {
		v.RetryCount = num[0]
	} else {
		v.RetryCount = 10
	}
}

func (v *AviObjectGraph) GetRetryCounter() int {
	// Overwrite the retry counter value.
	v.Lock.RLock()
	defer v.Lock.RUnlock()
	return v.RetryCount
}

func (v *AviObjectGraph) DecrementRetryCounter() {
	// Overwrite the retry counter value.
	if v.RetryCount != 0 {
		v.RetryCount = v.RetryCount - 1
	}
}

func (v *AviObjectGraph) CalculateCheckSum() {
	v.Lock.Lock()
	defer v.Lock.Unlock()
	v.GraphChecksum = 0
	for _, model := range v.modelNodes {
		if lib.IsEvhEnabled() {
			if modelVsNode, ok := model.(*AviEvhVsNode); ok {
				v.GraphChecksum = v.GraphChecksum + modelVsNode.CalculateForGraphChecksum()
				continue
			}
		} else {
			if modelVsNode, ok := model.(*AviVsNode); ok {
				v.GraphChecksum = v.GraphChecksum + modelVsNode.CalculateForGraphChecksum()
				continue
			}
		}
		v.GraphChecksum = v.GraphChecksum + model.GetCheckSum()
	}
}

func (v *AviVsNode) CalculateForGraphChecksum() uint32 {
	checksumStringSlice := []string{fmt.Sprint(v.GetCheckSum())}
	for _, pool := range v.PoolRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(pool.GetCheckSum()))
	}
	for _, pg := range v.PoolGroupRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(pg.GetCheckSum()))
	}
	for _, ds := range v.HTTPDSrefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(ds.GetCheckSum()))
	}
	for _, sni := range v.SniNodes {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(sni.CalculateForGraphChecksum()))
	}
	for _, passthrough := range v.PassthroughChildNodes {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(passthrough.CalculateForGraphChecksum()))
	}
	for _, cacert := range v.CACertRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(cacert.GetCheckSum()))
	}
	for _, sslkey := range v.SSLKeyCertRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(sslkey.GetCheckSum()))
	}
	for _, httppol := range v.HttpPolicyRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(httppol.GetCheckSum()))
	}
	for _, vsvip := range v.VSVIPRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(vsvip.GetCheckSum()))
	}
	for _, l4pol := range v.L4PolicyRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(l4pol.GetCheckSum()))
	}

	return utils.Hash(strings.Join(checksumStringSlice, ":"))
}

func (v *AviEvhVsNode) CalculateForGraphChecksum() uint32 {
	checksumStringSlice := []string{fmt.Sprint(v.GetCheckSum())}
	for _, pool := range v.PoolRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(pool.GetCheckSum()))
	}
	for _, pg := range v.PoolGroupRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(pg.GetCheckSum()))
	}
	for _, ds := range v.HTTPDSrefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(ds.GetCheckSum()))
	}
	for _, evh := range v.EvhNodes {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(evh.CalculateForGraphChecksum()))
	}
	for _, cacert := range v.CACertRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(cacert.GetCheckSum()))
	}
	for _, sslkey := range v.SSLKeyCertRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(sslkey.GetCheckSum()))
	}
	for _, httppol := range v.HttpPolicyRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(httppol.GetCheckSum()))
	}
	for _, vsvip := range v.VSVIPRefs {
		checksumStringSlice = append(checksumStringSlice, fmt.Sprint(vsvip.GetCheckSum()))
	}

	return utils.Hash(strings.Join(checksumStringSlice, ":"))
}

func NewAviObjectGraph() *AviObjectGraph {
	validator := NewNodesValidator()
	return &AviObjectGraph{Validator: validator}
}

func (o *AviObjectGraph) AddModelNode(node AviModelNode) {
	o.modelNodes = append(o.modelNodes, node)
}

func (o *AviObjectGraph) RemovePoolNodeRefs(poolName string) {
	utils.AviLog.Debugf("Removing Pool: %s", poolName)
	for _, node := range o.modelNodes {
		if node.GetNodeType() == "VirtualServiceNode" {
			for i, pool := range node.(*AviVsNode).PoolRefs {
				if pool.Name == poolName {
					utils.AviLog.Debugf("Removing poolref: %s", poolName)
					utils.AviLog.Debugf("Before removing the pool nodes are: %s", utils.Stringify(node.(*AviVsNode).PoolRefs))
					node.(*AviVsNode).PoolRefs = append(node.(*AviVsNode).PoolRefs[:i], node.(*AviVsNode).PoolRefs[i+1:]...)
					break
				}
			}
			utils.AviLog.Debugf("After removing the pool nodes are: %s", utils.Stringify(node.(*AviVsNode).PoolRefs))
		}
	}
}

func (o *AviObjectGraph) RemovePGNodeRefs(pgName string, vsNode *AviVsNode) {

	for i, pg := range vsNode.PoolGroupRefs {
		if pg.Name == pgName {
			utils.AviLog.Debugf("Removing pgRef: %s", pgName)
			vsNode.PoolGroupRefs = append(vsNode.PoolGroupRefs[:i], vsNode.PoolGroupRefs[i+1:]...)
			break
		}
	}
	utils.AviLog.Debugf("After removing the pg nodes are: %s", utils.Stringify(vsNode.PoolGroupRefs))

}

func (o *AviObjectGraph) RemoveHTTPRefsFromSni(httpPol string, sniNode *AviVsNode) {

	for i, pol := range sniNode.HttpPolicyRefs {
		if pol.Name == httpPol {
			utils.AviLog.Debugf("Removing http pol ref: %s", httpPol)
			sniNode.HttpPolicyRefs = append(sniNode.HttpPolicyRefs[:i], sniNode.HttpPolicyRefs[i+1:]...)
			break
		}
	}
	utils.AviLog.Debugf("After removing the http policy nodes are: %s", utils.Stringify(sniNode.HttpPolicyRefs))

}

func (o *AviObjectGraph) RemovePoolNodeRefsFromSni(poolName string, sniNode *AviVsNode) {

	for i, pool := range sniNode.PoolRefs {
		if pool.Name == poolName {
			utils.AviLog.Debugf("Removing pool ref: %s", poolName)
			sniNode.PoolRefs = append(sniNode.PoolRefs[:i], sniNode.PoolRefs[i+1:]...)
			break
		}
	}
	utils.AviLog.Debugf("After removing the pool ref nodes are: %s", utils.Stringify(sniNode.PoolRefs))

}

func (o *AviObjectGraph) RemovePoolRefsFromPG(poolName string, pgNode *AviPoolGroupNode) {
	if pgNode == nil {
		utils.AviLog.Warnf("cannot delete pool %s from nil PG node", poolName)
		return
	}
	for i, member := range pgNode.Members {
		if strings.TrimPrefix(*member.PoolRef, "/api/pool?name=") != poolName {
			continue
		}
		utils.AviLog.Debugf("Removing pool ref: %s from pg: %s", poolName, pgNode.Name)
		pgNode.Members = append(pgNode.Members[:i], pgNode.Members[i+1:]...)
		break
	}
	utils.AviLog.Debugf("After removing the pool %s, pg Members are: %s", poolName, utils.Stringify(pgNode.Members))
}

func (o *AviObjectGraph) GetOrderedNodes() []AviModelNode {
	return o.modelNodes
}

type AviVrfNode struct {
	Name             string
	StaticRoutes     []*avimodels.StaticRoute
	CloudConfigCksum uint32
}

func (v *AviVrfNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviVrfNode) GetNodeType() string {
	return "VrfNode"
}

func (v *AviVrfNode) CalculateCheckSum() {
	// A sum of fields for this vrf.
	v.CloudConfigCksum = lib.VrfChecksum(v.Name, v.StaticRoutes)
}

func (v *AviVrfNode) CopyNode() AviModelNode {
	newNode := AviVrfNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviVrfNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviVrfNode: %s", err)
	}
	return &newNode
}

func (o *AviObjectGraph) GetAviVRF() []*AviVrfNode {
	var aviVrf []*AviVrfNode
	for _, model := range o.modelNodes {
		vrf, ok := model.(*AviVrfNode)
		if ok {
			aviVrf = append(aviVrf, vrf)
		}
	}
	return aviVrf
}

type AviVsNode struct {
	Name                  string
	Tenant                string
	ServiceEngineGroup    string
	ApplicationProfile    string
	NetworkProfile        string
	Enabled               *bool
	EnableRhi             *bool
	PortProto             []AviPortHostProtocol // for listeners
	DefaultPool           string
	CloudConfigCksum      uint32
	DefaultPoolGroup      string
	HTTPChecksum          uint32
	SNIParent             bool
	PoolGroupRefs         []*AviPoolGroupNode
	PoolRefs              []*AviPoolNode
	HTTPDSrefs            []*AviHTTPDataScriptNode
	SniNodes              []*AviVsNode
	PassthroughChildNodes []*AviVsNode
	SharedVS              bool
	CACertRefs            []*AviTLSKeyCertNode
	SSLKeyCertRefs        []*AviTLSKeyCertNode
	HttpPolicyRefs        []*AviHttpPolicySetNode
	VSVIPRefs             []*AviVSVIPNode
	L4PolicyRefs          []*AviL4PolicyNode
	VHParentName          string
	VHDomainNames         []string
	TLSType               string
	IsSNIChild            bool
	ServiceMetadata       avicache.ServiceMetadataObj
	VrfContext            string
	WafPolicyRef          string
	AppProfileRef         string
	AnalyticsProfileRef   string
	ErrorPageProfileRef   string
	HttpPolicySetRefs     []string
	SSLProfileRef         string
	VsDatascriptRefs      []string
	SSLKeyCertAviRef      string
}

// Implementing AviVsEvhSniModel

func (v *AviVsNode) GetName() string {
	return v.Name
}

func (v *AviVsNode) SetName(Name string) {
	v.Name = Name
}

func (v *AviVsNode) GetPoolRefs() []*AviPoolNode {
	return v.PoolRefs
}

func (v *AviVsNode) SetPoolRefs(PoolRefs []*AviPoolNode) {
	v.PoolRefs = PoolRefs
}

func (v *AviVsNode) GetPoolGroupRefs() []*AviPoolGroupNode {
	return v.PoolGroupRefs
}

func (v *AviVsNode) SetPoolGroupRefs(poolGroupRefs []*AviPoolGroupNode) {
	v.PoolGroupRefs = poolGroupRefs
}

func (v *AviVsNode) GetSSLKeyCertRefs() []*AviTLSKeyCertNode {
	return v.SSLKeyCertRefs
}

func (v *AviVsNode) SetSSLKeyCertRefs(sslKeyCertRefs []*AviTLSKeyCertNode) {
	v.SSLKeyCertRefs = sslKeyCertRefs
}

func (v *AviVsNode) GetHttpPolicyRefs() []*AviHttpPolicySetNode {
	return v.HttpPolicyRefs
}

func (v *AviVsNode) SetHttpPolicyRefs(httpPolicyRefs []*AviHttpPolicySetNode) {
	v.HttpPolicyRefs = httpPolicyRefs
}

func (v *AviVsNode) GetServiceMetadata() avicache.ServiceMetadataObj {
	return v.ServiceMetadata
}

func (v *AviVsNode) SetServiceMetadata(serviceMetadata avicache.ServiceMetadataObj) {
	v.ServiceMetadata = serviceMetadata
}

func (v *AviVsNode) GetSSLKeyCertAviRef() string {
	return v.SSLKeyCertAviRef
}

func (v *AviVsNode) SetSSLKeyCertAviRef(sslKeyCertAviRef string) {
	v.SSLKeyCertAviRef = sslKeyCertAviRef
}

func (v *AviVsNode) GetWafPolicyRef() string {
	return v.WafPolicyRef
}

func (v *AviVsNode) SetWafPolicyRef(wafPolicyRef string) {
	v.WafPolicyRef = wafPolicyRef
}

func (v *AviVsNode) GetHttpPolicySetRefs() []string {
	return v.HttpPolicySetRefs
}

func (v *AviVsNode) SetHttpPolicySetRefs(httpPolicySetRefs []string) {
	v.HttpPolicySetRefs = httpPolicySetRefs
}

func (v *AviVsNode) GetAppProfileRef() string {
	return v.AppProfileRef
}

func (v *AviVsNode) SetAppProfileRef(appProfileRef string) {
	v.AppProfileRef = appProfileRef
}

func (v *AviVsNode) GetAnalyticsProfileRef() string {
	return v.AnalyticsProfileRef
}

func (v *AviVsNode) SetAnalyticsProfileRef(AnalyticsProfileRef string) {
	v.AnalyticsProfileRef = AnalyticsProfileRef
}

func (v *AviVsNode) GetErrorPageProfileRef() string {
	return v.ErrorPageProfileRef
}

func (v *AviVsNode) SetErrorPageProfileRef(ErrorPageProfileRef string) {
	v.ErrorPageProfileRef = ErrorPageProfileRef
}

func (v *AviVsNode) GetSSLProfileRef() string {
	return v.SSLProfileRef
}

func (v *AviVsNode) SetSSLProfileRef(SSLProfileRef string) {
	v.SSLProfileRef = SSLProfileRef
}

func (v *AviVsNode) GetVsDatascriptRefs() []string {
	return v.VsDatascriptRefs
}

func (v *AviVsNode) SetVsDatascriptRefs(VsDatascriptRefs []string) {
	v.VsDatascriptRefs = VsDatascriptRefs
}

func (v *AviVsNode) GetEnabled() *bool {
	return v.Enabled
}

func (v *AviVsNode) SetEnabled(Enabled *bool) {
	v.Enabled = Enabled
}

func (o *AviObjectGraph) GetAviVS() []*AviVsNode {
	var aviVs []*AviVsNode
	for _, model := range o.modelNodes {
		vs, ok := model.(*AviVsNode)
		if ok {
			aviVs = append(aviVs, vs)
		}
	}
	return aviVs
}

func (v *AviVsNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviVsNode) GetSniNodeForName(sniNodeName string) *AviVsNode {
	for _, sni := range v.SniNodes {
		if sni.Name == sniNodeName {
			return sni
		}
	}
	return nil
}

func (o *AviVsNode) CheckCACertNodeNameNChecksum(cacertNodeName string, checksum uint32) bool {
	for _, caCert := range o.CACertRefs {
		if caCert.Name == cacertNodeName {
			//Check if their checksums are same
			if caCert.GetCheckSum() == checksum {
				return false
			}
		}
	}
	return true
}

func (o *AviVsNode) CheckSSLCertNodeNameNChecksum(sslNodeName string, checksum uint32) bool {
	for _, sslCert := range o.SSLKeyCertRefs {
		if sslCert.Name == sslNodeName {
			//Check if their checksums are same
			if sslCert.GetCheckSum() == checksum {
				return false
			}
		}
	}
	return true
}

func (o *AviVsNode) CheckPGNameNChecksum(pgNodeName string, checksum uint32) bool {
	for _, pg := range o.PoolGroupRefs {
		if pg.Name == pgNodeName {
			//Check if their checksums are same
			if pg.GetCheckSum() == checksum {
				return false
			} else {
				return true
			}
		}
	}
	return true
}

func (o *AviVsNode) CheckPoolNChecksum(poolNodeName string, checksum uint32) bool {
	for _, pool := range o.PoolRefs {
		if pool.Name == poolNodeName {
			//Check if their checksums are same
			if pool.GetCheckSum() == checksum {
				return false
			}
		}
	}
	return true
}

func (o *AviVsNode) GetPGForVSByName(pgName string) *AviPoolGroupNode {
	for _, pgNode := range o.PoolGroupRefs {
		if pgNode.Name == pgName {
			return pgNode
		}
	}
	return nil
}

func (o *AviVsNode) ReplaceSniPoolInSNINode(newPoolNode *AviPoolNode, key string) {
	for i, pool := range o.PoolRefs {
		if pool.Name == newPoolNode.Name {
			o.PoolRefs = append(o.PoolRefs[:i], o.PoolRefs[i+1:]...)
			o.PoolRefs = append(o.PoolRefs, newPoolNode)
			utils.AviLog.Infof("key: %s, msg: replaced sni pool in model: %s Pool name: %s", key, o.Name, pool.Name)
			return
		}
	}
	// If we have reached here it means we haven't found a match. Just append the pool.
	o.PoolRefs = append(o.PoolRefs, newPoolNode)
	return
}

func (o *AviVsNode) ReplaceSniPGInSNINode(newPGNode *AviPoolGroupNode, key string) {
	for i, pg := range o.PoolGroupRefs {
		if pg.Name == newPGNode.Name {
			o.PoolGroupRefs = append(o.PoolGroupRefs[:i], o.PoolGroupRefs[i+1:]...)
			o.PoolGroupRefs = append(o.PoolGroupRefs, newPGNode)
			utils.AviLog.Infof("key: %s, msg: replaced sni pg in model: %s Pool name: %s", key, o.Name, pg.Name)
			return
		}
	}
	// If we have reached here it means we haven't found a match. Just append.
	o.PoolGroupRefs = append(o.PoolGroupRefs, newPGNode)
	return
}

func (o *AviVsNode) ReplaceSniHTTPRefInSNINode(newHttpNode *AviHttpPolicySetNode, key string) {
	for i, http := range o.HttpPolicyRefs {
		if http.Name == newHttpNode.Name {
			o.HttpPolicyRefs = append(o.HttpPolicyRefs[:i], o.HttpPolicyRefs[i+1:]...)
			o.HttpPolicyRefs = append(o.HttpPolicyRefs, newHttpNode)
			utils.AviLog.Infof("key: %s, msg: replaced sni http in model: %s Pool name: %s", key, o.Name, http.Name)
			return
		}
	}
	// If we have reached here it means we haven't found a match. Just append.
	o.HttpPolicyRefs = append(o.HttpPolicyRefs, newHttpNode)
	return
}

func (o *AviVsNode) DeleteCACertRefInSNINode(cacertNodeName, key string) {
	for i, cacert := range o.CACertRefs {
		if cacert.Name == cacertNodeName {
			o.CACertRefs = append(o.CACertRefs[:i], o.CACertRefs[i+1:]...)
			utils.AviLog.Infof("key: %s, msg: replaced cacert for sni in model: %s Pool name: %s", key, o.Name, cacert.Name)
			return
		}
	}
}

func (o *AviVsNode) ReplaceCACertRefInSNINode(cacertNode *AviTLSKeyCertNode, key string) {
	for i, cacert := range o.CACertRefs {
		if cacert.Name == cacertNode.Name {
			o.CACertRefs = append(o.CACertRefs[:i], o.CACertRefs[i+1:]...)
			o.CACertRefs = append(o.CACertRefs, cacertNode)
			utils.AviLog.Infof("key: %s, msg: replaced cacert for sni in model: %s Pool name: %s", key, o.Name, cacert.Name)
			return
		}
	}
	// If we have reached here it means we haven't found a match. Just append.
	o.CACertRefs = append(o.CACertRefs, cacertNode)
}

func (o *AviVsNode) ReplaceSniSSLRefInSNINode(newSslNode *AviTLSKeyCertNode, key string) {
	for i, ssl := range o.SSLKeyCertRefs {
		if ssl.Name == newSslNode.Name {
			o.SSLKeyCertRefs = append(o.SSLKeyCertRefs[:i], o.SSLKeyCertRefs[i+1:]...)
			o.SSLKeyCertRefs = append(o.SSLKeyCertRefs, newSslNode)
			utils.AviLog.Infof("key: %s, msg: replaced sni ssl in model: %s Pool name: %s", key, o.Name, ssl.Name)
			return
		}
	}
	// If we have reached here it means we haven't found a match. Just append.
	o.SSLKeyCertRefs = append(o.SSLKeyCertRefs, newSslNode)
	return
}

func (o *AviVsNode) CheckHttpPolNameNChecksum(httpNodeName string, checksum uint32) bool {
	for _, http := range o.HttpPolicyRefs {
		if http.Name == httpNodeName {
			//Check if their checksums are same
			if http.GetCheckSum() == checksum {
				return false
			}
		}
	}
	return true
}

func (v *AviVsNode) GetNodeType() string {
	// Calculate checksum and return
	return "VirtualServiceNode"
}

func (v *AviVsNode) CalculateCheckSum() {
	portproto := v.PortProto
	sort.Slice(portproto, func(i, j int) bool {
		return portproto[i].Name < portproto[j].Name
	})

	var checksumStringSlice []string

	for _, ds := range v.HTTPDSrefs {
		checksumStringSlice = append(checksumStringSlice, "HTTPDS"+ds.Name)
	}

	for _, httppol := range v.HttpPolicyRefs {
		checksumStringSlice = append(checksumStringSlice, "HttpPolicy"+httppol.Name)
	}

	for _, cacert := range v.CACertRefs {
		checksumStringSlice = append(checksumStringSlice, "CACert"+cacert.Name)
	}

	for _, sslkeycert := range v.SSLKeyCertRefs {
		checksumStringSlice = append(checksumStringSlice, "SSLKeyCert"+sslkeycert.Name)
	}

	for _, vsvipref := range v.VSVIPRefs {
		checksumStringSlice = append(checksumStringSlice, "VSVIP"+vsvipref.Name)
	}

	for _, l4policy := range v.L4PolicyRefs {
		checksumStringSlice = append(checksumStringSlice, "L4Policy"+l4policy.Name)
	}

	for _, vhdomain := range v.VHDomainNames {
		checksumStringSlice = append(checksumStringSlice, "VHDomain"+vhdomain)
	}

	// Note: Changing the order of strings being appended, while computing vsRefs and checksum,
	// will change the eventual checksum Hash.

	// keep the order of these policies
	policies := v.HttpPolicySetRefs
	scripts := v.VsDatascriptRefs

	vsRefs := v.WafPolicyRef +
		v.AppProfileRef +
		v.AnalyticsProfileRef +
		v.ErrorPageProfileRef +
		v.SSLProfileRef

	if len(scripts) > 0 {
		vsRefs += utils.Stringify(scripts)
	}

	if len(policies) > 0 {
		vsRefs += utils.Stringify(policies)
	}

	if len(v.ServiceMetadata.HostNames) > 0 {
		sort.Strings(v.ServiceMetadata.HostNames)
		vsRefs += utils.Stringify(v.ServiceMetadata.HostNames)
	}

	sort.Strings(checksumStringSlice)
	checksum := utils.Hash(strings.Join(checksumStringSlice, delim) +
		v.ApplicationProfile +
		v.ServiceEngineGroup +
		v.NetworkProfile +
		utils.Stringify(portproto))

	for _, sninode := range v.SniNodes {
		checksum += sninode.GetCheckSum()
	}
	for _, passthroughChild := range v.PassthroughChildNodes {
		checksum += passthroughChild.GetCheckSum()
	}

	if vsRefs != "" {
		checksum += utils.Hash(vsRefs)
	}

	if v.Enabled != nil {
		checksum += utils.Hash(utils.Stringify(v.Enabled))
	}

	if lib.GetGRBACSupport() {
		checksum += lib.GetClusterLabelChecksum()
	}

	if v.EnableRhi != nil {
		checksum += utils.Hash(utils.Stringify(*v.EnableRhi))
	}

	v.CloudConfigCksum = checksum
}

func (v *AviVsNode) CopyNode() AviModelNode {
	newNode := AviVsNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviVsNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviVsNode: %s", err)
	}
	return &newNode
}

type AviL4PolicyNode struct {
	Name             string
	Tenant           string
	CloudConfigCksum uint32
	PortPool         []AviHostPathPortPoolPG
}

func (v *AviL4PolicyNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviL4PolicyNode) CalculateCheckSum() {
	// A sum of fields for this VS.
	var checksum uint32
	var ports []int64
	for _, hpp := range v.PortPool {
		ports = append(ports, int64(hpp.Port))
	}
	if len(v.PortPool) > 0 {
		checksum = lib.L4PolicyChecksum(ports, v.PortPool[0].Protocol, nil, false)
	}
	v.CloudConfigCksum = checksum
}

func (v *AviL4PolicyNode) GetNodeType() string {
	// Calculate checksum and return
	return "AviL4PolicyNode"
}

func (v *AviL4PolicyNode) CopyNode() AviModelNode {
	newNode := AviL4PolicyNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviL4PolicyNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviL4PolicyNode: %s", err)
	}
	return &newNode
}

type AviHttpPolicySetNode struct {
	Name             string
	Tenant           string
	CloudConfigCksum uint32
	HppMap           []AviHostPathPortPoolPG
	RedirectPorts    []AviRedirectPort
	HeaderReWrite    *AviHostHeaderRewrite
	SecurityRules    []AviHTTPSecurity
}

func (v *AviHttpPolicySetNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviHttpPolicySetNode) CalculateCheckSum() {
	// A sum of fields for this VS.
	var checksum uint32
	for _, hpp := range v.HppMap {
		sort.Strings(hpp.Path)
		sort.Strings(hpp.Host)
		checksum = checksum + utils.Hash(utils.Stringify(hpp))

	}
	for _, redir := range v.RedirectPorts {
		sort.Strings(redir.Hosts)
		checksum = checksum + utils.Hash(utils.Stringify(redir.Hosts))
	}
	for _, sec_rule := range v.SecurityRules {
		checksum = checksum + utils.Hash(sec_rule.Action) + utils.Hash(sec_rule.MatchCriteria)
		checksum = checksum + uint32(sec_rule.Port)
	}
	if v.HeaderReWrite != nil {
		checksum = checksum + utils.Hash(utils.Stringify(v.HeaderReWrite))
	}
	if lib.GetGRBACSupport() {
		checksum += lib.GetClusterLabelChecksum()
	}
	v.CloudConfigCksum = checksum
}

func (v *AviHttpPolicySetNode) GetNodeType() string {
	// Calculate checksum and return
	return "HTTPPolicyNode"
}

func (v *AviHttpPolicySetNode) CopyNode() AviModelNode {
	newNode := AviHttpPolicySetNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviHttpPolicySetNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviHttpPolicySetNode: %s", err)
	}
	return &newNode
}

type AviHostPathPortPoolPG struct {
	Host          []string
	Path          []string
	Port          uint32
	Pool          string
	PoolGroup     string
	MatchCriteria string
	Protocol      string
}

type AviRedirectPort struct {
	Hosts        []string
	RedirectPort int32
	StatusCode   string
	VsPort       int32
}
type AviHTTPSecurity struct {
	Action        string
	MatchCriteria string
	Enable        bool
	Port          int64
}
type AviHostHeaderRewrite struct {
	SourceHost string
	TargetHost string
}

type AviTLSKeyCertNode struct {
	Name             string
	Tenant           string
	CloudConfigCksum uint32
	Key              []byte
	Cert             []byte
	CACert           string
	Port             int32
	Type             string
}

func (v *AviTLSKeyCertNode) CalculateCheckSum() {
	// A sum of fields for this SSL cert.
	checksum := lib.SSLKeyCertChecksum(v.Name, string(v.Cert), v.CACert, nil, false)
	v.CloudConfigCksum = checksum
}

func (v *AviTLSKeyCertNode) GetCheckSum() uint32 {
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviTLSKeyCertNode) GetNodeType() string {
	// Calculate checksum and return
	return "TLSCertNode"
}

func (v *AviTLSKeyCertNode) CopyNode() AviModelNode {
	newNode := AviTLSKeyCertNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviTLSKeyCertNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviTLSKeyCertNode: %s", err)
	}
	return &newNode
}

type AviPortHostProtocol struct {
	PortMap     map[string][]int32
	Port        int32
	Protocol    string
	Hosts       []string
	Secret      string
	Passthrough bool
	Redirect    bool
	EnableSSL   bool
	Name        string
}

type AviVSVIPNode struct {
	Name                    string
	Tenant                  string
	CloudConfigCksum        uint32
	FQDNs                   []string
	VrfContext              string
	IPAddress               string
	VipNetworks             []akov1alpha1.AviInfraSettingVipNetwork
	BGPPeerLabels           []string
	SecurePassthroughNode   *AviVsNode
	InsecurePassthroughNode *AviVsNode
	T1Lr                    string
}

func (v *AviVSVIPNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviVSVIPNode) CalculateCheckSum() {
	var checksum uint32
	if len(v.FQDNs) > 0 {
		sort.Strings(v.FQDNs)
		checksum = utils.Hash(utils.Stringify(v.FQDNs))
	}

	if v.IPAddress != "" {
		checksum += utils.Hash(v.IPAddress)
	}

	if len(v.VipNetworks) > 0 {
		var vipNetworkStringList []string
		for _, vipNetwork := range v.VipNetworks {
			vipNetworkStringList = append(vipNetworkStringList, vipNetwork.NetworkName+":"+vipNetwork.Cidr)
		}
		sort.Strings(vipNetworkStringList)
		checksum += utils.Hash(utils.Stringify(vipNetworkStringList))
	}

	if lib.GetGRBACSupport() {
		checksum += lib.GetClusterLabelChecksum()
	}

	if len(v.BGPPeerLabels) > 0 {
		sort.Strings(v.BGPPeerLabels)
		checksum += utils.Hash(utils.Stringify(v.BGPPeerLabels))
	}

	if v.T1Lr != "" {
		checksum += utils.Hash(v.T1Lr)
	}

	v.CloudConfigCksum = checksum
}

func (v *AviVSVIPNode) GetNodeType() string {
	return "VSVIPNode"
}

func (v *AviVSVIPNode) CopyNode() AviModelNode {
	newNode := AviVSVIPNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviVSVIPNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviVSVIPNode: %s", err)
	}
	return &newNode
}

type AviPoolGroupNode struct {
	Name                  string
	Tenant                string
	CloudConfigCksum      uint32
	Members               []*avimodels.PoolGroupMember
	Port                  string
	ImplicitPriorityLabel bool
}

func (v *AviPoolGroupNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviPoolGroupNode) CalculateCheckSum() {
	// A sum of fields for this PG.
	pgMembers := v.Members
	sort.Slice(pgMembers, func(i, j int) bool {
		return *pgMembers[i].PoolRef < *pgMembers[j].PoolRef
	})
	checksum := utils.Hash(utils.Stringify(pgMembers))
	if lib.GetGRBACSupport() {
		checksum += lib.GetClusterLabelChecksum()
	}
	v.CloudConfigCksum = checksum
}

func (o *AviObjectGraph) GetPoolGroupByName(pgName string) *AviPoolGroupNode {
	for _, model := range o.modelNodes {
		pg, ok := model.(*AviPoolGroupNode)
		if ok {
			if pg.Name == pgName {
				utils.AviLog.Debugf("Found PG with name: %s", pg.Name)
				return pg
			}
		}
	}
	return nil
}

func (v *AviPoolGroupNode) GetNodeType() string {
	return "PoolGroupNode"
}

func (v *AviPoolGroupNode) CopyNode() AviModelNode {
	newNode := AviPoolGroupNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviPoolGroupNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviPoolGroupNode: %s", err)
	}
	return &newNode
}

type AviHTTPDataScriptNode struct {
	Name             string
	Tenant           string
	CloudConfigCksum uint32
	PoolGroupRefs    []string
	ProtocolParsers  []string
	*DataScript
}

func (v *AviHTTPDataScriptNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviHTTPDataScriptNode) CalculateCheckSum() {
	// A sum of fields for this VS.
	checksum := lib.DSChecksum(v.PoolGroupRefs, nil, false)
	if lib.GetEnableCtrl2014Features() {
		checksum = utils.Hash(fmt.Sprint(checksum) + utils.HTTP_DS_SCRIPT_MODIFIED)
	}
	v.CloudConfigCksum = checksum
}

func (v *AviHTTPDataScriptNode) GetNodeType() string {
	return "HTTPDataScript"
}

func (v *AviHTTPDataScriptNode) CopyNode() AviModelNode {
	newNode := AviHTTPDataScriptNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviHTTPDataScriptNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviHTTPDataScriptNode: %s", err)
	}
	return &newNode
}

func (o *AviObjectGraph) GetAviHTTPDSNode() []*AviHTTPDataScriptNode {
	var aviDS []*AviHTTPDataScriptNode
	for _, model := range o.modelNodes {
		ds, ok := model.(*AviHTTPDataScriptNode)
		if ok {
			aviDS = append(aviDS, ds)
		}
	}
	return aviDS
}

type DataScript struct {
	Evt    string
	Script string
}

type AviPkiProfileNode struct {
	Name             string
	Tenant           string
	CloudConfigCksum uint32
	CACert           string
}

func (v *AviPkiProfileNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviPkiProfileNode) CalculateCheckSum() {
	checksum := lib.SSLKeyCertChecksum(v.Name, "", v.CACert, nil, false)
	v.CloudConfigCksum = checksum
}

type AviPoolNode struct {
	Name                   string
	Tenant                 string
	CloudConfigCksum       uint32
	Port                   int32
	TargetPort             int32
	PortName               string
	Servers                []AviPoolMetaServer
	Protocol               string
	LbAlgorithm            string
	LbAlgorithmHash        string
	LbAlgoHostHeader       string
	IngressName            string
	PriorityLabel          string
	ServiceMetadata        avicache.ServiceMetadataObj
	SniEnabled             bool
	SslProfileRef          string
	PkiProfile             *AviPkiProfileNode
	HealthMonitors         []string
	ApplicationPersistence string
	VrfContext             string
	T1Lr                   string // Only applicable to NSX-T cloud, if this value is set, we automatically should unset the VRF context value.
	Markers                map[string]string
}

func (v *AviPoolNode) GetCheckSum() uint32 {
	// Calculate checksum and return
	v.CalculateCheckSum()
	return v.CloudConfigCksum
}

func (v *AviPoolNode) CalculateCheckSum() {
	servers := v.Servers
	sort.Slice(servers, func(i, j int) bool {
		return *servers[i].Ip.Addr < *servers[j].Ip.Addr
	})
	// nodeNetworkMap is the placement nw details for the pool which is constant for the AKO instance.
	nodeNetworkMap, _ := lib.GetNodeNetworkMap()

	// A sum of fields for this Pool.
	checksumStringSlice := []string{
		v.Protocol,
		strconv.Itoa(int(v.Port)),
		v.PortName,
		utils.Stringify(servers),
		v.LbAlgorithm,
		v.LbAlgorithmHash,
		v.LbAlgoHostHeader,
		utils.Stringify(v.SniEnabled),
		v.SslProfileRef,
		v.PriorityLabel,
		utils.Stringify(nodeNetworkMap),
	}

	if len(v.ServiceMetadata.NamespaceServiceName) > 0 {
		sort.Strings(v.ServiceMetadata.NamespaceServiceName)
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.ServiceMetadata.NamespaceServiceName))
	}

	if len(v.ServiceMetadata.HostNames) > 0 {
		sort.Strings(v.ServiceMetadata.HostNames)
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.ServiceMetadata.HostNames))
	}

	chksumStr := fmt.Sprint(strings.Join(checksumStringSlice, delim))

	checksum := utils.Hash(chksumStr)

	if len(v.HealthMonitors) > 0 {
		checksum += utils.Hash(utils.Stringify(v.HealthMonitors))
	}

	if v.PkiProfile != nil {
		checksum += v.PkiProfile.GetCheckSum()
	}

	if v.ApplicationPersistence != "" {
		checksum += utils.Hash(v.ApplicationPersistence)
	}

	if lib.GetGRBACSupport() {
		checksum += lib.GetClusterLabelChecksum()
	}
	if v.T1Lr != "" {
		checksum += utils.Hash(v.T1Lr)
	}

	v.CloudConfigCksum = checksum
}

func (v *AviPoolNode) GetNodeType() string {
	return "PoolNode"
}

func (v *AviPoolNode) CopyNode() AviModelNode {
	newNode := AviPoolNode{}
	bytes, err := json.Marshal(v)
	if err != nil {
		utils.AviLog.Warnf("Unable to marshal AviPoolNode: %s", err)
	}
	err = json.Unmarshal(bytes, &newNode)
	if err != nil {
		utils.AviLog.Warnf("Unable to unmarshal AviPoolNode: %s", err)
	}
	return &newNode
}

func (o *AviObjectGraph) GetAviPoolNodesByIngress(tenant string, ingName string) []*AviPoolNode {
	var aviPool []*AviPoolNode
	for _, model := range o.modelNodes {
		if model.GetNodeType() == "VirtualServiceNode" {
			for _, pool := range model.(*AviVsNode).PoolRefs {
				if pool.IngressName == ingName && tenant == pool.ServiceMetadata.Namespace {
					utils.AviLog.Debugf("Found Pool with name: %s Adding...", pool.IngressName)
					aviPool = append(aviPool, pool)
				}
			}
		}
	}
	return aviPool
}

func (o *AviObjectGraph) GetAviPoolNodeByName(poolname string) *AviPoolNode {
	for _, model := range o.modelNodes {
		if model.GetNodeType() == "VirtualServiceNode" {
			for _, pool := range model.(*AviVsNode).PoolRefs {
				if pool.Name == poolname {
					utils.AviLog.Debugf("Found Pool with name: %s", pool.Name)
					return pool
				}
			}
		}
	}
	return nil
}

type AviPoolMetaServer struct {
	Ip         avimodels.IPAddr
	ServerNode string
	Port       int32
}

type IngressHostPathSvc struct {
	ServiceName string
	Path        string
	PathType    networkingv1beta1.PathType
	Port        int32
	weight      int32 //required for alternate backends in openshift route
	PortName    string
	TargetPort  int32
}

type IngressHostMap map[string]HostMetadata

type HostMetadata struct {
	ingressHPSvc   []IngressHostPathSvc
	gslbHostHeader string
}

type TlsSettings struct {
	Hosts            map[string]HostMetadata
	SecretName       string
	SecretNS         string
	key              string
	cert             string
	cacert           string
	destCA           string //for reencrypt
	reencrypt        bool
	redirect         bool
	blockHTTPTraffic bool
	//tlstype    string
}

type PassthroughSettings struct {
	PathSvc  []IngressHostPathSvc
	host     string
	redirect bool
	//tlstype    string
}

type IngressConfig struct {
	PassthroughCollection map[string]PassthroughSettings
	TlsCollection         []TlsSettings
	IngressHostMap
	InsecureEdgeTermAllow bool
}

type SecureHostNameMapProp struct {
	// This method is only used in case of hostname based sharding. Hostname sharding uses a single thread in layer 2
	// Hence locking is avoided. Secondly, hostname based shards are agnostic of namespaces, hence namespaces are kept only as a
	// naming constuct. Only used for secure hosts.
	// hostname1(this is persisted in the store) --> ingress1 + ns --> path: [/foo, /bar], secrets: [secret1]
	// 			 --> ingress2 + ns --> path: [/baz], secrets: [secret3]
	HostNameMap map[string]HostNamePathSecrets
}

func NewSecureHostNameMapProp() SecureHostNameMapProp {
	hostNameMap := SecureHostNameMapProp{HostNameMap: make(map[string]HostNamePathSecrets)}
	return hostNameMap
}

func (h *SecureHostNameMapProp) GetPathsForHostName(hostname string) []string {
	var paths []string
	for _, v := range h.HostNameMap {
		paths = append(paths, v.paths...)
	}
	return paths
}

func (h *SecureHostNameMapProp) GetIngressesForHostName(hostname string) []string {
	var ingresses []string
	for k := range h.HostNameMap {
		ingresses = append(ingresses, k)
	}
	return ingresses
}

func (h *SecureHostNameMapProp) GetSecretsForHostName(hostname string) []string {
	var secrets []string
	for _, v := range h.HostNameMap {
		secrets = append(secrets, v.secretName)
	}
	return secrets
}

type HostNamePathSecrets struct {
	secretName string
	paths      []string
}
