package models

type DotVertex interface {
	VertexClusterID() string
	VertexName() string
	VertexShape() string
}

type DotEdge interface {
	EdgeName() string
	HasEdgeUUID() bool
	GetUUID() string
}
