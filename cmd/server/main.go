package main

import (
	"fmt"
	"github.com/andrewbackes/tourney/data/service"
	"github.com/andrewbackes/tourney/data/stores/memdb"
	"github.com/andrewbackes/tourney/server"
	log "github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

func main() {
	log.SetFormatter(new(prefixed.TextFormatter))
	log.SetLevel(log.DebugLevel)

	fmt.Println("Server")
	datastore := memdb.NewMemDB()
	service := service.New(datastore)
	s := server.New(":9090", service)
	s.Start()
}
