package types

type NodeType int8

const (
	UnspecifiedNodeType NodeType = iota
	ValidatorNode
	RpcNode
	SnapshotNode
)

var nodeTypeNameToType = map[string]NodeType{
	"validator": ValidatorNode,
	"rpc":       RpcNode,
	"snapshot":  SnapshotNode,
}

func (t NodeType) String() string {
	for name, nodeType := range nodeTypeNameToType {
		if nodeType == t {
			return name
		}
	}

	return "unspecified"
}

func NodeTypeFromString(name string) NodeType {
	if nodeType, found := nodeTypeNameToType[name]; found {
		return nodeType
	}

	return UnspecifiedNodeType
}

func AllNodeTypeNames() []string {
	var names []string
	for name := range nodeTypeNameToType {
		names = append(names, name)
	}
	return names
}
