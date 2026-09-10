package projectv8instance

import (
	"testing"

	"seme.local/reference/wire"
)

func TestMatchPackagesAcceptsAuthenticatedDisconnectedForest(t *testing.T) {
	snapshot, complete, graph := id("aa01"), id("aa02"), id("aa03")
	root, library := id("aa10"), id("aa11")
	rootDetail, libraryDetail := id("aa20"), id("aa21")
	e := wire.Envelope{Entities: map[wire.ID]wire.Entity{
		snapshot: {ID: snapshot, Schema: id("e011"), Version: 1, Fields: map[wire.ID]wire.Value{
			id("e112"): {Tag: 7, List: []wire.Value{ref(root), ref(library)}},
			id("e113"): ref(root),
		}},
		complete: {ID: complete, Schema: id("b029"), Version: 1, Fields: map[wire.ID]wire.Value{id("b290"): ref(graph)}},
		graph: {ID: graph, Schema: id("b020"), Version: 1, Fields: map[wire.ID]wire.Value{
			id("b200"): {Tag: 7, List: []wire.Value{ref(rootDetail), ref(libraryDetail)}},
		}},
		rootDetail:    {ID: rootDetail, Schema: id("b021"), Version: 1, Fields: map[wire.ID]wire.Value{id("b210"): ref(root)}},
		libraryDetail: {ID: libraryDetail, Schema: id("b021"), Version: 1, Fields: map[wire.ID]wire.Value{id("b210"): ref(library)}},
		root:          {ID: root, Schema: id("b010"), Version: 1, Fields: map[wire.ID]wire.Value{}},
		library:       {ID: library, Schema: id("b010"), Version: 1, Fields: map[wire.ID]wire.Value{}},
	}}
	if err := matchPackages(e, snapshot, complete); err != nil {
		t.Fatal(err)
	}

	bad := e
	bad.Entities = make(map[wire.ID]wire.Entity, len(e.Entities))
	for k, v := range e.Entities {
		bad.Entities[k] = v
	}
	q := bad.Entities[snapshot]
	q.Fields = map[wire.ID]wire.Value{id("e112"): {Tag: 7, List: []wire.Value{ref(root)}}, id("e113"): ref(root)}
	bad.Entities[snapshot] = q
	if err := matchPackages(bad, snapshot, complete); err == nil {
		t.Fatal("accepted detail for unlisted package")
	}
}
