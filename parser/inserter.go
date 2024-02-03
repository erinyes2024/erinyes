package parser

import (
	"erinyes/logs"
	"erinyes/models"
	"fmt"
	"gorm.io/gorm"
)

type Inserter struct {
	ParsedLogCh *chan ParsedLog
}

func (pi *Inserter) InsertOrQueryVertex(db *gorm.DB, vertexI ParsedVertex, count *int) (int, error) {
	if vertexI.VertexType() == PROCESSTYPE {
		vertex := vertexI.(ProcessVertex)
		processPO := models.Process{
			HostID:         vertex.HostID,
			HostName:       vertex.HostName,
			ContainerID:    vertex.ContainerID,
			ContainerName:  vertex.ContainerName,
			ProcessVPID:    vertex.ProcessVPID,
			ProcessName:    vertex.ProcessName,
			ProcessExepath: vertex.ProcessExepath,
		}
		result := db.Create(&processPO)
		if result.Error != nil {
			r := db.Where("container_id = ? AND host_id = ? AND process_vpid = ? AND process_name = ?", vertex.ContainerID, vertex.HostID, vertex.ProcessVPID, vertex.ProcessName).First(&processPO)
			if r.Error != nil {
				return 0, r.Error
			}
		} else {
			*count++
		}
		return processPO.ID, nil
	} else if vertexI.VertexType() == FILETYPE {
		vertex := vertexI.(FileVertex)
		filePO := models.File{
			HostID:        vertex.HostID,
			HostName:      vertex.HostName,
			ContainerID:   vertex.ContainerID,
			ContainerName: vertex.ContainerName,
			FilePath:      vertex.FilePath,
		}
		//logs.Logger.Infof("fileP0: %s", filePO.FilePath)
		result := db.Create(&filePO)
		if result.Error != nil {
			r := db.Where("container_id = ? AND host_id = ? AND file_path = ?", vertex.ContainerID, vertex.HostID, vertex.FilePath).First(&filePO)
			if r.Error != nil {
				return 0, r.Error
			}
		} else {
			*count++
		}
		return filePO.ID, nil
	} else if vertexI.VertexType() == SOCKETTYPE {
		vertex := vertexI.(SocketVertex)
		socketPO := models.Socket{
			HostID:        vertex.HostID,
			HostName:      vertex.HostName,
			ContainerID:   vertex.ContainerID,
			ContainerName: vertex.ContainerName,
			DstIP:         vertex.DstIP,
			DstPort:       vertex.DstPort,
		}
		socketPO.RelateHostAndCin()
		socketPO.UnionGateway()
		result := db.Create(&socketPO)
		if result.Error != nil {
			r := db.Where("container_id = ? AND host_id = ? AND dst_ip = ? AND dst_port = ?", socketPO.ContainerID, socketPO.HostID, socketPO.DstIP, socketPO.DstPort).First(&socketPO)
			if r.Error != nil {
				return 0, r.Error
			}
		} else {
			*count++
		}
		return socketPO.ID, nil
	}
	return 0, fmt.Errorf("unknown vertex type: %s", vertexI.VertexType())
}

func (pi *Inserter) InsertEdge(db *gorm.DB, edgeI ParsedEdge, startID int, endID int, count *int, repeat bool) {
	if edgeI.LogType() == SYSDIGTYPE {
		sysdigEdge := edgeI.(ParsedSysdigLog)
		if !repeat {
			models.Mu.Lock()
			defer models.Mu.Unlock()
			var existSysdigPO models.Event
			result := db.Where("src_id = ? AND dst_id = ? AND event_class = ? AND operation = ? AND uuid = ?", startID, endID, sysdigEdge.EventCLass, sysdigEdge.Operation, sysdigEdge.UUID).First(&existSysdigPO)
			if result.Error == nil { // 存在该记录 不需要插入
				return
			}
		}
		sysdigPO := models.Event{
			SrcID:      startID,
			DstID:      endID,
			EventClass: sysdigEdge.EventCLass,
			Relation:   sysdigEdge.Relation,
			Operation:  sysdigEdge.Operation,
			Time:       sysdigEdge.Time,
			UUID:       sysdigEdge.UUID,
		}
		result := db.Create(&sysdigPO)
		if result.Error != nil {
			logs.Logger.WithError(result.Error).Errorf("insert failed %w", sysdigPO)
		}
		*count++
		return
	} else if edgeI.LogType() == NETTYPE {
		netEdge := edgeI.(ParsedNetLog)
		if !repeat {
			models.Mu.Lock()
			defer models.Mu.Unlock()
			var existNetPO models.Net
			result := db.Where("src_id = ? AND dst_id = ? AND method = ? AND uuid = ?", startID, endID, netEdge.Method, netEdge.UUID).First(&existNetPO)
			if result.Error == nil {
				return
			}
		}
		netPO := models.Net{
			SrcID:      startID,
			DstID:      endID,
			Method:     netEdge.Method,
			Payload:    netEdge.Payload,
			PayloadLen: netEdge.PayloadLen,
			SeqNum:     netEdge.SeqNum,
			AckNum:     netEdge.AckNum,
			Time:       netEdge.Time,
			UUID:       netEdge.UUID,
		}
		result := db.Create(&netPO)
		if result.Error != nil {
			logs.Logger.WithError(result.Error).Errorf("insert failed %w", netPO)
		}
		*count++
		return
	}
	logs.Logger.Errorf("Unknown edge type")

}

func (pi *Inserter) Insert(goroutine int, repeat bool) {
	logs.Logger.Infof("Start inserter routine %d...", goroutine)
	db := models.GetMysqlDB()
	cnt := 0
	edgeCnt := 0
	vertexCnt := 0
	for parsedLog := range *pi.ParsedLogCh {
		cnt += 1
		if cnt%1000 == 0 {
			logs.Logger.Infof("[Inserter goroutine %d] Now solved %d logs", goroutine, cnt)
		}
		EdgeI := parsedLog.Log

		StartVertexI := parsedLog.StartVertex
		EndVertexI := parsedLog.EndVertex
		startID, err := pi.InsertOrQueryVertex(db, StartVertexI, &vertexCnt)
		if err != nil {
			logs.Logger.WithError(err).Errorf("[Inserter goroutine %d] Insert or query vertex failed", goroutine)
			continue
		}
		endID, err := pi.InsertOrQueryVertex(db, EndVertexI, &vertexCnt)
		if err != nil {
			logs.Logger.WithError(err).Errorf("[Inserter goroutine %d] Insert or query vertex failed", goroutine)
			continue
		}
		pi.InsertEdge(db, EdgeI, startID, endID, &edgeCnt, repeat)
	}
	logs.Logger.Infof("Complete inserter goroutine %d, insert %d edges and %d vertexs", goroutine, edgeCnt, vertexCnt)
}
