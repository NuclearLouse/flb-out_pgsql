package main

import "C"
import (
	"context"
	"encoding/json"
	"flb-out_pgsql/pgclient"
	"fmt"
	"sync"
	"unsafe"

	"log"
	"time"

	"github.com/fluent/fluent-bit-go/output"
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "pgsql", "PostgreSQL GO!")
}

var id int

type pgClients struct {
	sync.RWMutex
	m map[string]*pgclient.PGClient
}

var outputs *pgClients = &pgClients{
	m: make(map[string]*pgclient.PGClient),
}

// func init() {
// 	outputs = &pgClients{
// 		m: make(map[string]*pgclient.PGClient),
// 	}
// }

func (p *pgClients) Set(key string, client *pgclient.PGClient) {
	p.Lock()
	if p.m == nil {
		p.m = make(map[string]*pgclient.PGClient)
	}
	if _, ok := p.m[key]; !ok {
		p.m[key] = client
	}
	p.Unlock()
}

func (p *pgClients) Get(key string) *pgclient.PGClient {
	p.RLock()
	defer p.RUnlock()
	if cl, ok := p.m[key]; ok {
		return cl
	}
	return nil
}

//export FLBPluginInit
func FLBPluginInit(ctx unsafe.Pointer) int {

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable",
		output.FLBPluginConfigKey(ctx, "user"),
		output.FLBPluginConfigKey(ctx, "password"),
		output.FLBPluginConfigKey(ctx, "host_db"),
		output.FLBPluginConfigKey(ctx, "port_db"),
		output.FLBPluginConfigKey(ctx, "database"),
	)
	schema := output.FLBPluginConfigKey(ctx, "schema")
	table := output.FLBPluginConfigKey(ctx, "table")
	// match := output.FLBPluginConfigKey(ctx, "match")
	// outTable := fmt.Sprintf("%s_%d", table, IDOut)
	idOut := fmt.Sprintf("%d", id)
	output.FLBPluginSetContext(ctx, idOut)

	log.Printf("[ info] [pg_plugin] new pg_client id: %s connString:%s schema:%s table:%s\n", idOut, connString, schema, table)

	pgClient, err := pgclient.New(context.Background(), pgclient.NewConfig(connString, schema, table))
	if err != nil {
		log.Fatalln("[error] [pg_plugin] new pg_client:", err.Error())
	}

	outputs.Set(idOut, pgClient)

	if err := pgClient.CheckIfTableExist(context.Background()); err != nil {
		log.Fatalln("[error] [pg_plugin] check if table exists:", err.Error())
	}

	id++
	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	var datas []json.RawMessage
	dec := output.NewDecoder(data, int(length))
	for {
		ret, _, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}

		// timestamp := ts.(output.FLBTime)
		// date := time.Since(t).Seconds()

		for _, v := range record {
			type dataJSON struct {
				Log  string  `json:"log"`
				Date float64 `json:"date"`
			}

			t, _ := time.Parse("2006-01-02 15:04:05", "1970-01-01 00:00:00")
			switch rec := v.(type) {
			case []byte:
				d := dataJSON{
					Log:  string(rec),
					Date: time.Since(t).Seconds(),
				}
				b, err := json.Marshal(d)
				if err != nil {
					log.Println("[error] [pg_plugin] marshal log-record into json:", err.Error())
					continue
				}

				datas = append(datas, b)
			}

		}
	}

	connect := output.FLBPluginGetContext(ctx).(string)
	client := outputs.Get(connect)

	if len(datas) > 0 {
		log.Printf("[ info] [pg_plugin] flush called for table: %s.%s tag:%s logs:%d\n",client.Config.Schema, client.Config.Table, C.GoString(tag), len(datas))
		err := client.FlushLogs(context.Background(), C.GoString(tag), datas)
		if err != nil {
			log.Println("[error] [pg_plugin] flush json log-records to database:", err.Error())
		}
	} else {
		return output.FLB_RETRY
	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

//export FLBPluginExitCtx
func FLBPluginExitCtx(ctx unsafe.Pointer) int {
	connect := output.FLBPluginGetContext(ctx).(string)
	client := outputs.Get(connect)
	log.Println("[ info] [pg_plugin] exit called for instance:", connect)
	client.Close()
	return output.FLB_OK
}

//export FLBPluginFlush
// func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
// 	// log.Print("[pg_plugin] Flush called for unknown instance")
// 	return output.FLB_OK
// }

//export FLBPluginExit
// func FLBPluginExit() int {
// 	// log.Print("[pg_plugin] Exit called for unknown instance")
// 	return output.FLB_OK
// }

func main() {}
