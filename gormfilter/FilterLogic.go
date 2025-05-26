package gormfilter

import (
	"gorm.io/gorm"
	"reflect"
)

type condStruct struct {
	sql  string
	vals []interface{}
}

// FindUsersWithFilter dynamically applies filters from UserFilter and returns matching users.
func BuildGormQuery(query *gorm.DB, filter interface{}) error {
	// 1) Prepare a map: preloadPath → slice of (qrstr + values)
	//    We will collect conditions grouped by preload tag.
	assocConditions := make(map[string][]condStruct)
	//    Also collect top‐level (users.*) conditions under the empty key ""
	topConds := []condStruct{}

	// 2)scan the filter struct to fill assocConditions & topConds
	scanValue := reflect.ValueOf(filter) // underlying struct
	scanType := reflect.TypeOf(filter)   // type of UserFilter
	collectConditions(scanType, scanValue, assocConditions, &topConds)

	// 3) For each association path in assocConditions:
	for assocPath, conds := range assocConditions {
		if assocPath == "" {
			// This would never happen: we store top-level in topConds.
			continue
		}
		if len(conds) == 0 {
			// No filter: preload everything under that path
			query = query.Preload(assocPath)
		} else {
			// If there are any conditions, apply them in a closure
			query = query.Preload(assocPath, func(gdb *gorm.DB) *gorm.DB {
				for _, c := range conds {
					gdb = gdb.Where(c.sql, c.vals...)
				}
				return gdb
			})
			//query.Joins(assocPath) // Ensure the join is applied
		}
	}

	// 4) Apply any top-level conditions (filter on users table itself)
	for _, c := range topConds {
		query = query.Where(c.sql, c.vals...)
	}

	return nil
}

// collectConditions walks through a struct “typ”/“val”
// and collects conditions based on `qrstr` and `preload` tags.
// - parentAssoc is the association path so far (e.g. "Groups" or "Groups.Permissions"). "" for top-level fields.
// - assocConds accumulates map[assocPath][]condStruct
// - topConds accumulates conditions for top-level (assocPath = "")
func collectConditions(
	typ reflect.Type,
	val reflect.Value,
	assocConds map[string][]condStruct,
	topConds *[]condStruct,
) {
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fieldVal := val.Field(i)

		// Non-nil pointer: gather the condition
		qrstrTag := field.Tag.Get("qrstr")
		preloadTag := field.Tag.Get("preload") // e.g. "Roles" or ""
		if qrstrTag == "" {
			// If no qrstr, skip
			continue
		}

		//checking if assocConds has a key
		if _, ok := assocConds[field.Name]; !ok {
			//create an empty array with preload key inside assocConds
			assocConds[preloadTag] = []condStruct{}
		}

		if fieldVal.IsNil() {
			continue // no filter provided
		}

		// Dereference pointer to get actual value (which could be scalar or slice)
		var argValues []interface{}
		if fieldVal.Elem().Kind() == reflect.Slice {
			sliceVal := fieldVal.Elem()
			for j := 0; j < sliceVal.Len(); j++ {
				argValues = append(argValues, sliceVal.Index(j).Interface())
			}
		} else {
			argValues = append(argValues, fieldVal.Elem().Interface())
		}

		// Determine where to store this condition:
		if preloadTag != "" {
			// This filter applies to an association
			assocConds[preloadTag] = append(assocConds[preloadTag], condStruct{
				sql:  qrstrTag,
				vals: argValues,
			})
		} else {
			// This filter applies to the users table itself
			*topConds = append(*topConds, condStruct{
				sql:  qrstrTag,
				vals: argValues,
			})
		}

	}
}
