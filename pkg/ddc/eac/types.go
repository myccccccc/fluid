/*
  Copyright 2022 The Fluid Authors.

  Licensed under the Apache License, Version 2.0 (the "License");
  you may not use this file except in compliance with the License.
  You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

  Unless required by applicable law or agreed to in writing, software
  distributed under the License is distributed on an "AS IS" BASIS,
  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
  See the License for the specific language governing permissions and
  limitations under the License.
*/

package eac

import (
	"github.com/fluid-cloudnative/fluid/pkg/common"
	v1 "k8s.io/api/core/v1"
)

// The value yaml file
type EAC struct {
	FullnameOverride string          `yaml:"fullnameOverride"`
	PlacementMode    string          `yaml:"placement,omitempty"`
	Master           Master          `yaml:"master"`
	Worker           Worker          `yaml:"worker"`
	Fuse             Fuse            `yaml:"fuse"`
	InitFuse         InitFuse        `yaml:"initFuse"`
	OSAdvise         OSAdvise        `yaml:"osAdvise"`
	Tolerations      []v1.Toleration `yaml:"tolerations,omitempty"`
}

type OSAdvise struct {
	OSVersion string `yaml:"osVersion,omitempty"`
	Enabled   bool   `yaml:"enabled"`
}

type Master struct {
	common.ImageInfo `yaml:",inline"`
	MountPoint       string            `yaml:"mountPoint,omitempty"`
	Replicas         int32             `yaml:"count,omitempty"`
	Enabled          bool              `yaml:"enabled"`
	Options          string            `yaml:"option,omitempty"`
	Resources        common.Resources  `yaml:"resources,omitempty"`
	NodeSelector     map[string]string `yaml:"nodeSelector,omitempty"`
	HostNetwork      bool              `yaml:"hostNetwork"`
	TieredStore      TieredStore       `yaml:"tieredstore,omitempty"`
}

type Worker struct {
	common.ImageInfo `yaml:",inline"`
	Port             Port              `yaml:"port,omitempty"`
	Enabled          bool              `yaml:"enabled"`
	Options          string            `yaml:"option,omitempty"`
	Resources        common.Resources  `yaml:"resources,omitempty"`
	NodeSelector     map[string]string `yaml:"nodeSelector,omitempty"`
	HostNetwork      bool              `yaml:"hostNetwork"`
	TieredStore      TieredStore       `yaml:"tieredstore,omitempty"`
}

type Fuse struct {
	common.ImageInfo `yaml:",inline"`
	MountPoint       string            `yaml:"mountPoint,omitempty"`
	HostMountPath    string            `yaml:"hostMountPath,omitempty"`
	Port             Port              `yaml:"port,omitempty"`
	Options          string            `yaml:"option,omitempty"`
	Resources        common.Resources  `yaml:"resources,omitempty"`
	NodeSelector     map[string]string `yaml:"nodeSelector,omitempty"`
	HostNetwork      bool              `yaml:"hostNetwork"`
	TieredStore      TieredStore       `yaml:"tieredstore,omitempty"`
	CriticalPod      bool              `yaml:"criticalPod"`
}

type InitFuse struct {
	common.ImageInfo `yaml:",inline"`
}

type TieredStore struct {
	Levels []Level `yaml:"levels,omitempty"`
}

type Level struct {
	Alias      string `yaml:"alias,omitempty"`
	Level      int    `yaml:"level"`
	MediumType string `yaml:"mediumtype,omitempty"`
	Type       string `yaml:"type,omitempty"`
	Path       string `yaml:"path,omitempty"`
	Quota      string `yaml:"quota,omitempty"`
	High       string `yaml:"high,omitempty"`
	Low        string `yaml:"low,omitempty"`
}

type Port struct {
	Rpc     int `yaml:"rpc,omitempty"`
	Monitor int `yaml:"monitor,omitempty"`
}

type WorkerEndPoints struct {
	ContainerEndPoints []string `json:"containerendpoints,omitempty"`
}

type cacheHitStates struct {
	cacheHitRatio  string
	localHitRatio  string
	remoteHitRatio string

	localThroughputRatio  string
	remoteThroughputRatio string
	cacheThroughputRatio  string

	//bytesReadLocal int64
	//bytesReadRemote int64
	//bytesReadUfsAll int64

	//timestamp time.Time
}

type cacheStates struct {
	cacheCapacity string
	// cacheable        string
	// lowWaterMark     string
	// highWaterMark    string
	cached           string
	cachedPercentage string
	cacheHitStates   cacheHitStates
	// nonCacheable     string
}

type MountInfo struct {
	MountPoint      string
	ServiceAddr     string
	FileSystemId    string
	DirPath         string
	AccessKeyID     string
	AccessKeySecret string
}

func (value *EAC) getTiredStoreLevel0Path() (path string) {
	for _, level := range value.Worker.TieredStore.Levels {
		if level.Level == 0 {
			path = level.Path
			break
		}
	}
	return
}

func (value *EAC) getTiredStoreLevel0Type() (t string) {
	for _, level := range value.Worker.TieredStore.Levels {
		if level.Level == 0 {
			t = level.Type
			break
		}
	}
	return
}

func (value *EAC) getTiredStoreLevel0Quota() (quota string) {
	for _, level := range value.Worker.TieredStore.Levels {
		if level.Level == 0 {
			quota = level.Quota
			break
		}
	}
	return
}
