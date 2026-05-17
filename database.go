/*
* Copyright 2025-2026 Thorsten A. Knieling
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*    http://www.apache.org/licenses/LICENSE-2.0
*
 */

package energymonitor

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/tknie/ecoflow"
	"github.com/tknie/flynn"
	"github.com/tknie/flynn/common"
	"github.com/tknie/log"
	"github.com/tknie/services"
)

var dbTables []string
var dbRef *common.Reference
var dbPassword string

var specialFields = []string{"timestamp", "vendor", "serial_number"}

type storeElement struct {
	sn     string
	object any
}

var msgChan = make(chan *storeElement, 100)

// InitDatabase init database connections
func InitDatabase() {
	databaseUrl := adapter.DatabaseConfig.Target
	if databaseUrl == "" {
		databaseUrl = os.Getenv("ECOFLOW_DB_URL")
		databaseUrl = os.ExpandEnv(databaseUrl)
	}
	var err error
	dbRef, dbPassword, err = common.NewReference(databaseUrl)
	if err != nil {
		services.ServerMessage("Shuting down ... URL is incorrect or cannot be parsed: %v", err)
		log.Log.Fatalf("REST audit URL incorrect: " + databaseUrl)
	}
	dbRef.Options = append(dbRef.Options, fmt.Sprintf("application_name=energymonitor %s", Version))
	if dbRef.User == "" {
		dbRef.User = os.Getenv("ECOFLOW_DB_USER")
	}
	if dbPassword == "" {
		dbPassword = os.Getenv("ECOFLOW_DB_PASS")
	}
	log.Log.Debugf("DB password: %s", dbPassword)
	_, err = flynn.Handler(dbRef, dbPassword)
	if err != nil {
		services.ServerMessage("Shuting down ... register database error: %v", err)
		log.Log.Fatalf("Register error log: %v", err)
	}
	readDatabaseMaps()
	go storeDatabase()
}

// readDatabaseMaps read database tables to check for
func readDatabaseMaps() {
	dbTables = flynn.Maps()
	log.Log.Debugf("Tables: %#v", dbTables)
}

// ConnnectDatabase connect connection to database for the corresponding storage
func ConnnectDatabase() common.RegDbID {
	log.Log.Debugf("Connected to database %s", dbRef)
	id, err := flynn.Handler(dbRef, dbPassword)
	if err != nil {
		services.ServerMessage("Shuting down, connect database error: %v", err)
		log.Log.Fatalf("Connect database error: %v", err)
	}
	return id
}

// storeDatabase final insert into database with device information
func storeDatabase() {
	storeid := ConnnectDatabase()
	for m := range msgChan {
		tn := strings.ToLower("mqtt_" + ecoflow.GetTypeName(m.object))
		m.checkStoreElementTable(tn, storeid)

		log.Log.Debugf("Insert structFields: %T into tn", m.object)
		fields := []string{"*"}
		insert := &common.Entries{DataStruct: m.object,
			Fields: fields}
		insert.Values = [][]any{{m.object}}
		_, err := storeid.Insert(tn, insert)
		if err != nil {
			services.ServerMessage("Error inserting record: %v", err)
			// Reconnecting ...
			storeid.Close()
			storeid = ConnnectDatabase()
			// log.Log.Fatal("Error inserting record: ", err)
		} else {
			getDbStatEntry(tn).counter++
		}

	}
}

// CheckTableExists check if table is available and if not, create it
func (m *storeElement) checkStoreElementTable(tn string, storeid common.RegDbID) {
	if tn == "" {
		log.Log.Fatal("Check failed, database not given")
	}
	if !slices.Contains(dbTables, tn) {
		services.ServerMessage("Database %s need to be created", tn)
		err := storeid.CreateTable(tn, m.object)
		if err != nil {
			log.Log.Fatal("Error creating database table: ", err)
		}
		readDatabaseMaps()
	}
}

// CheckTableExists check table and if not available, create table
func CheckTableExists(storeid common.RegDbID, tn string, generateColumns func() []*common.Column) bool {
	if tn == "" {
		log.Log.Fatal("Error check table, database not defined")
	}
	readDatabaseMaps()
	if !slices.Contains(dbTables, strings.ToLower(tn)) {
		services.ServerMessage("Database check failed, %s need to be created", tn)
		err := storeid.CreateTable(tn, generateColumns())
		if err != nil {
			services.ServerMessage("Shuting down ... error creating database for %s : %v", tn, err)
			log.Log.Fatal("Error creating database table: ", err)
		}
		readDatabaseMaps()
		return false
	}
	return true
}

// InsertTable insert data into database
func InsertTable(storeid common.RegDbID, tn, prefix string, data map[string]interface{}, generateColumns func(prefix string, data map[string]interface{}) ([]string, [][]any)) error {
	fields, values := generateColumns(prefix, data)
	log.Log.Debugf("Insert columnFields: %#v", fields)
	if len(fields) == 0 {
		return nil
	}
	insert := &common.Entries{
		Values: values,
		Fields: fields}
	_, err := storeid.Insert(tn, insert)
	if err != nil {
		services.ServerMessage("Error inserting record in table: %v", err)
	} else {
		getDbStatEntry(tn).counter++
	}
	return err
}

// InsertTable insert data into database
func readBatch(readid common.RegDbID, tn string, selectCmd string, f func(search *common.Query, result *common.Result) error) error {
	query := common.Query{Search: selectCmd}
	err := readid.BatchSelectFct(&query, f)
	return err
}

// createValueColumn create value columns dependent of the information
// received by HTTP request
func CreateValueColumn(name string, v interface{}) *common.Column {
	switch strings.ToLower(name) {
	case "timestamp":
		return &common.Column{Name: name, DataType: common.CurrentTimestamp, Length: 8}
	case "vendor":
		return &common.Column{Name: name, DataType: common.Alpha, Length: 255}
	case "serial_number":
		return &common.Column{Name: name, DataType: common.Alpha, Length: 100}
	}
	switch val := v.(type) {
	case string:
		return &common.Column{Name: name, DataType: common.Alpha, Length: 255}
	case bool:
		return &common.Column{Name: name, DataType: common.Bytes, Length: 1}
	case time.Time:
		return &common.Column{Name: name, DataType: common.CurrentTimestamp, Length: 8}
	case float64:
		if val == math.Trunc(val) && val < math.MaxInt64 {
			return &common.Column{Name: name, DataType: common.BigInteger, Length: 0}
		} else {
			return &common.Column{Name: name, DataType: common.Decimal, Length: 8}
		}
	case int64:
		return &common.Column{Name: name, DataType: common.BigInteger, Length: 0}
	case uint8:
		return &common.Column{Name: name, DataType: common.BigInteger, Length: 0}
	case []interface{}, map[string]interface{}:
		b, err := json.Marshal(val)
		if err != nil {
			services.ServerMessage("Error marshal: %#v", val)
			return nil
		}
		s := string(b)
		l := uint16(1024)
		if len(s) > 1024 {
			l += uint16(1024) + uint16(len(s))
		}
		return &common.Column{Name: name, DataType: common.Alpha, Length: l}
	default:
		services.ServerMessage("Unknown type %s=%T", name, v)
	}
	log.Log.Errorf("Unknown type %s=%T", name, v)
	return nil
}

// checkTableColumns check if new parameters are in current request to adapt table
func AddTableColumns(id common.RegDbID, tn, prefix string, data map[string]interface{}) {
	col, err := id.GetTableColumn(tn)
	if err != nil {
		services.ServerMessage("Get table column %v", err)
		return
	}
	log.Log.Debugf("Validate to defined columns %#v", col)
	columns := make([]*common.Column, 0)
	fields, value := collectKeys(data, prefix)
	for i, k := range fields {
		name := strings.ReplaceAll(strings.ToLower(k), ".", "_")
		if len(name) > 63 {
			name = name[0:63]
		}
		if !slices.Contains(col, name) {
			log.Log.Debugf("Column not in table %s=%s", k, name)
			c := CreateValueColumn(name, value[i])
			columns = append(columns, c)
		}
	}
	if len(columns) > 0 {
		log.Log.Debugf("Add %d. columns to table %T", len(columns), columns)
		err = id.AdaptTable(tn, columns)
		log.Log.Debugf("Added %d. columns to table: %v", len(columns), err)
	}
}

func InsertDeapData(prefix string, data map[string]interface{}) ([]string, [][]any) {

	fields, columns := collectKeys(data, prefix)
	for i, f := range fields {
		log.Log.Debugf("Insert columnFields: %s=%v", f, columns[i])
	}
	return fields, [][]any{columns}
}

func collectKeys(data map[string]interface{}, prefix string) ([]string, []any) {
	fields := make([]string, 0)
	columns := make([]any, 0)
	for k, v := range data {
		switch k {
		case "timestamp":
			fields = append(fields, "timestamp")
			columns = append(columns, v.(time.Time))
		case "vendor":
			fields = append(fields, "vendor")
			columns = append(columns, v.(string))
		case "serial_number":
			fields = append(fields, "serial_number")
			columns = append(columns, v.(string))
		default:
			log.Log.Debugf("Collect key: %s", prefix+"_"+k)
			switch val := v.(type) {
			case string:
				fields = append(fields, prefix+"_"+strings.ReplaceAll(strings.ToLower(k), ".", "_"))
				columns = append(columns, val)
			case bool:
				fields = append(fields, prefix+"_"+strings.ReplaceAll(strings.ToLower(k), ".", "_"))
				if val {
					columns = append(columns, byte(1))
				} else {
					columns = append(columns, byte(0))
				}
			case float64:
				fields = append(fields, prefix+"_"+strings.ReplaceAll(strings.ToLower(k), ".", "_"))
				if val == math.Trunc(val) {
					columns = append(columns, int64(val))
				} else {
					columns = append(columns, val)
				}
			case int64:
				fields = append(fields, prefix+"_"+strings.ReplaceAll(strings.ToLower(k), ".", "_"))
				columns = append(columns, val)
			case uint8:
				fields = append(fields, prefix+"_"+strings.ReplaceAll(strings.ToLower(k), ".", "_"))
				columns = append(columns, val)
			case time.Time:
				fields = append(fields, prefix+"_"+strings.ReplaceAll(strings.ToLower(k), ".", "_"))
				columns = append(columns, val)
			case map[string]interface{}:
				subf, subv := collectKeys(val, prefix+"_"+strings.ReplaceAll(strings.ToLower(k), ".", "_"))
				fields = append(fields, subf...)
				columns = append(columns, subv...)
			case []interface{}:
				columns = append(columns, fmt.Sprintf("%v", val))
			default:
				services.ServerMessage("Unknown HTTP JSON type %s=%T -> %v", k, v, v)
				log.Log.Errorf("Unknown type %s=%T", k, v)
			}
		}
	}
	return fields, columns
}

// insertHttpData prepare database data to be inserted into the database
func InsertHttpData(prefix string, data map[string]interface{}) ([]string, [][]any) {
	keys := make([]string, 0)
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	columns := make([]any, 0)
	// prefix := ""
	fields := make([]string, 0)
	for _, k := range keys {
		v := data[k]
		// prefix = strings.Split(k, ".")[0]
		// name := "eco_" + strings.ReplaceAll(k[len(prefix)+1:], ".", "_")
		name := strings.ReplaceAll(k, ".", "_")
		if !slices.Contains(specialFields, name) {
			name = prefix + "_" + name
		}
		fields = append(fields, name)
		switch val := v.(type) {
		case string:
			columns = append(columns, val)
		case bool:
			if val {
				columns = append(columns, byte(1))
			} else {
				columns = append(columns, byte(0))
			}
		case int64:
			columns = append(columns, val)
		case uint8:
			columns = append(columns, val)
		case float64:
			if val == math.Trunc(val) {
				columns = append(columns, int64(val))
			} else {
				columns = append(columns, val)
			}
		case time.Time:
			columns = append(columns, val)
		case []interface{}, map[string]interface{}:
			b, err := json.Marshal(val)
			if err != nil {
				services.ServerMessage("Error marshal: %#v", val)
			} else {
				s := string(b)
				columns = append(columns, s)
			}
		default:
			services.ServerMessage("Unknown HTTP JSON type %s=%T", k, v)
			log.Log.Errorf("Unknown type %s=%T", k, v)
		}
	}
	return fields, [][]any{columns}
}
