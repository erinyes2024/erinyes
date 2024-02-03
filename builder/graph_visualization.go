package builder

import (
	"bytes"
	"erinyes/logs"
	"fmt"
	"github.com/awalterschulze/gographviz"
	"gonum.org/v1/gonum/graph/multi"
	"io/ioutil"
	"os/exec"
)

var entityType2Shape = map[NodeType]string{
	Process: "rect", File: "ellipse", Socket: "diamond",
}

func callSystem(s string, args ...string) error {
	cmd := exec.Command(s, args...)
	var out bytes.Buffer

	cmd.Stdout = &out
	err := cmd.Run()
	fmt.Printf("%s", out.String())
	return err
}

func Visualize(g *multi.WeightedDirectedGraph, filename string) error {
	graphAst, _ := gographviz.ParseString(`digraph G{}`)
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return err
	}
	nodes := g.Nodes()
	logs.Logger.Infof("Nodes: %d", nodes.Len())
	for nodes.Next() {
		N := nodes.Node()
		n := N.(GraphNode)
		GenerateVertex(n, graph)
	}

	edges := g.Edges()
	count := 0
	for edges.Next() {
		e := edges.Edge()
		lines := g.WeightedLines(e.From().ID(), e.To().ID())
		for lines.Next() {
			count++
			L := lines.WeightedLine()
			l := L.(GraphLine)
			From := l.From()
			from := From.(GraphNode)
			To := l.To()
			to := To.(GraphNode)
			//if l.TimeStamp != 0 {
			//	if err := graph.AddEdge(from.VertexName(), to.VertexName(), true, map[string]string{"label": fmt.Sprintf("%s_%ds", l.Relation, l.TimeStamp)}); err != nil {
			//		logs.Logger.Warnf("failed to add edge to the graphviz, edge = [from: %s, to: %s]", from.VertexName(), to.VertexName())
			//	}
			//} else {
			//	if err := graph.AddEdge(from.VertexName(), to.VertexName(), true, map[string]string{"label": l.Relation}); err != nil {
			//		logs.Logger.Warnf("failed to add edge to the graphviz, edge = [from: %s, to: %s]", from.VertexName(), to.VertexName())
			//	}
			//}
			if err := graph.AddEdge(from.VertexName(), to.VertexName(), true, map[string]string{"label": l.Relation}); err != nil {
				logs.Logger.Warnf("failed to add edge to the graphviz, edge = [from: %s, to: %s]", from.VertexName(), to.VertexName())
			}
		}
	}
	logs.Logger.Infof("Edges: %d", count)
	//fmt.Println(graph.String())
	if err := ioutil.WriteFile("graphs/"+filename+".dot", []byte(graph.String()), 0666); err != nil {
		return err
	}
	return callSystem("dot", "-T", "svg", "graphs/"+filename+".dot", "-o", "graphs/"+filename+".svg")
	//return nil
}
