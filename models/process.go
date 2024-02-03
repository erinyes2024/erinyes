package models

import (
	"erinyes/helper"
	"erinyes/logs"
	"gorm.io/gorm"
)

type Process struct {
	ID             int    `gorm:"primaryKey;column:id"`
	HostID         string `gorm:"column:host_id"`
	HostName       string `gorm:"column:host_name"`
	ContainerID    string `gorm:"column:container_id"`
	ContainerName  string `gorm:"column:container_name"`
	ProcessVPID    string `gorm:"column:process_vpid"`
	ProcessName    string `gorm:"column:process_name"`
	ProcessExepath string `gorm:"column:process_exe_path"`
}

func (Process) TableName() string {
	return "process"
}

func (p *Process) FindByID(db *gorm.DB, id int) bool {
	err := db.First(p, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logs.Logger.Errorf("can't find process by id = %d", id)
		} else {
			logs.Logger.Errorf("query process by id = %d failed: %w", id, err)
		}
		return false
	}
	return true
}

func (p Process) VertexClusterID() string {
	return helper.AddQuotation("cluster" + p.HostID + "_" + p.ContainerID)
}

func (p Process) VertexName() string {
	return helper.AddQuotation(p.ProcessVPID + "_" + p.ProcessName + "#" + p.HostID + "_" + p.ContainerID)
}

func (p Process) VertexShape() string {
	return "box"
}
