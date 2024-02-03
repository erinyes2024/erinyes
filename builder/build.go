package builder

import (
	"erinyes/helper"
	"erinyes/logs"
	"erinyes/models"
	"erinyes/parser"
	"github.com/awalterschulze/gographviz"
	"os"
	"strings"
)

func createDir(dirName string) {
	if _, err := os.Stat(dirName); os.IsNotExist(err) {
		err := os.MkdirAll(dirName, 0755)
		if err != nil {
			logs.Logger.Errorf(err.Error())
		}
	}
}

func GenerateDotGraph(uuid string) *gographviz.Graph {
	graphAst, _ := gographviz.Parse([]byte(`digraph G{}`))
	graph := gographviz.NewGraph()
	gographviz.Analyse(graphAst, graph)
	db := models.GetMysqlDB()

	pageSize := 100
	pageNumber := 1
	for {
		var events []models.Event
		db.Order("id").Limit(pageSize).Offset((pageNumber - 1) * pageSize).Find(&events)
		if len(events) == 0 {
			break
		}
		for _, event := range events {
			var start models.DotVertex
			var end models.DotVertex
			switch event.EventClass {
			case parser.PROCESS:
				var startProcess models.Process
				var endProcess models.Process
				result1 := startProcess.FindByID(db, event.SrcID)
				result2 := endProcess.FindByID(db, event.DstID)
				if !(result1 && result2) {
					continue
				}
				start = startProcess
				end = endProcess
			case parser.FILEV1:
				var startProcess models.Process
				var endFile models.File
				result1 := startProcess.FindByID(db, event.SrcID)
				result2 := endFile.FindByID(db, event.DstID)
				if !(result1 && result2) {
					continue
				}
				start = startProcess
				end = endFile
			case parser.FILEV2:
				var startFile models.File
				var endProcess models.Process
				result1 := startFile.FindByID(db, event.SrcID)
				result2 := endProcess.FindByID(db, event.DstID)
				if !(result1 && result2) {
					continue
				}
				start = startFile
				end = endProcess
			case parser.NETWORKV1:
				var startProcess models.Process
				var endSocket models.Socket
				result1 := startProcess.FindByID(db, event.SrcID)
				result2 := endSocket.FindByID(db, event.DstID)
				if !(result1 && result2) {
					continue
				}
				start = startProcess
				end = endSocket
			case parser.NETWORKV2:
				var startSocket models.Socket
				var endProcess models.Process
				result1 := startSocket.FindByID(db, event.SrcID)
				result2 := endProcess.FindByID(db, event.DstID)
				if !(result1 && result2) {
					continue
				}
				start = startSocket
				end = endProcess
			default:
				logs.Logger.Warnf("Unknown event class: %s in event tables", event.EventClass)
			}
			GenerateEdge(start, end, event, graph, uuid)
		}
		pageNumber++
	}
	pageNumber = 1
	for {
		var nets []models.Net
		db.Order("id").Limit(pageSize).Offset((pageNumber - 1) * pageSize).Find(&nets)
		if len(nets) == 0 {
			break
		}
		for _, net := range nets {
			var startSocket models.Socket
			var endSocket models.Socket
			result1 := startSocket.FindByID(db, net.SrcID)
			result2 := endSocket.FindByID(db, net.DstID)
			if !(result1 && result2) {
				continue
			}
			GenerateEdge(startSocket, endSocket, net, graph, uuid)
		}
		pageNumber++
	}
	return graph
}

func GenerateDot(fileName string, uuid string) {
	createDir("graphs/")
	dotName := "graphs/" + fileName + ".dot"
	graph := GenerateDotGraph(uuid)
	fo, err := os.OpenFile(dotName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		logs.Logger.WithError(err).Fatalf("Open file %s failed", dotName)
	}
	defer fo.Close()
	fo.WriteString(graph.String())
}

func GenerateEdge(startVertex models.DotVertex, endVertex models.DotVertex, edge models.DotEdge, graph *gographviz.Graph, uuid string) {
	if uuid != "" {
		if !edge.HasEdgeUUID() {
			return
		}
		uuidStr := edge.GetUUID()
		uuids := strings.Split(uuidStr, ",")
		if !helper.SliceContainsTarget(uuids, uuid) {
			return
		}
	}

	edgeM := make(map[string]string)
	edgeM["label"] = edge.EdgeName()
	startVertexM := make(map[string]string)
	startVertexM["shape"] = startVertex.VertexShape()
	endVertexM := make(map[string]string)
	endVertexM["shape"] = endVertex.VertexShape()

	startSubG := startVertex.VertexClusterID()
	startSubGM := make(map[string]string)
	startSubGM["label"] = startVertex.VertexClusterID()
	graph.AddSubGraph("G", startSubG, startSubGM)
	graph.AddNode(startSubG, startVertex.VertexName(), startVertexM)

	endSubG := endVertex.VertexClusterID()
	endSubGM := make(map[string]string)
	endSubGM["label"] = endVertex.VertexClusterID()
	graph.AddSubGraph("G", endSubG, endSubGM)
	graph.AddNode(endSubG, endVertex.VertexName(), endVertexM)

	graph.AddEdge(startVertex.VertexName(), endVertex.VertexName(), true, edgeM)
}

func GenerateVertex(vertex models.DotVertex, graph *gographviz.Graph) {
	vertexM := make(map[string]string)
	vertexM["shape"] = vertex.VertexShape()

	SubG := vertex.VertexClusterID()
	SubGM := make(map[string]string)
	SubGM["label"] = vertex.VertexClusterID()
	graph.AddSubGraph("G", SubG, SubGM)
	graph.AddNode(SubG, vertex.VertexName(), vertexM)
}
