package forms

import "cloudiac/portal/models"

type DeclareEnvForm struct {
	BaseForm
	AppStack  string      `json:"appStack"`
	Cloud     string      `json:"cloud"`
	Region    string      `json:"region"`
	Zone      string      `json:"zone"`
	Instances instance    `json:"instances"`
	ExtraData models.JSON `json:"extraData"`
}

type instance struct {
	InstanceNumber      string `json:"instanceNumber"`
	ChargeType          string `json:"chargeType"`
	InstanceUnit        string `json:"instanceUnit"`
	SysDiskCategory     string `json:"sysDiskCategory"`
	SysDiskPerformance  string `json:"sysDiskPerformance"`
	SysDiskSize         string `json:"sysDiskSize"`
	DataDiskSize        string `json:"dataDiskSize"`
	DataDiskCategory    string `json:"dataDiskCategory"`
	DataDiskPerformance string `json:"dataDiskPerformance"`
	InstanceType        string `json:"instanceType"`
	ImageId             string `json:"imageId"`
	InstanceChargeType  string `json:"instanceChargeType"`
	UserData            string `json:"userData"`
	Tags                string `json:"tags"`
	FirstIndex          string `json:"firstIndex"`
	EnvironmentId       string `json:"environmentId"`
	KeyName             string `json:"keyName"`
}
