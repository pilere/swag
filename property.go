package swag

import (
	"go/ast"
	"log"
)

// getPropertyName returns the string value for the given field if it exists, otherwise it panics.
func getPropertyName(field *ast.Field) string {
	var name string
	if _, ok := field.Type.(*ast.SelectorExpr); ok {
		panic("not supported 'astSelectorExpr' yet.")
	} else if astTypeIdent, ok := field.Type.(*ast.Ident); ok {
		name = astTypeIdent.Name
	} else if _, ok := field.Type.(*ast.StarExpr); ok {
		panic("not supported astStarExpr yet.")
	} else if _, ok := field.Type.(*ast.MapType); ok { // if map
		//TODO: support map
		return "object"
	} else if _, ok := field.Type.(*ast.ArrayType); ok { // if array
		return "array"
	} else if _, ok := field.Type.(*ast.StructType); ok { // if struct
		//TODO: support nested struct
		return "object"
	} else {
		log.Fatalf("Something goes wrong: %#v", field.Type)
	}

	return name
}
