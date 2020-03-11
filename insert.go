package tablegateway

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

func (dao *TableGateway) getDBFieldnames(data interface{}) string {
	fields := dao.DB.Mapper.FieldMap(reflect.ValueOf(data))
	if dao.nameHash == nil {
		// FieldMap maps Substructs with a path-structure like: parent.child where parent is the name of the field
		// in the main struct and child ist the name of one of the fields in the sub-struct.
		// we will need to scan for such paths and make sure that ony the fieldnames of the main struct are returned
		dao.nameHash = make(map[string]int, len(fields))

		for k := range fields {
			// check for names like "id.Int64" because these have been expanded by the Mapper
			// from structs like sql.NullInt64. For such fields, only the part before the "." are needed
			if strings.Contains(k, ".") {
				parts := strings.Split(k, ".")
				dao.nameHash[parts[0]] = 1
			} else {
				dao.nameHash[k] = 1
			}
		}
	}
	// check if primaryKeyField is a NullInt64 and it is nil. If so, skip field
	removeKeyField := 0
	idField, found := fields[dao.KeyFieldName]
	if found && idField.Type() == reflect.TypeOf(sql.NullInt64{0, false}) {
		ni, _ := idField.Interface().(sql.NullInt64)
		if !ni.Valid {
			removeKeyField = 1
		}
	}
	// now assemble the field list string from the resulting hash, omitting blacklisted fields...
	names := make([]string, len(dao.nameHash)-removeKeyField)
	i := 0
	for k := range dao.nameHash {
		if removeKeyField == 0 || k != dao.KeyFieldName {
			names[i] = k
			i++
		}
	}

	return strings.Join(names, ",")
}

func makePlaceholders(fields string) string {
	if fields == "" {
		return ""
	}
	return ":" + strings.ReplaceAll(fields, ",", ",:")
}

func (dao *TableGateway) insertPostgres(data interface{}) (lastInsertId int64, err error) {
	fields := dao.getDBFieldnames(data)
	placeholders := makePlaceholders(fields)
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s", dao.TableName, fields, placeholders, dao.KeyFieldName)
	stmt, err := dao.DB.PrepareNamed(q)
	if err != nil {
		return 0, err
	}
	err = stmt.Get(&lastInsertId, data)
	return
}

func (dao *TableGateway) insertMysql(data interface{}) (lastInsertId int64, err error) {
	fields := dao.getDBFieldnames(data)
	placeholders := makePlaceholders(fields)
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", dao.TableName, fields, placeholders)
	fmt.Printf("SQL: %s", q)
	res, err := dao.DB.NamedExec(q, data)
	if err != nil {
		return 0, err
	}
	lastInsertId, err = res.LastInsertId()
	return
}
