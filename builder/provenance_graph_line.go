package builder

import "gonum.org/v1/gonum/graph"

type GraphLine struct {
	F, T      graph.Node
	W         float64
	Relation  string
	TimeStamp int64
	UID       int64
}

// From To ReversedLine ID Weight implements the WeightedLine interface
func (l GraphLine) From() graph.Node { return l.F }

func (l GraphLine) To() graph.Node { return l.T }

func (l GraphLine) ReversedLine() graph.Line { l.F, l.T = l.T, l.F; return l }

func (l GraphLine) ID() int64 { return l.UID }

// Weight returns the weight of the edge.
func (l GraphLine) Weight() float64 { return l.W }
