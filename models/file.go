package models

import (
	"erinyes/helper"
	"erinyes/logs"
	"gorm.io/gorm"
)

type File struct {
	ID            int    `gorm:"primaryKey;column:id"`
	HostID        string `gorm:"column:host_id"`
	HostName      string `gorm:"column:host_name"`
	ContainerID   string `gorm:"column:container_id"`
	ContainerName string `gorm:"column:container_name"`
	FilePath      string `gorm:"column:file_path"`
}

func (File) TableName() string {
	return "file"
}

func (f *File) FindByID(db *gorm.DB, id int) bool {
	err := db.First(f, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logs.Logger.Errorf("can't find file by id = %d", id)
		} else {
			logs.Logger.Errorf("query file by id = %d failed: %w", id, err)
		}
		return false
	}
	return true
}

func (f File) VertexClusterID() string {
	return helper.AddQuotation("cluster" + f.HostID + "_" + f.ContainerID)
}

func (f File) VertexName() string {
	return helper.AddQuotation(f.FilePath + "#" + f.HostID + "_" + f.ContainerID)
}

func (f File) VertexShape() string {
	return "ellipse"
}
