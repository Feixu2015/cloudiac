// Copyright (c) 2015-2022 CloudJ Technology Co., Ltd.

package alicloud

import (
	"cloudiac/portal/services/forecast/reource/alicloud"
	"cloudiac/portal/services/forecast/schema"
)

func getDBInstanceRegistryItem() *schema.RegistryItem {
	return &schema.RegistryItem{
		Name:  "alicloud_db_instance",
		Notes: []string{},
		RFunc: NewDBInstance,
	}
}

func NewDBInstance(d *schema.ResourceData) *schema.Resource {
	region := d.Get("region").String()

	a := &alicloud.DBInstance{
		Address:               d.Address,
		Provider:              d.ProviderName,
		Region:                region,
		InstanceType:          d.Get("instance_type").String(),
		DbInstanceStorageType: d.Get("db_instance_storage_type").String(),
		InstanceStorage:       d.Get("instance_storage").Int(),
		Engine:                d.Get("engine").String(),
		EngineVersion:         d.Get("engine_version").String(),
	}

	return a.BuildResource()

}