package tmap

import (
	"image/color"

	"github.com/nikolaydubina/treemap"
	"github.com/nikolaydubina/treemap/parser"
	"github.com/nikolaydubina/treemap/render"
)

//
// External Inferface
//

func Tmap(in string) string {
	var (
		w             float64 = 600
		h             float64 = 400
		marginBox     float64 = 0
		paddingBox    float64 = 0
		padding       float64 = 0
		keepLongPaths bool    = true
	)
	parser := parser.CSVTreeParser{}
	tree, err := parser.ParseString(in)
	if err != nil || tree == nil {
		return ""
	}
	treemap.SetNamesFromPaths(tree)
	if !keepLongPaths {
		treemap.CollapseLongPaths(tree)
	}
	sizeImputer := treemap.SumSizeImputer{EmptyLeafSize: 1}
	sizeImputer.ImputeSize(*tree)
	tree.NormalizeHeat()
	var colorer render.Colorer
	treeHueColorer := render.TreeHueColorer{
		Offset: 0,
		Hues:   map[string]float64{},
		C:      0.5,
		L:      0.5,
		DeltaH: 10,
		DeltaC: 0.3,
		DeltaL: 0.1,
	}
	var borderColor color.Color
	borderColor = color.White
	colorer = treeHueColorer
	uiBuilder := render.UITreeMapBuilder{
		Colorer:     colorer,
		BorderColor: borderColor,
	}
	spec := uiBuilder.NewUITreeMap(*tree, w, h, marginBox, paddingBox, padding)
	r := render.SVGRenderer{}
	return string(r.Render(spec, w, h))
}
