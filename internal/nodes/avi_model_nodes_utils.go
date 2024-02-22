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

// Code generated by a tool; DO NOT EDIT.

package nodes

import (
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/apis/ako/v1alpha2"
	"github.com/vmware/load-balancer-and-ingress-services-for-kubernetes/pkg/utils"
)

type AviVsNodeGeneratedFields struct {
	AllowInvalidClientCert        *bool
	BotPolicyRef                  *string
	CloseClientConnOnConfigUpdate *bool
	Fqdn                          *string
	HostNameXlate                 *string
	IgnPoolNetReach               *bool
	LoadBalancerIP                *string
	MinPoolsUp                    *uint32
	NetworkProfileRef             *string
	NetworkSecurityPolicyRef      *string
	OauthVsConfig                 *v1alpha2.OAuthVSConfig
	PerformanceLimits             *v1alpha2.PerformanceLimits
	RemoveListeningPortOnVsDown   *bool
	SamlSpConfig                  *v1alpha2.SAMLSPConfig
	SecurityPolicyRef             *string
	Services                 []*v1alpha2.Service
	SslSessCacheAvgSize           *uint32
	SsoPolicyRef                  *string
	TrafficCloneProfileRef        *string
}

func (v *AviVsNodeGeneratedFields) CalculateCheckSumOfGeneratedCode() uint32 {
	checksumStringSlice := make([]string, 0, 20)
	if v.AllowInvalidClientCert != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.AllowInvalidClientCert))
	}

	if v.BotPolicyRef != nil {
		checksumStringSlice = append(checksumStringSlice, *v.BotPolicyRef)
	}

	if v.CloseClientConnOnConfigUpdate != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.CloseClientConnOnConfigUpdate))
	}

	if v.Fqdn != nil {
		checksumStringSlice = append(checksumStringSlice, *v.Fqdn)
	}

	if v.HostNameXlate != nil {
		checksumStringSlice = append(checksumStringSlice, *v.HostNameXlate)
	}

	if v.IgnPoolNetReach != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.IgnPoolNetReach))
	}

	if v.LoadBalancerIP != nil {
		checksumStringSlice = append(checksumStringSlice, *v.LoadBalancerIP)
	}

	if v.MinPoolsUp != nil {
		checksumStringSlice = append(checksumStringSlice, strconv.Itoa(int(*v.MinPoolsUp)))
	}

	if v.NetworkProfileRef != nil {
		checksumStringSlice = append(checksumStringSlice, *v.NetworkProfileRef)
	}

	if v.NetworkSecurityPolicyRef != nil {
		checksumStringSlice = append(checksumStringSlice, *v.NetworkSecurityPolicyRef)
	}

	if v.OauthVsConfig != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.OauthVsConfig))
	}

	if v.PerformanceLimits != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.PerformanceLimits))
	}

	if v.RemoveListeningPortOnVsDown != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.RemoveListeningPortOnVsDown))
	}

	if v.SamlSpConfig != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.SamlSpConfig))
	}

	if v.SecurityPolicyRef != nil {
		checksumStringSlice = append(checksumStringSlice, *v.SecurityPolicyRef)
	}

	if v.SslSessCacheAvgSize != nil {
		checksumStringSlice = append(checksumStringSlice, strconv.Itoa(int(*v.SslSessCacheAvgSize)))
	}

	if v.Services != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.Services))
	}

	if v.SsoPolicyRef != nil {
		checksumStringSlice = append(checksumStringSlice, *v.SsoPolicyRef)
	}

	if v.TrafficCloneProfileRef != nil {
		checksumStringSlice = append(checksumStringSlice, *v.TrafficCloneProfileRef)
	}

	chksumStr := strings.Join(checksumStringSlice, delim)
	checksum := utils.Hash(chksumStr)
	return checksum
}

func (o *AviVsNodeGeneratedFields) ConvertToRef() {
	if o != nil {
		if o.BotPolicyRef != nil {
			o.BotPolicyRef = proto.String("/api/botpolicy?name=" + *o.BotPolicyRef)
		}
		if o.NetworkProfileRef != nil {
			o.NetworkProfileRef = proto.String("/api/networkprofile?name=" + *o.NetworkProfileRef)
		}
		if o.NetworkSecurityPolicyRef != nil {
			o.NetworkSecurityPolicyRef = proto.String("/api/networksecuritypolicy?name=" + *o.NetworkSecurityPolicyRef)
		}
		if o.OauthVsConfig != nil {
			for ii := range o.OauthVsConfig.OauthSettings {
				if o.OauthVsConfig.OauthSettings[ii] != nil {
					if o.OauthVsConfig.OauthSettings[ii].AuthProfileRef != nil {
						o.OauthVsConfig.OauthSettings[ii].AuthProfileRef = proto.String("/api/authprofile?name=" + *o.OauthVsConfig.OauthSettings[ii].AuthProfileRef)
					}
				}
			}
		}
		if o.SamlSpConfig != nil {
			if o.SamlSpConfig.SigningSslKeyAndCertificateRef != nil {
				o.SamlSpConfig.SigningSslKeyAndCertificateRef = proto.String("/api/sslkeyandcertificate?name=" + *o.SamlSpConfig.SigningSslKeyAndCertificateRef)
			}
		}
		if o.SecurityPolicyRef != nil {
			o.SecurityPolicyRef = proto.String("/api/securitypolicy?name=" + *o.SecurityPolicyRef)
		}
		if o.SsoPolicyRef != nil {
			o.SsoPolicyRef = proto.String("/api/ssopolicy?name=" + *o.SsoPolicyRef)
		}
		if o.TrafficCloneProfileRef != nil {
			o.TrafficCloneProfileRef = proto.String("/api/trafficcloneprofile?name=" + *o.TrafficCloneProfileRef)
		}
	}
}

func (o *AviVsNodeCommonFields) ConvertToRef() {
	if o != nil {
		if o.AnalyticsProfileRef != nil {
			o.AnalyticsProfileRef = proto.String("/api/analyticsprofile?name=" + *o.AnalyticsProfileRef)
		}
		if o.ApplicationProfileRef != nil {
			o.ApplicationProfileRef = proto.String("/api/applicationprofile?name=" + *o.ApplicationProfileRef)
		}
		SslKeyAndCertificateRefs := make([]string, 0, len(o.SslKeyAndCertificateRefs))
		for i := range o.SslKeyAndCertificateRefs {
			ref := fmt.Sprintf("/api/sslkeyandcertificate?name=" + o.SslKeyAndCertificateRefs[i])
			if !utils.HasElem(SslKeyAndCertificateRefs, ref) {
				SslKeyAndCertificateRefs = append(SslKeyAndCertificateRefs, ref)
			}
		}
		o.SslKeyAndCertificateRefs = SslKeyAndCertificateRefs
		if o.SslProfileRef != nil {
			o.SslProfileRef = proto.String("/api/sslprofile?name=" + *o.SslProfileRef)
		}
		VsDatascriptRefs := make([]string, 0, len(o.VsDatascriptRefs))
		for i := range o.VsDatascriptRefs {
			ref := fmt.Sprintf("/api/vsdatascriptset?name=" + o.VsDatascriptRefs[i])
			if !utils.HasElem(VsDatascriptRefs, ref) {
				VsDatascriptRefs = append(VsDatascriptRefs, ref)
			}
		}
		o.VsDatascriptRefs = VsDatascriptRefs
		if o.WafPolicyRef != nil {
			o.WafPolicyRef = proto.String("/api/wafpolicy?name=" + *o.WafPolicyRef)
		}
	}
}

func (o *AviVsNodeGeneratedFields) ConvertL7RuleFieldsToNil() {
	if o != nil {
		o.AllowInvalidClientCert = nil
		o.BotPolicyRef =nil
		o.CloseClientConnOnConfigUpdate =nil
		o.HostNameXlate =nil
		o.IgnPoolNetReach =nil
		o.MinPoolsUp =nil
		o.PerformanceLimits =nil
		o.RemoveListeningPortOnVsDown =nil
		o.SslSessCacheAvgSize =nil
		o.TrafficCloneProfileRef =nil
		o.SecurityPolicyRef =nil
	}
}

func (o *AviVsNodeGeneratedFields) ConvertL7RuleParentOnlyFieldsToNil(){
	if o != nil {
		o.HostNameXlate = nil
		o.SecurityPolicyRef =nil
		o.PerformanceLimits =nil

	}
}

type AviPoolGeneratedFields struct {
	AnalyticsPolicy *v1alpha2.PoolAnalyticsPolicy
	Enabled         *bool
	MinServersUp    *uint32
}

func (v *AviPoolGeneratedFields) CalculateCheckSumOfGeneratedCode() uint32 {
	checksumStringSlice := make([]string, 0, 3)
	if v.AnalyticsPolicy != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.AnalyticsPolicy))
	}

	if v.Enabled != nil {
		checksumStringSlice = append(checksumStringSlice, utils.Stringify(v.Enabled))
	}

	if v.MinServersUp != nil {
		checksumStringSlice = append(checksumStringSlice, strconv.Itoa(int(*v.MinServersUp)))
	}

	chksumStr := strings.Join(checksumStringSlice, delim)
	checksum := utils.Hash(chksumStr)
	return checksum
}

func (o *AviPoolGeneratedFields) ConvertToRef() {
}

func (o *AviPoolCommonFields) ConvertToRef() {
	if o != nil {
		if o.ApplicationPersistenceProfileRef != nil {
			o.ApplicationPersistenceProfileRef = proto.String("/api/applicationpersistenceprofile?name=" + *o.ApplicationPersistenceProfileRef)
		}
		HealthMonitorRefs := make([]string, 0, len(o.HealthMonitorRefs))
		for i := range o.HealthMonitorRefs {
			ref := fmt.Sprintf("/api/healthmonitor?name=" + o.HealthMonitorRefs[i])
			if !utils.HasElem(HealthMonitorRefs, ref) {
				HealthMonitorRefs = append(HealthMonitorRefs, ref)
			}
		}
		o.HealthMonitorRefs = HealthMonitorRefs
		if o.PkiProfileRef != nil {
			o.PkiProfileRef = proto.String("/api/pkiprofile?name=" + *o.PkiProfileRef)
		}
		if o.SslKeyAndCertificateRef != nil {
			o.SslKeyAndCertificateRef = proto.String("/api/sslkeyandcertificate?name=" + *o.SslKeyAndCertificateRef)
		}
		if o.SslProfileRef != nil {
			o.SslProfileRef = proto.String("/api/sslprofile?name=" + *o.SslProfileRef)
		}
	}
}

