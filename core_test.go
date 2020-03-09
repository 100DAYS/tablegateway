package tablegateway

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

func TestSqlx(t *testing.T) {
	var db *sqlx.DB

	// exactly the same as the built-in
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Errorf("Error opening sqlx: %s", err)
	}
	err = db.Ping()
	if err != nil {
		t.Errorf("Error pinging sqlx: %s", err)
	}

	schema := `CREATE TABLE places (
    id INTEGER PRIMARY KEY AUTOINCREMENT ,
    country text,
    city text NULL,
    telcode integer);`

	// execute a query on the server
	_, err = db.Exec(schema)
	if err != nil {
		t.Errorf("Error creating Schema: %s", err)
	}

	dao := NewGw(db, "places", "id")

	type place struct {
		Id      sql.NullInt64  `db:"id"`
		Country string         `db:"country"`
		City    sql.NullString `db:"city"`
		Telcode int            `db:"telcode"`
	}

	p := place{Id: NullInt64(23), Country: "Germany", City: NullString("Stuttgart"), Telcode: 711}
	lastId, err := dao.Insert(p)
	if err != nil {
		t.Errorf("Error inserting to place: %s", err)
	}
	fmt.Printf("LastId was: %d\n", lastId)

	affected, err := dao.Update(lastId, map[string]interface{}{"city": "Stuggi", "telcode": 712})
	if err != nil {
		t.Errorf("Error updating record: %s", err)
	}
	if affected != 1 {
		t.Errorf("Update should affect 1 row, did affect %d", affected)
	}

	p2 := &place{}
	err = dao.Find(lastId, p2)
	if err != nil {
		t.Errorf("Error findind record: %s", err)
	}
	if p2.Country != "Germany" {
		t.Errorf("Data wrong after find. Expected 'Germany' got %s", p2.Country)
	}
	if p2.City.String != "Stuggi" {
		t.Errorf("Data wrong after update. Expected 'Stuggi' got %#v", p2.City)
	}
	if p2.Telcode != 712 {
		t.Errorf("Data wrong after update. Expected 'Stuggi' got %d", p2.Telcode)
	}
	if !p2.Id.Valid || p2.Id.Int64 != lastId {
		t.Errorf("Wrong id. Expected %d found %d", lastId, p2.Id.Int64)
	}
	fmt.Printf("Found: %#v\n", p2)

	affected, err = dao.Delete(lastId)
	if err != nil {
		t.Errorf("Error deleting: %s", err)
	}

	if affected != 1 {
		t.Errorf("AffectedRows not correct. Expected 1 found %d", affected)
	}
}

type PlaceGw struct {
	TableGateway
}

func NewPlaceGw(db *sqlx.DB) PlaceGw {
	return PlaceGw{NewGw(db, "places", "id")}
}

type Place struct {
	Id      sql.NullInt64  `db:"id"`
	Country string         `db:"country"`
	City    sql.NullString `db:"city"`
	Telcode int            `db:"telcode"`
}

func (pg *PlaceGw) CreateTable() (err error) {
	schema := `CREATE TABLE places (
    id INTEGER PRIMARY KEY AUTOINCREMENT ,
    country text,
    city text NULL,
    telcode integer);`
	_, err = pg.DB.Exec(schema)
	return
}

func (pg *PlaceGw) GetStruct() interface{} {
	return &Place{}
}
func (pg *PlaceGw) GetStructList() interface{} {
	return &[]Place{}
}

func (pg *PlaceGw) FindByCounty(country string) (places []Place, err error) {
	q := "SELECT * from places WHERE Country=?"
	err = pg.DB.Select(&places, q, country)
	return
}

/*
func (pg *PlaceGw)Find(id int64) (pl Place, err error) {
    err = pg.TableGateway.Find(id, &pl)
    return
}
*/

func TestGw(t *testing.T) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	if err != nil {
		t.Errorf("Error opening sqlx: %s", err)
	}

	pg := NewPlaceGw(db)
	err = pg.CreateTable()
	if err != nil {
		t.Errorf("Error creating table: %s", err)
	}

	if _, err = pg.Insert(Place{Country: "Germany", City: NullString("Stuttgart"), Telcode: 711}); err != nil {
		t.Errorf("Error inserting: %s", err)
	}
	if _, err = pg.Insert(Place{Country: "Germany", City: NullString("München"), Telcode: 89}); err != nil {
		t.Errorf("Error inserting: %s", err)
	}
	if _, err = pg.Insert(Place{Country: "Germany", City: NullString("Berlin"), Telcode: 40}); err != nil {
		t.Errorf("Error inserting: %s", err)
	}
	if _, err = pg.Insert(Place{Country: "Italy", City: NullString("Rome"), Telcode: 815}); err != nil {
		t.Errorf("Error inserting: %s", err)
	}

	p := pg.GetStruct()
	err = pg.Find(1, p)
	if err != nil {
		t.Errorf("Error querying: %s", err)
	}
	fmt.Printf("Record 1: \n%#v\n", p)

	list, err := pg.FindByCounty("Germany")
	if err != nil {
		t.Errorf("Error querying: %s", err)
	}
	if len(list) != 3 {
		t.Errorf("Expected 3 results, found %d", len(list))
	}
	fmt.Printf("Result: \n%#v\n", list)

	pl := pg.GetStructList()
	err = pg.FilterQuery(map[string]interface{}{"country": "Germany", "city": "München"}, []string{"telcode"}, 0, 10, pl)
	if err != nil {
		t.Errorf("Error filtering: %s", err)
	}

	fmt.Printf("Filter Result is %#v\n", pl)

	pl2 := []Place{}
	qb := pg.InitQueryBuilder().Where("country=?", "Germany").OrderBy("telcode")
	err = pg.Query(qb, &pl2)
	if err != nil {
		t.Errorf("Error filtering: %s", err)
	}
	fmt.Printf("Found %d records in germany\n", len(pl2))
}
