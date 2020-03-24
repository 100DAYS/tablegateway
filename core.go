package tablegateway

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
)

type TableDataGateway interface {
	Find(int64, interface{}) error
	Update(int64, map[string]interface{}) (int64, error)
	Insert(interface{}) (int64, error)
	Delete(int64) (int64, error)
}

type AutomatableTableDataGateway interface {
	TableDataGateway
	GetStruct() interface{}
	GetStructList() interface{}
	FilterQuery(filters map[string]interface{}, order []string, offset int, limit int, into interface{}) error
	GetId(interface{}) (int64, error)
}

type TableGateway struct {
	DB           *sqlx.DB
	TableName    string
	KeyFieldName string
	nameHash     map[string]int
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

func (dao *TableGateway) Insert(data interface{}) (lastInsertId int64, err error) {
	if dao.isPostgres {
		return dao.insertPostgres(data)
	} else {
		return dao.insertMysql(data)
	}
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

func (dao *TableGateway) Query(builder squirrel.SelectBuilder, into interface{}) error {
	var s, args, err = builder.ToSql()
	if err != nil {
		return err
	}
	err = dao.DB.Select(into, s, args...)
	return err
}

func (dao *TableGateway) FilterQuery(filters map[string]interface{}, order []string, offset int, limit int, into interface{}) error {
	qb := dao.SelectBuilder().Where(filters).OrderBy(strings.Join(order, ",")).Offset(uint64(offset)).Limit(uint64(limit))
	q, args, _ := qb.ToSql()
	fmt.Printf("SQL is: %s with args: %#v", q, args)
	return dao.Query(qb, into)
}

// This interface schould be Implemented by all sql.NullXXX types
type Nullable interface {
	Value() (driver.Value, error)
}

func (dao *TableGateway) GetId(rec interface{}) (interface{}, error) {
	fields := dao.DB.Mapper.FieldMap(reflect.ValueOf(rec))
	idField, found := fields[dao.KeyFieldName]
	if !found {
		return 0, fmt.Errorf("Id Field %s not found in struct", dao.KeyFieldName)
	}
	if n, ok := idField.Interface().(Nullable); ok {
		dval, err := n.Value() // returns driver.Value
		return dval, err
	} else if idField.Kind() == reflect.Struct {
		return nil, fmt.Errorf("Struct not allowed as ID Field %#v", idField.Interface())
	} else {
		return idField.Interface(), nil
	}
}
