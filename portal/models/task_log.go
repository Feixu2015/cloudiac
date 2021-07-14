package models

import (
	"cloudiac/portal/libs/db"
	"time"
)

type DBStorage struct {
	Id        uint   `gorm:"primary_key" json:"-"`
	Path      string `gorm:"NOT NULL;UNIQUE"`
	Content   []byte `gorm:"type:MEDIUMBLOB"` // MEDIUMBLOB 支持最大长度约 16M
	CreatedAt time.Time
}

func (DBStorage) TableName() string {
	return "iac_storage"
}

func (DBStorage) Migrate(s *db.Session) error {
	if err := s.DB().ModifyColumn("content", "MEDIUMBLOB").Error; err != nil {
		return err
	}
	return nil
}

func (DBStorage) Validate() error {
	return nil
}

func (DBStorage) ValidateAttrs(attrs Attrs) error {
	return nil
}