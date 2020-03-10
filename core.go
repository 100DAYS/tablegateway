package tablegateway

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type TableDataGateway interface {
	Find(int64, interface{}) error
	Update(int64, map[string]interface{}) error
	Insert(interface{}) error
	Delete(int64) error
}

type AutomatableTableDataGateway interface {
	TableDataGateway
	GetStruct() interface{}
	GetStructList() *[]interface{}
	FilterQuery(filters map[string]interface{}, order []string, offest int, limit int, into *[]interface{}) error
}

type TableGateway struct {
	DB           *sqlx.DB
	TableName    string
	KeyFieldName string
	names        string
	isPostgres   bool
	sq           squirrel.StatementBuilderType
}

func NewGw(DB *sqlx.DB, tableName string, keyFieldName string) TableGateway {
	isPgsql := (DB.DriverName() == "postgres")
	tg := TableGateway{
		DB:           DB,
		TableName:    tableName,
		KeyFieldName: keyFieldName,
		isPostgres:   isPgsql,
	}
	if isPgsql {
		tg.sq = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)
	} else {
		tg.sq = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question)
	}
	return tg
}

func NullString(s string) sql.NullString {
	return sql.NullString{Valid: true, String: s}
}

func NullInt64(s int64) sql.NullInt64 {
	return sql.NullInt64{Valid: true, Int64: s}
}

func NullInt32(s int32) sql.NullInt32 {
	return sql.NullInt32{Valid: true, Int32: s}
}

func NullFloat64(s float64) sql.NullFloat64 {
	return sql.NullFloat64{Valid: true, Float64: s}
}

func NullBool(s bool) sql.NullBool {
	return sql.NullBool{Valid: true, Bool: s}
}

func NullTime(s time.Time) sql.NullTime {
	return sql.NullTime{Valid: true, Time: s}
}

func (dao *TableGateway) getDBFieldnames(data interface{}) string {
	if dao.names == "" {
		fields := dao.DB.Mapper.FieldMap(reflect.ValueOf(data))
		// FieldMap maps Substructs with a path-structure like: parent.child where parent is the name of the field
		// in the main struct and child ist the name of one of the fields in the sub-struct.
		// we will need to scan for such paths and make sure that ony the fieldnames of the main struct are returned
		keys := make([]string, len(fields))
		mainkeys := make(map[string]int)

		i := 0
		for k := range fields {
			if strings.Contains(k, ".") {
				parts := strings.Split(k, ".")
        if parts[0] != dao.KeyFieldName {
          mainkeys[parts[0]] = 1
        }
      } else {
				keys[i] = k
				i++
			}
		}
		for k := range mainkeys {
			keys[i] = k
			i++
		}
		dao.names = strings.Join(keys[0:i-1], ",")
	}
	return dao.names
}

func makePlaceholders(fields string) string {
	if fields == "" {
		return ""
	}
	return ":" + strings.ReplaceAll(fields, ",", ",:")
}

func (dao *TableGateway) Insert(data interface{}) (lastInsertId int64, err error) {
	fields := dao.getDBFieldnames(data)
	placeholders := makePlaceholders(fields)
	q := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", dao.TableName, fields, placeholders)
	res, err := dao.DB.NamedExec(q, data) // @todo seems we need to do this the hard way...
	if err != nil {
		return
	}
	lastInsertId, err = res.LastInsertId()
	return
}

func (dao *TableGateway) Exec(qb squirrel.Sqlizer) (affected int64, err error) {
	s, args, err := qb.ToSql()
	if err != nil {
		return
	}
	res, err := dao.DB.Exec(s, args...)
	if err != nil {
		return
	}
	affected, err = res.RowsAffected()
	return
}

func (dao *TableGateway) Update(id int64, changes map[string]interface{}) (affected int64, err error) {
	qb := dao.sq.Update(dao.TableName).Where(dao.KeyFieldName+"=?", id)
	for fieldname, value := range changes {
		qb = qb.Set(fieldname, value)
	}
	return dao.Exec(qb)
}

func (dao *TableGateway) Delete(id int64) (affectedRows int64, err error) {
	qb := dao.sq.Delete(dao.TableName).Where(dao.KeyFieldName+"=?", id)
	return dao.Exec(qb)
}

func (dao *TableGateway) Find(id int64, target interface{}) error {
	qb := dao.SelectBuilder().Where(dao.KeyFieldName+"=?", id)
	q, args, err := qb.ToSql()
	if err != nil {
		return err
	}
	return dao.DB.Get(target, q, args...)
}

func (dao *TableGateway) SelectBuilder() squirrel.SelectBuilder {
	return dao.sq.Select("*").From(dao.TableName)
}

func (dao *TableGateway) Builder() squirrel.StatementBuilderType {
	return dao.sq
}

func (dao *TableGateway) FilterQuery(filters map[string]interface{}, order []string, offset uint64, limit int, into interface{}) error {
	qb := dao.SelectBuilder().Where(filters).OrderBy(strings.Join(order, ",")).Offset(offset).Limit(uint64(limit))
	return dao.Query(qb, into)
}

func (dao *TableGateway) Query(builder squirrel.SelectBuilder, into interface{}) error {
	var s, args, err = builder.ToSql()
	if err != nil {
		return err
	}
	err = dao.DB.Select(into, s, args...)
	return err
}
