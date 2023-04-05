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

package v1alpha2


type AnalyticsPolicy struct {
	FullClientLogs *FullClientLogs `json:"fullClientLogs,omitempty"`
	UdfLogThrottle *int32          `json:"udfLogThrottle,omitempty"`
}

type BackendProperties struct {
	AnalyticsPolicy                  *PoolAnalyticsPolicy `json:"analyticsPolicy,omitempty"`
	AnalyticsProfileRef              *string              `json:"analyticsProfileRef,omitempty"`
	ApplicationPersistenceProfileRef *string              `json:"applicationPersistenceProfileRef,omitempty"`
	DefaultServerPort                *int32               `json:"defaultServerPort,omitempty"`
	Enabled                          *bool                `json:"enabled,omitempty"`
	HealthMonitorRefs                []string             `json:"healthMonitorRefs,omitempty"`
	InlineHealthMonitor              *bool                `json:"inlineHealthMonitor,omitempty"`
	LbAlgorithm                      *string              `json:"lbAlgorithm,omitempty"`
	LbAlgorithmHash                  *string              `json:"lbAlgorithmHash,omitempty"`
	MinServersUp                     *int32               `json:"minServersUp,omitempty"`
	PkiProfileRef                    *string              `json:"pkiProfileRef,omitempty"`
	Port                             *int                 `json:"port"`
	Protocol                         *string              `json:"protocol"`
	SslKeyAndCertificateRef          *string              `json:"sslKeyAndCertificateRef,omitempty"`
	SslProfileRef                    *string              `json:"sslProfileRef,omitempty"`
	UseServicePort                   *bool                `json:"useServicePort,omitempty"`
}

type FullClientLogs struct {
	Duration *int32 `json:"duration,omitempty"`
	Enabled  *bool  `json:"enabled"`
	Throttle *int32 `json:"throttle,omitempty"`
}

type PoolAnalyticsPolicy struct {
	EnableRealtimeMetrics *bool `json:"enableRealtimeMetrics,omitempty"`
}

