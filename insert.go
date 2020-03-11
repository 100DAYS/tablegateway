package tablegateway

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

func (dao *TableGateway) getDBFieldnames(data interface{}) string {
	if dao.names == "" {
		fields := dao.DB.Mapper.FieldMap(reflect.ValueOf(data))
		// FieldMap maps Substructs with a path-structure like: parent.child where parent is the name of the field
		// in the main struct and child ist the name of one of the fields in the sub-struct.
		// we will need to scan for such paths and make sure that ony the fieldnames of the main struct are returned
		fieldHash := make(map[string]int, len(fields))
		blackList := make(map[string]int)

		for k, f := range fields {
			// check if primaryKeyField is a NullInt64 and it is nil. If so, skip field
			if k == dao.KeyFieldName && f.Type() == reflect.TypeOf(sql.NullInt64{0, false}) {
				ni, _ := f.Interface().(sql.NullInt64)
				if !ni.Valid {
					blackList[k] = 1
				}
			}
			// check for names like "id.Int64" because these have been expanded by the Mapper
			// from structs like sql.NullInt64. For such fields, only the part before the "." are needed
			if strings.Contains(k, ".") {
				parts := strings.Split(k, ".")
				fieldHash[parts[0]] = 1
			} else {
				fieldHash[k] = 1
			}
		}
		// now assemble the field list string from the resulting hash, omitting blacklisted fields...
		res := make([]string, len(fieldHash)-len(blackList))
		i := 0
		for k := range fieldHash {
			if _, found := blackList[k]; !found {
				res[i] = k
				i++
			}
		}
		dao.names = strings.Join(res, ",")
	}
	return dao.names
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
	res, err := dao.DB.NamedExec(q, data)
	if err != nil {
		return 0, err
	}
	lastInsertId, err = res.LastInsertId()
	return
}
