package models

import (
	"cloudiac/libs/db"
)

const (
	SysCfgNameMaxJobsPerRunner = "MAX_JOBS_PER_RUNNER"
	SysCfgNamePeriodOfLogSave  = "PERIOD_OF_LOG_SAVE"
)

type SystemCfg struct {
	BaseModel

	Name        string `json:"name" gorm:"not null;comment:'设定名'"`
	Value       string `json:"value" gorm:"size:32;not null;comment:'设定值'"`
	Description string `json:"description" gorm:"size:32;comment:'描述'"`
}

func (SystemCfg) TableName() string {
	return "iac_system_cfg"
}

func (o SystemCfg) Migrate(sess *db.Session) (err error) {
	if err != nil {
		return err
	}

	return nil
}
