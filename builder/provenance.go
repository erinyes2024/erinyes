package builder

import (
	"erinyes/helper"
	"erinyes/logs"
	"erinyes/models"
	"erinyes/parser"
	"fmt"
	"gonum.org/v1/gonum/graph/multi"
	"strconv"
	"strings"
	"time"
)

const (
	ProcessTable = "process"
	FileTable    = "file"
	SocketTable  = "socket"
)

type RecordLoc struct {
	Key   int    // primary key
	Table string // identify which table
}

func Provenance(hostID string, containerID string, processID string, processName string, timestamp *int64, depth *int, timeLimit bool, uuid string) *multi.WeightedDirectedGraph {
	// get root process
	mysqlDB := models.GetMysqlDB()
	var process models.Process
	if err := mysqlDB.First(&process, models.Process{HostID: hostID, ContainerID: containerID, ProcessVPID: processID, ProcessName: processName}).Error; err != nil {
		logs.Logger.WithError(err).Errorf("failed to build subgraph for process[host: %s, container: %s,process_vid: %s, process_name: %s]", hostID, containerID, processID, processName)
		return nil
	}
	g := multi.NewWeightedDirectedGraph()
	addedEventLine := make(map[int]bool)
	addedNetLine := make(map[int]bool)
	addedNode := make(map[RecordLoc]int64)
	id := AddNewGraphNode(g, Process,
		ProcessInfo{
			Path:          process.ProcessExepath,
			Name:          process.ProcessName,
			Pid:           process.ProcessVPID,
			ContainerID:   process.ContainerID,
			ContainerName: process.ContainerName,
			HostID:        process.HostID,
			HostName:      process.HostName})

	root := RecordLoc{Key: process.ID, Table: ProcessTable}
	addedNode[root] = id
	//logs.Logger.Infof("Start forward tracing，processID: %s, processName: %s, table: %s, primary key: %d", processID, process.ProcessName, ProcessTable, process.ID)
	//logs.Logger.Infof("Start forward tracing...")
	//startTime := time.Now()
	node2time := make(map[RecordLoc]int64)
	//if timestamp != nil {
	//	node2time[root] = *timestamp
	//}
	//BFS(g, root, addedEventLine, addedNetLine, addedNode, node2time, false, depth, timeLimit, uuid)
	//logs.Logger.Infof("It takes about %v seconds to forward BFS", time.Since(startTime).Seconds())
	middleTime := time.Now()
	logs.Logger.Infof("Start backward tracing...")
	node2time = make(map[RecordLoc]int64)
	if timestamp != nil {
		node2time[root] = *timestamp
	}
	BFS(g, root, addedEventLine, addedNetLine, addedNode, node2time, true, depth, timeLimit, uuid)
	logs.Logger.Infof("It takes about %v seconds to backward BFS", time.Since(middleTime).Seconds())
	logs.Logger.Infof("subgraph generated success!")
	//logs.Logger.Infof("It takes about %v seconds to build Provenance Graph", time.Since(startTime).Seconds())
	return g
}

func AddNewGraphNode(g *multi.WeightedDirectedGraph, nodeType NodeType, nodeInfo NodeInfo) int64 {
	temp := g.NewNode()
	graphNode := GraphNode{
		id:       temp.ID(),
		nodeType: nodeType,
		nodeInfo: nodeInfo,
	}
	g.AddNode(graphNode)
	return temp.ID()
}

func AddNewGraphEdge(g *multi.WeightedDirectedGraph, from int64, to int64, relation string, timestamp int64, weight float64) {
	weightedLine := g.NewWeightedLine(GraphNode{id: from}, GraphNode{id: to}, weight)
	graphLine := GraphLine{
		F:         g.Node(from),
		T:         g.Node(to),
		W:         weightedLine.Weight(),
		Relation:  relation,
		TimeStamp: timestamp,
		UID:       weightedLine.ID(),
	}
	g.SetWeightedLine(graphLine)
}

func BFS(g *multi.WeightedDirectedGraph, root RecordLoc, addedEventLine map[int]bool, addedNetLine map[int]bool, addedNode map[RecordLoc]int64, node2time map[RecordLoc]int64, reverse bool, maxLevel *int, timeLimit bool, uuid string) {
	visitedNode := map[RecordLoc]bool{root: true}
	var queue []RecordLoc
	currLevel := 0
	queue = append(queue, root)
	for {
		if len(queue) == 0 { // isEmpty(queue)
			break
		}
		if maxLevel != nil {
			if currLevel >= *maxLevel {
				break
			}
		}
		size := len(queue)
		for i := 0; i < size; i++ {
			cur := queue[0]
			events := FetchEvents(cur.Key, cur.Table, reverse)
			for _, e := range events {
				if uuid != "" {
					uuids := strings.Split(e.UUID, ",")
					if !helper.SliceContainsTarget(uuids, uuid) {
						continue
					}
				}
				if timeLimit {
					if reverse {
						if node2time[cur] != 0 && node2time[cur] < e.Time {
							continue
						}
					} else {
						if node2time[cur] != 0 && node2time[cur] > e.Time {
							continue
						}
					}
				}
				var tempRecord RecordLoc
				tableName, err := GetTableName(e.EventClass, reverse)
				if err != nil {
					logs.Logger.WithError(err).Errorf("failed to get table name")
					continue
				}
				if reverse {
					tempRecord = RecordLoc{Key: e.SrcID, Table: tableName}
				} else {
					tempRecord = RecordLoc{Key: e.DstID, Table: tableName}
				}
				if _, ok := visitedNode[tempRecord]; !ok {
					if _, ok := addedNode[tempRecord]; ok {
						visitedNode[tempRecord] = true
						queue = append(queue, tempRecord)
					} else {
						if nodeType, nodeInfo, err := GetEntityNode(tempRecord); err != nil {
							logs.Logger.WithError(err).Errorf("failed to fetch entity")
							continue
						} else {
							id := AddNewGraphNode(g, nodeType, nodeInfo)
							visitedNode[tempRecord] = true
							addedNode[tempRecord] = id
							queue = append(queue, tempRecord)
						}
					}
					if timeLimit {
						node2time[tempRecord] = e.Time
					}
				} else {
					if timeLimit {
						if reverse {
							if node2time[tempRecord] != 0 && node2time[tempRecord] < e.Time {
								node2time[tempRecord] = e.Time
							}
						} else {
							if node2time[tempRecord] != 0 && node2time[tempRecord] > e.Time {
								node2time[tempRecord] = e.Time
							}
						}
					}
				}
				if _, ok := addedEventLine[e.ID]; !ok {
					addedEventLine[e.ID] = true
					var (
						fromID int64
						toID   int64
						tempA  int64
						tempB  int64
					)
					if tempA, ok = addedNode[tempRecord]; !ok {
						panic("No such Node in the graph")
					}
					if tempB, ok = addedNode[cur]; !ok {
						panic("No such Node in the graph")
					}
					if reverse {
						fromID = tempA
						toID = tempB
					} else {
						fromID = tempB
						toID = tempA
					}
					AddNewGraphEdge(g, fromID, toID, e.Relation, e.Time, 0)
				}
			}
			nets := FetchNets(cur.Key, cur.Table, reverse)
			for _, n := range nets {
				if uuid != "" {
					uuids := strings.Split(n.UUID, ",")
					if !helper.SliceContainsTarget(uuids, uuid) {
						continue
					}
				}
				if timeLimit {
					if reverse {
						if node2time[cur] != 0 && node2time[cur] < n.Time {
							continue
						}
					} else {
						if node2time[cur] != 0 && node2time[cur] > n.Time {
							continue
						}
					}
				}
				var tempRecord RecordLoc
				if reverse {
					tempRecord = RecordLoc{Key: n.SrcID, Table: SocketTable}
				} else {
					tempRecord = RecordLoc{Key: n.DstID, Table: SocketTable}
				}
				if _, ok := visitedNode[tempRecord]; !ok {
					if _, ok := addedNode[tempRecord]; ok {
						visitedNode[tempRecord] = true
						queue = append(queue, tempRecord)
					} else {
						if nodeType, nodeInfo, err := GetEntityNode(tempRecord); err != nil {
							logs.Logger.WithError(err).Errorf("failed to fetch entity")
							continue
						} else {
							id := AddNewGraphNode(g, nodeType, nodeInfo)
							visitedNode[tempRecord] = true
							addedNode[tempRecord] = id
							queue = append(queue, tempRecord)
						}
					}
					if timeLimit {
						node2time[tempRecord] = n.Time
					}
				} else {
					if timeLimit {
						if reverse {
							if node2time[tempRecord] != 0 && node2time[tempRecord] < n.Time {
								node2time[tempRecord] = n.Time
							}
						} else {
							if node2time[tempRecord] != 0 && node2time[tempRecord] > n.Time {
								node2time[tempRecord] = n.Time
							}
						}
					}
				}
				if _, ok := addedNetLine[n.ID]; !ok {
					addedNetLine[n.ID] = true
					var (
						fromID int64
						toID   int64
						tempA  int64
						tempB  int64
					)
					if tempA, ok = addedNode[tempRecord]; !ok {
						panic("No such Node in the graph")
					}
					if tempB, ok = addedNode[cur]; !ok {
						panic("No such Node in the graph")
					}
					if reverse {
						fromID = tempA
						toID = tempB
					} else {
						fromID = tempB
						toID = tempA
					}
					AddNewGraphEdge(g, fromID, toID, n.Method, n.Time, 0)
				}
			}
			queue = queue[1:]
		}
		currLevel++
	}
}

func FetchEvents(key int, table string, reverse bool) []models.Event {
	mysqlDB := models.GetMysqlDB()
	sqlStr := "event_class = ?"
	switch table {
	case ProcessTable:
		if reverse { // 1. process -> process 2. file -> process 3. socket -> process
			mysqlDB = mysqlDB.Where(sqlStr+" or "+sqlStr+" or "+sqlStr,
				parser.PROCESS, parser.FILEV2, parser.NETWORKV2)
		} else { // 1. process -> process 2. process -> file 3. process -> socket
			mysqlDB = mysqlDB.Where(sqlStr+" or "+sqlStr+" or "+sqlStr,
				parser.PROCESS, parser.FILEV1, parser.NETWORKV1)
		}
	case FileTable:
		if reverse { // 1. process -> file
			mysqlDB = mysqlDB.Where(sqlStr, parser.FILEV1)
		} else { // 1. file -> process
			mysqlDB = mysqlDB.Where(sqlStr, parser.FILEV2)
		}
	case SocketTable:
		if reverse { // 1. process -> socket
			mysqlDB = mysqlDB.Where(sqlStr, parser.NETWORKV1)
		} else { // 1. socket -> process
			mysqlDB = mysqlDB.Where(sqlStr, parser.NETWORKV2)
		}
	default:
		logs.Logger.Errorf("failed to parse table %s, fetch events failed", table)
		return nil
	}
	if reverse {
		mysqlDB = mysqlDB.Where("dst_id = ?", strconv.Itoa(key))
	} else {
		mysqlDB = mysqlDB.Where("src_id = ?", strconv.Itoa(key))
	}
	var events []models.Event
	if err := mysqlDB.Find(&events).Error; err != nil {
		logs.Logger.WithError(err).Errorf("failed to fetch events(edges) from db")
		return nil
	}
	return events
}

func FetchNets(key int, table string, reverse bool) []models.Net {
	if table != SocketTable {
		return nil
	}
	var nets []models.Net
	mysqlDB := models.GetMysqlDB()
	if reverse {
		mysqlDB = mysqlDB.Where("dst_id = ?", strconv.Itoa(key))
	} else {
		mysqlDB = mysqlDB.Where("src_id = ?", strconv.Itoa(key))
	}
	if err := mysqlDB.Find(&nets).Error; err != nil {
		logs.Logger.WithError(err).Errorf("failed to fetch nets(edges) from db")
		return nil
	}
	return nets
}

func GetTableName(eventClass string, reverse bool) (string, error) {
	switch eventClass {
	case parser.PROCESS: // process -> process
		return ProcessTable, nil
	case parser.FILEV1: // process -> file
		return helper.MyStringIf(reverse, ProcessTable, FileTable), nil
	case parser.FILEV2: // file -> process
		return helper.MyStringIf(reverse, FileTable, ProcessTable), nil
	case parser.NETWORKV1: // process -> socket
		return helper.MyStringIf(reverse, ProcessTable, SocketTable), nil
	case parser.NETWORKV2: // socket -> process
		return helper.MyStringIf(reverse, SocketTable, ProcessTable), nil
	}
	return "", fmt.Errorf("failed to calculate the table by eventClass: %s", eventClass)
}

func GetEntityNode(r RecordLoc) (NodeType, NodeInfo, error) {
	mysqlDB := models.GetMysqlDB()
	switch r.Table {
	case ProcessTable:
		var process models.Process
		if err := mysqlDB.First(&process, r.Key).Error; err != nil {
			return -1, nil, fmt.Errorf("failed to get process entity node from db, err: %w", err)
		}
		return Process, ProcessInfo{
			Path:          process.ProcessExepath,
			Name:          process.ProcessName,
			Pid:           process.ProcessVPID,
			HostName:      process.HostName,
			HostID:        process.HostID,
			ContainerName: process.ContainerName,
			ContainerID:   process.ContainerID}, nil
	case FileTable:
		var file models.File
		if err := mysqlDB.First(&file, r.Key).Error; err != nil {
			return -1, nil, fmt.Errorf("failed to get file entity node from db, err: %w", err)
		}
		return File, FileInfo{
			HostName:      file.HostName,
			HostID:        file.HostID,
			ContainerID:   file.ContainerID,
			ContainerName: file.ContainerName,
			Path:          file.FilePath}, nil
	case SocketTable:
		var socket models.Socket
		if err := mysqlDB.First(&socket, r.Key).Error; err != nil {
			return -1, nil, fmt.Errorf("failed to get socket entity node from db, err: %w", err)
		}
		return Socket, SocketInfo{
			DstIP:         socket.DstIP,
			DstPort:       socket.DstPort,
			ContainerName: socket.ContainerName,
			ContainerID:   socket.ContainerID,
			HostID:        socket.HostID,
			HostName:      socket.HostName}, nil
	}
	return -1, nil, fmt.Errorf("unknown record %s, can't find the entity from db", r.Table)
}
