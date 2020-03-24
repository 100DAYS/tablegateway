package tablegateway

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

func prepareDb() (*sqlx.DB, error) {
	db, err := sqlx.Open("sqlite3", ":memory:")
	//db, err := sqlx.Connect("postgres", "host=localhost user=example password=example dbname=example sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("Error opening sqlx: %s", err)
	}
	err = db.Ping()
	if err != nil {
		return nil, fmt.Errorf("Error pinging sqlx: %s", err)
	}

	_, err = db.Exec("DROP TABLE IF EXISTS places")
	if err != nil {
		return nil, fmt.Errorf("Error dropping Table: %s", err)
	}

	var ddl string
	if db.DriverName() == "postgres" {
		ddl = `CREATE TABLE places (
                id SERIAL PRIMARY KEY,
                country text,
                city text NULL,
                telcode integer);`
	} else {
		ddl = `CREATE TABLE places (
                id INTEGER PRIMARY KEY AUTOINCREMENT ,
                country text,
                city text NULL,
                telcode integer);`
	}

	_, err = db.Exec(ddl)
	if err != nil {
		return nil, fmt.Errorf("Error creating Schema: %s", err)
	}

	return db, err
}

func TestSqlx(t *testing.T) {

	db, err := prepareDb()
	if err != nil {
		t.Errorf("Cannot open DB: %s", err)
		return
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
		return
	}
	fmt.Printf("LastId was: %d\n", lastId)

	affected, err := dao.Update(lastId, map[string]interface{}{"city": "Stuggi", "telcode": 712})
	if err != nil {
		t.Errorf("Error updating record: %s", err)
		return
	}
	if affected != 1 {
		t.Errorf("Update should affect 1 row, did affect %d", affected)
		return
	}

	p2 := &place{}
	err = dao.Find(lastId, p2)
	if err != nil {
		t.Errorf("Error findind record: %s", err)
		return
	}
	if p2.Country != "Germany" {
		t.Errorf("Data wrong after find. Expected 'Germany' got %s", p2.Country)
		return
	}
	if p2.City.String != "Stuggi" {
		t.Errorf("Data wrong after update. Expected 'Stuggi' got %#v", p2.City)
		return
	}
	if p2.Telcode != 712 {
		t.Errorf("Data wrong after update. Expected 'Stuggi' got %d", p2.Telcode)
		return
	}
	if !p2.Id.Valid || p2.Id.Int64 != lastId {
		t.Errorf("Wrong id. Expected %d found %d", lastId, p2.Id.Int64)
		return
	}
	fmt.Printf("Found: %#v\n", p2)

	affected, err = dao.Delete(lastId)
	if err != nil {
		t.Errorf("Error deleting: %s", err)
		return
	}

	if affected != 1 {
		t.Errorf("AffectedRows not correct. Expected 1 found %d", affected)
		return
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

func (pg *PlaceGw) GetStruct() interface{} {
	return &Place{}
}
func (pg *PlaceGw) GetStructList() interface{} {
	return &[]Place{}
}

func (pg *PlaceGw) FindByCounty(country string) (places []Place, err error) {
	q := pg.SelectBuilder().Where("Country=?", country)
	err = pg.Query(q, &places)
	return
}

/*
func (pg *PlaceGw)Find(id int64) (pl Place, err error) {
    err = pg.TableGateway.Find(id, &pl)
    return
}
*/

func TestGw(t *testing.T) {
	db, err := prepareDb()
	if err != nil {
		t.Errorf("Error opening sqlx: %s", err)
		return
	}

	pg := NewPlaceGw(db)

	if _, err = pg.Insert(Place{Country: "Germany", City: NullString("Stuttgart"), Telcode: 711}); err != nil {
		t.Errorf("Error inserting: %s", err)
		return
	}
	if _, err = pg.Insert(Place{Country: "Germany", City: NullString("München"), Telcode: 89}); err != nil {
		t.Errorf("Error inserting: %s", err)
		return
	}
	if _, err = pg.Insert(Place{Country: "Germany", City: NullString("Berlin"), Telcode: 40}); err != nil {
		t.Errorf("Error inserting: %s", err)
		return
	}
	if _, err = pg.Insert(Place{Country: "Italy", City: NullString("Rome"), Telcode: 815}); err != nil {
		t.Errorf("Error inserting: %s", err)
		return
	}

	p := pg.GetStruct()
	err = pg.Find(1, p)
	if err != nil {
		t.Errorf("Error querying: %s", err)
		return
	}
	fmt.Printf("Record 1: \n%#v\n", p)

	foundId, err := pg.GetId(p)
	fmt.Printf("GetId says: %d\n", foundId)
	if foundId.(int64) != 1 {
		t.Errorf("GetID sollte 1 geben, hat aber %d returned.\n", foundId)
		return
	}

	list, err := pg.FindByCounty("Germany")
	if err != nil {
		t.Errorf("Error querying: %s", err)
		return
	}
	if len(list) != 3 {
		t.Errorf("Expected 3 results, found %d", len(list))
		return
	}
	fmt.Printf("Result: \n%#v\n", list)

	pl := pg.GetStructList()
	err = pg.FilterQuery(map[string]interface{}{"country": "Germany", "city": "München"}, []string{"telcode"}, 0, 10, pl)
	if err != nil {
		t.Errorf("Error filtering: %s", err)
		return
	}

	fmt.Printf("Filter Result is %#v\n", pl)

	pl2 := []Place{}
	qb := pg.SelectBuilder().Where("country=?", "Germany").OrderBy("telcode")
	err = pg.Query(qb, &pl2)
	if err != nil {
		t.Errorf("Error filtering: %s", err)
		return
	}
	fmt.Printf("Found %d records in germany\n", len(pl2))
}
