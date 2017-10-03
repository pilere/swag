package swag

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-openapi/jsonreference"
	"github.com/go-openapi/spec"
)

// Operation describes a single API operation on a path.
//
//  For more information: https://github.com/swaggo/swag#api-operation
type Operation struct {
	HttpMethod string
	Path       string
	spec.Operation

	parser *Parser // TODO: we don't need it
}

// NewOperation creates a new Operation with default properties.
// map[int]Response
func NewOperation() *Operation {
	return &Operation{
		HttpMethod: "get",
		Operation: spec.Operation{
			OperationProps: spec.OperationProps{},
		},
	}
}

// ParseComment parses comment for gived comment string and returns error if error occurs.
func (operation *Operation) ParseComment(comment string) error {
	commentLine := strings.TrimSpace(strings.TrimLeft(comment, "//"))
	if len(commentLine) == 0 {
		return nil
	}

	attribute := strings.Fields(commentLine)[0]
	switch strings.ToLower(attribute) {
	case "@description":
		operation.Description = strings.TrimSpace(commentLine[len(attribute):])
	case "@summary":
		operation.Summary = strings.TrimSpace(commentLine[len(attribute):])
	case "@accept":
		if err := operation.ParseAcceptComment(strings.TrimSpace(commentLine[len(attribute):])); err != nil {
			return err
		}
	case "@produce":
		if err := operation.ParseProduceComment(strings.TrimSpace(commentLine[len(attribute):])); err != nil {
			return err
		}
	case "@param":
		if err := operation.ParseParamComment(strings.TrimSpace(commentLine[len(attribute):])); err != nil {
			return err
		}
	case "@success", "@failure":
		if err := operation.ParseResponseComment(strings.TrimSpace(commentLine[len(attribute):])); err != nil {
			return err
		}
	case "@router":
		if err := operation.ParseRouterComment(strings.TrimSpace(commentLine[len(attribute):])); err != nil {
			return err
		}
	}

	return nil
}

// Parse params return []string of param properties
// @Param	queryText		form	      string	  true		        "The email for login"
// 			[param name]    [paramType] [data type]  [is mandatory?]   [Comment]
// @Param   some_id     path    int     true        "Some ID"
func (operation *Operation) ParseParamComment(commentLine string) error {
	paramString := commentLine

	re := regexp.MustCompile(`([-\w]+)[\s]+([\w]+)[\s]+([\S.]+)[\s]+([\w]+)[\s]+"([^"]+)"`)

	if matches := re.FindStringSubmatch(paramString); len(matches) != 6 {
		return fmt.Errorf("Can not parse param comment \"%s\", skipped.", paramString)
	} else {
		name := matches[1]
		paramType := matches[2]

		schemaType := matches[3]

		requiredText := strings.ToLower(matches[4])
		required := (requiredText == "true" || requiredText == "required")
		description := matches[5]

		var param spec.Parameter

		//five possible parameter types.
		switch paramType {
		case "query", "path":
			param = createParameter(paramType, description, name, schemaType, required)
		case "body":
			param = createParameter(paramType, description, name, "object", required) // TODO: if Parameter types can be objects, but also primitives and arrays

			// TODO: this snippets have to extract out
			refSplit := strings.Split(schemaType, ".")
			if len(refSplit) == 2 {
				//TODO:---------- extract out
				pkgName := refSplit[0]
				typeName := refSplit[1]
				if typeSpec, ok := operation.parser.TypeDefinitions[pkgName][typeName]; ok {
					operation.parser.registerTypes[schemaType] = typeSpec
				} else {
					return fmt.Errorf("Can not find ref type:\"%s\".", schemaType)
				}
				//TODO:----------
				param.Schema.Ref = spec.Ref{
					Ref: jsonreference.MustCreateRef("#/definitions/" + schemaType),
				}
			}

			//case "Header": // TODO: support Header and Form
			//	panic("not supported Header paramType yet.")
			//case "Form":
			//	panic("not supported Form paramType yet.")
		}

		operation.Operation.Parameters = append(operation.Operation.Parameters, param)
	}

	return nil
}
func (operation *Operation) ParseAcceptComment(commentLine string) error {
	accepts := strings.Split(commentLine, ",")
	for _, a := range accepts {
		switch a {
		case "json", "application/json":
			operation.Consumes = append(operation.Consumes, "application/json")
		case "xml", "text/xml":
			operation.Consumes = append(operation.Consumes, "text/xml")
		case "plain", "text/plain":
			operation.Consumes = append(operation.Consumes, "text/plain")
		case "html", "text/html":
			operation.Consumes = append(operation.Consumes, "text/html")
		case "mpfd", "multipart/form-data":
			operation.Consumes = append(operation.Consumes, "multipart/form-data")
		default:
			return fmt.Errorf("%v accept type can't accepted.", a)
		}
	}
	return nil
}

func (operation *Operation) ParseProduceComment(commentLine string) error {
	produces := strings.Split(commentLine, ",")
	for _, a := range produces {
		switch a {
		case "json", "application/json":
			operation.Produces = append(operation.Produces, "application/json")
		case "xml", "text/xml":
			operation.Produces = append(operation.Produces, "text/xml")
		case "plain", "text/plain":
			operation.Produces = append(operation.Produces, "text/plain")
		case "html", "text/html":
			operation.Produces = append(operation.Produces, "text/html")
		case "mpfd", "multipart/form-data":
			operation.Produces = append(operation.Produces, "multipart/form-data")
		default:
			return fmt.Errorf("%v produce type can't accepted.", a)
		}
	}
	return nil
}

func (operation *Operation) ParseRouterComment(commentLine string) error {
	re := regexp.MustCompile(`([\w\.\/\-{}]+)[^\[]+\[([^\]]+)`)
	var matches []string

	if matches = re.FindStringSubmatch(commentLine); len(matches) != 3 {
		return fmt.Errorf("Can not parse router comment \"%s\", skipped.", commentLine)
	}
	path := matches[1]
	httpMethod := matches[2]

	operation.Path = path
	operation.HttpMethod = strings.ToUpper(httpMethod)

	return nil
}

func (operation *Operation) ParseResponseComment(commentLine string) error {
	re := regexp.MustCompile(`([\d]+)[\s]+([\w\{\}]+)[\s]+([\w\-\.\/]+)[^"]*(.*)?`)
	var matches []string

	if matches = re.FindStringSubmatch(commentLine); len(matches) != 5 {
		return fmt.Errorf("Can not parse response comment \"%s\".", commentLine)
	}

	response := spec.Response{}

	code, _ := strconv.Atoi(matches[1])

	response.Description = strings.Trim(matches[4], "\"")

	schemaType := strings.Trim(matches[2], "{}")
	refType := matches[3]

	if operation.parser != nil { // checking refType has existing in 'TypeDefinitions'
		refSplit := strings.Split(refType, ".")
		if len(refSplit) == 2 {
			pkgName := refSplit[0]
			typeName := refSplit[1]
			if typeSpec, ok := operation.parser.TypeDefinitions[pkgName][typeName]; ok {
				operation.parser.registerTypes[refType] = typeSpec
			} else {
				operation.parser.registerTypes[refType] = nil
				//return fmt.Errorf("Can not find ref type:\"%s\".", refType)
			}

		}
	}

	// so we have to know all type in app
	//TODO: we might omitted schema.type if schemaType equals 'object'
	response.Schema = &spec.Schema{SchemaProps: spec.SchemaProps{Type: []string{schemaType}}}

	if schemaType == "object" {
		response.Schema.Ref = spec.Ref{
			Ref: jsonreference.MustCreateRef("#/definitions/" + refType),
		}
	}

	if schemaType == "array" {
		response.Schema.Items = &spec.SchemaOrArray{
			Schema: &spec.Schema{
				SchemaProps: spec.SchemaProps{
					Ref: spec.Ref{Ref: jsonreference.MustCreateRef("#/definitions/" + refType)},
				},
			},
		}

	}

	if operation.Responses == nil {
		operation.Responses = &spec.Responses{
			ResponsesProps: spec.ResponsesProps{
				StatusCodeResponses: make(map[int]spec.Response),
			},
		}
	}

	operation.Responses.StatusCodeResponses[code] = response

	return nil
}

// createParamter returns swagger spec.Parameter for gived  paramType, description, paramName, schemaType, required
func createParameter(paramType, description, paramName, schemaType string, required bool) spec.Parameter {
	// //five possible parameter types. 	query, path, body, header, form
	paramProps := spec.ParamProps{
		Name:        paramName,
		Description: description,
		Required:    required,
		In:          paramType,
	}
	if paramType == "body" {
		paramProps.Schema = &spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{schemaType},
			},
		}
		parameter := spec.Parameter{
			ParamProps: paramProps,
		}

		return parameter
	} else {
		parameter := spec.Parameter{
			ParamProps: paramProps,
			SimpleSchema: spec.SimpleSchema{
				Type: schemaType,
			},
		}
		return parameter

	}
}
