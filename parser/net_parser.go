package parser

import (
	"erinyes/conf"
	"erinyes/logs"
	"strings"
)

type NetParser struct {
	pusher *Pusher
}

func NewNetParser(pusher *Pusher) *NetParser {
	return &NetParser{
		pusher: pusher,
	}
}

func (p *NetParser) ParserType() string {
	return NET
}

func (p *NetParser) ParsePushLine(rawLine string) error {
	err, netLog := SplitNetLine(rawLine)
	if err != nil {
		return err
	}

	pl := ParsedLog{}
	if containerNameAndID, ok := conf.Config.IPMap[netLog.IPSrc]; ok {
		result := strings.Split(containerNameAndID, "$")
		//pl.StartVertex = SocketVertex{
		//	HostID:        conf.MockHostID,
		//	HostName:      conf.MockHostName,
		//	ContainerID:   result[1],
		//	ContainerName: result[0],
		//	DstIP:         netLog.IPSrc,
		//	DstPort:       netLog.PortSrc,
		//}
		pl.StartVertex = ProcessVertex{
			HostID:         conf.MockHostID,
			HostName:       conf.MockHostName,
			ContainerID:    result[1],
			ContainerName:  result[0],
			ProcessVPID:    "1",
			ProcessName:    "fwatchdog",
			ProcessExepath: "unknwon",
		}
	} else {
		pl.StartVertex = SocketVertex{
			HostID:        conf.MockHostID,
			HostName:      conf.MockHostName,
			ContainerID:   conf.OuterContainerID,
			ContainerName: conf.OuterContainerName,
			DstIP:         netLog.IPSrc,
			DstPort:       netLog.PortSrc,
		}
	}
	if containerNameAndID, ok := conf.Config.IPMap[netLog.IPDst]; ok {
		result := strings.Split(containerNameAndID, "$")
		//pl.EndVertex = SocketVertex{
		//	HostID:        conf.MockHostID,
		//	HostName:      conf.MockHostName,
		//	ContainerID:   result[1],
		//	ContainerName: result[0],
		//	DstIP:         netLog.IPDst,
		//	DstPort:       netLog.PortDst,
		//}
		pl.EndVertex = ProcessVertex{
			HostID:         conf.MockHostID,
			HostName:       conf.MockHostName,
			ContainerID:    result[1],
			ContainerName:  result[0],
			ProcessVPID:    "1",
			ProcessName:    "fwatchdog",
			ProcessExepath: "unknwon",
		}
	} else {
		pl.EndVertex = SocketVertex{
			HostID:        conf.MockHostID,
			HostName:      conf.MockHostName,
			ContainerID:   conf.OuterContainerID,
			ContainerName: conf.OuterContainerName,
			DstIP:         netLog.IPDst,
			DstPort:       netLog.PortDst,
		}
	}

	if pl.StartVertex.VertexType() == SOCKETTYPE && pl.EndVertex.VertexType() == SOCKETTYPE {
		pl.Log = ParsedNetLog{
			Method:     netLog.Method,
			PayloadLen: netLog.PayLoadLen,
			SeqNum:     netLog.SeqNum,
			AckNum:     netLog.AckNum,
			Time:       netLog.Time,
			UUID:       netLog.UUID,
		}
		p.pusher.PushParsedLog(pl)
	} else if pl.StartVertex.VertexType() == PROCESSTYPE && pl.EndVertex.VertexType() == SOCKETTYPE {
		pl.Log = ParsedSysdigLog{
			EventCLass: NETWORKV1,
			Relation:   netLog.Method,
			Operation:  netLog.Method,
			Time:       netLog.Time,
			UUID:       netLog.UUID,
		}
		p.pusher.PushParsedLog(pl)
	} else if pl.StartVertex.VertexType() == SOCKETTYPE && pl.EndVertex.VertexType() == PROCESSTYPE {
		pl.Log = ParsedSysdigLog{
			EventCLass: NETWORKV2,
			Relation:   netLog.Method,
			Operation:  netLog.Method,
			Time:       netLog.Time,
			UUID:       netLog.UUID,
		}
		p.pusher.PushParsedLog(pl)
	} else {
		logs.Logger.Error("There are two process vertex in on edge")
	}
	return nil
}
