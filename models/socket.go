package models

import (
	"erinyes/conf"
	"erinyes/helper"
	"erinyes/logs"
	"gorm.io/gorm"
)

type Socket struct {
	ID            int    `gorm:"primaryKey;column:id"`
	HostID        string `gorm:"column:host_id"`
	HostName      string `gorm:"column:host_name"`
	ContainerID   string `gorm:"column:container_id"`
	ContainerName string `gorm:"column:container_name"`
	DstIP         string `gorm:"column:dst_ip"`
	DstPort       string `gorm:"column:dst_port"`
}

func (Socket) TableName() string {
	return "socket"
}

func (s *Socket) FindByID(db *gorm.DB, id int) bool {
	err := db.First(s, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			logs.Logger.Errorf("can't find socket by id = %d", id)
		} else {
			logs.Logger.Errorf("query socket by id = %d failed: %w", id, err)
		}
		return false
	}
	return true
}

func (s Socket) VertexClusterID() string {
	return helper.AddQuotation("cluster" + s.HostID + "_" + s.ContainerID)
}

func (s Socket) VertexName() string {
	return helper.AddQuotation(s.DstIP + ":" + s.DstPort + "#" + s.HostID + "_" + s.ContainerID)
}

func (s Socket) VertexShape() string {
	return "diamond"
}

func (s *Socket) RelateHostAndCin() {
	if s.DstIP == conf.Config.Cin0IP {
		s.DstIP = conf.Config.HostIP
		s.DstPort = "8085"
	} else if s.DstIP == conf.Config.HostIP {
		s.DstPort = "8085"
	} else if s.DstIP == "127.0.0.1" {
		s.DstIP = conf.Config.HostIP
		s.DstPort = "8085"
	}
}

func (s *Socket) UnionGateway() {
	gateways := conf.Config.GatewayMap
	if _, exist := gateways[s.DstIP]; exist {
		s.DstIP = "gateway"
		s.DstPort = "8080"
	}
}
