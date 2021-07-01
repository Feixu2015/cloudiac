package models

import "cloudiac/portal/libs/db"

type Project struct {
	SoftDeleteModel

	OrgId       Id     `gorm:"size:32;not null"`
	Name        string `gorm:"not null;"`
	Description string `json:"description" gorm:"type:text"`
	CreatorId   Id     `json:"creatorId" form:"creatorId" `
	Status      string `json:"status" gorm:"type:enum('enable','disable');default:'enable';comment:'状态'"`
}

func (Project) TableName() string {
	return "iac_project"
}

func (p *Project) Migrate(sess *db.Session) (err error) {
	if err := p.AddUniqueIndex(sess,
		"unique__org__project__name", "org_id", "name"); err != nil {
		return err
	}
	return nil
}
