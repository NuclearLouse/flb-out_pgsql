package main

import "C"
import (
	"context"
	"encoding/json"
	"flb-out_pgsql/pgclient"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"time"

	"github.com/fluent/fluent-bit-go/output"

	"flb-out_pgsql/logger"

	"github.com/sirupsen/logrus"
	"gopkg.in/ini.v1"
)

var (
	pgClientID int
	cfg        *ini.File
	log        *logrus.Logger
)

const (
	version     = "2.1.1"
	configFile  = "pg-plugin.conf"
	pluginName  = "pgsql"
	description = "Fluent-Bit Postgresql Output Plugin written in Golang!"
)

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	exec, err := os.Executable()
	if err != nil {
		return output.FLB_ERROR

	}
	exepath := filepath.Dir(exec)
	newPath := filepath.Join(strings.Replace(exepath, filepath.Base(exepath), "conf", 1))

	cfg, err = ini.LoadSources(ini.LoadOptions{
		SpaceBeforeInlineComment: true,
	},
		filepath.Join(newPath, configFile))
	if err != nil {
		return output.FLB_ERROR
	}
	cfg.NameMapper = ini.TitleUnderscore
	cfgLog := logger.DefaultConfig()

	if err := cfg.Section("logger").MapTo(cfgLog); err != nil {
		return output.FLB_ERROR
	}

	log = logger.New(cfgLog)

	log.Infof("[pg-plugin] register %s Version:%s with name:%s", description, version, pluginName)

	return output.FLBPluginRegister(ctx, pluginName, description)
}

type pgClients struct {
	sync.RWMutex
	m map[string]*pgclient.PGClient
}

var outputs *pgClients = &pgClients{
	m: make(map[string]*pgclient.PGClient),
}

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
	idOut := fmt.Sprintf("%d", pgClientID)
	output.FLBPluginSetContext(ctx, idOut)

	pgClient, err := pgclient.New(context.Background(), pgclient.NewConfig(connString, schema, table), log)
	if err == nil {
		log.Infof("[pg-plugin] init new pg_client id: %s connString:%s schema:%s table:%s\n", idOut, connString, schema, table)
	} else {
		log.Errorln("[pg-plugin] init new pg_client:", err)
		return output.FLB_ERROR
	}

	outputs.Set(idOut, pgClient)

	if err := pgClient.CheckIfTableExist(context.Background()); err != nil {
		log.Errorln("[pg-plugin] check exists table:", err)
		return output.FLB_ERROR
	}

	pgClientID++
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
					log.Errorf("[pg-plugin] marshal log-record into json: %s", err)
					continue
				}
				datas = append(datas, b)
			}
		}
	}

	connect := output.FLBPluginGetContext(ctx).(string)
	client := outputs.Get(connect)

	if len(datas) > 0 {
		log.Infof("[pg-plugin] flush called for table: %s.%s tag:%s logs:%d\n", client.Config.Schema, client.Config.Table, C.GoString(tag), len(datas))
		err := client.FlushLogs(context.Background(), C.GoString(tag), datas)
		if err != nil {
			log.Errorln("[pg-plugin] flush json log-records to database:", err)
			//? он будет пытаться отправит только эти данные, надо дать число попыток?
			return output.FLB_RETRY
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
	log.Infoln("[pg-plugin] exit called for instance:", connect)
	client.Close()
	return output.FLB_OK
}

func main() {}
