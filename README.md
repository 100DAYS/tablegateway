# Go Table Data Gateway

Simple implementation of Martin Fowler's Table Data Gateway Pattern in go.

We build on the shoulders of giants, therefore we require sqlx and the squirrel query builder.

## Example Table

```sqlite
    CREATE TABLE places (
        id INTEGER PRIMARY KEY AUTOINCREMENT ,
        country text,
        city text NULL,
        telcode integer
    )
```

## Instantiating a generic Table Gateway
We need a sqlx connection, a table name and the name of the primary key field.
Currently each table must have a int64 primary key field. This might change in later versions.

```Go
    db,err := sqlx.Open("sqlite3", ":memory:")
    if err != nil {
        t.Errorf("Error opening sqlx: %s", err)
    }

    gw := tablegateway.NewGw(db, "places", "id")
```

## Insert

We use a auto_inccrement field here, so we need to be able to set NULL values for the id field in order for it to work.
The sql package defines some NullXXX Types for this (thanks, @itinance for pointing this out.).
We added Factories for the NullXXX values to conveniently create non-null values. Uninitialized values are considered null.

### Insert with fix id
```Go
    type place struct {
        Id sql.NullInt64  `db:"id"`
        Country string `db:"country"`
        City sql.NullString `db:"city"`
        Telcode int    `db:"telcode"`
    }

    p := place{Id: NullInt64(23), Country: "Germany", City: NullString("Stuttgart"), Telcode: 711 }
    lastId, err := dao.Insert(p)
    if err != nil {
        t.Errorf("Error inserting to place: %s", err)
    }
    fmt.Printf("LastId was: %d\n", lastId)

```

### Insert with auto-generated id
We currently only support auto-increment primary keys like in sqlite, mysql or serial fields in postgresql
 
```Go
    type place struct {
        Id sql.NullInt64  `db:"id"`
        Country string `db:"country"`
        City sql.NullString `db:"city"`
        Telcode int    `db:"telcode"`
    }

    p := place{Country: "Germany", City: NullString("Stuttgart"), Telcode: 711 }
    lastId, err := dao.Insert(p)
    if err != nil {
        t.Errorf("Error inserting to place: %s", err)
    }
    fmt.Printf("LastId was: %d\n", lastId)
```
## Find
Each Table must have a integer primary key (for now). To load a records with a known key, we use:
```Go
    p2 := &place{}
    err = dao.Find(lastId, p2)
    if err != nil {
        t.Errorf("Error finding record: %s", err)
    }
    if p2.Country != "Germany" {
        t.Errorf("Data wrong after find. Expected 'Germany' got %s", p2.Country)
    }
    if !p2.Id.Valid || p2.Id.Int64 != lastId {
        t.Errorf("Wrong id. Expected %d found %d", lastId, p2.Id.Int64)
    }
    fmt.Printf("Found: %#v\n", p2)
```
## Update
```Go
    affected, err := dao.Update(lastId, map[string]interface{}{"city": "Stuggi", "telcode": 712})
    if err != nil {
        t.Errorf("Error updating record: %s", err)
    }
    if affected != 1 {
        t.Errorf("Update should affect 1 row, did affect %d", affected)
    }
```

## Delete
```Go
    affected, err := dao.Delete(lastId)
    if err != nil {
        t.Errorf("Error deleting: %s", err)
    }

    if affected != 1 {
        t.Errorf("AffectedRows not correct. Expected 1 found %d", affected)
    }
```

## Custom Gateways
Table Data Gateways should contain all Queries that will ever be thrown at the database so database optimization can be done based on the contents of these classes.
In order to do so, we should inherit from TableGateway base Class to Produce new specialized Gateways as:  
```Go
type PlaceGw struct {
    TableGateway
}

func NewPlaceGw(db *sqlx.DB) PlaceGw {
    return PlaceGw{NewGw(db, "places", "id")}
}

type Place struct {
    Id sql.NullInt64  `db:"id"`
    Country string `db:"country"`
    City sql.NullString `db:"city"`
    Telcode int    `db:"telcode"`
}

func (pg *PlaceGw)FindByCounty(country string) (places []Place, err error) {
    q := "SELECT * from places WHERE Country=?"
    err = pg.DB.Select(&places, q, country)
    return
}
```
Calling code will then just instantiate a PlaceGw handing over a Database instance and then it can make queries by calling functions on this object.

```Go
    pg := NewPlaceGw(db)
    list, err := pg.FindByCounty("Germany")
    if err != nil {
        t.Errorf("Error querying: %s", err)
    }
    // list will now be a []Place data structure
```

## Automation / Generalization
In order to use the TableGateway in a generalized way, derived classes can implement the AutomatableTableDataGateway.
This requires the coding of some helper functions to enable generalized use of the class by calling code.
NOTE: It is important these functions return pointers to the respective data structures.

```Go
func (pg *PlaceGw)GetStruct() interface{} {
    return &Place{}
}
func (pg *PlaceGw)GetStructList() interface{} {
    return &[]Place{}
}
``` 

These functions return the result structure to be used to instantiate the needed data structures for the results of Find(...) and FilterQuery functions:
```Go
    p := pg.GetStruct()
    err = pg.Find(1, p)
    if err != nil {
        t.Errorf("Error querying: %s", err)
    }
    fmt.Printf("Record 1: \n%#v\n", p)

    // or for lists:

    pl := pg.GetStructList()
    err = pg.FilterQuery(map[string]interface{}{"country": "Germany", "city": "München"}, []string{"telcode"}, 0, 10, pl )
    if err != nil {
        t.Errorf("Error filtering: %s", err)
    }
```
This can be helpful if the calling code would want to just serialize the response...

