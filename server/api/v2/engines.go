package api

import (
	"fmt"
	"github.com/andrewbackes/tourney/data/services"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
)

func getEngineFile(s services.Tournament) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		name := vars["name"]
		version := vars["version"]
		osName := vars["os"]
		filepath := "tourney_storage/engineFiles/" + name + "/" + version + "/" + osName
		filename := name + "-" + version + "-" + osName
		if _, err := os.Stat(filepath + "/" + filename); os.IsNotExist(err) {
			log.Error(filepath + "/" + filename + " does not exist")
			return
		}
		http.ServeFile(w, req, filepath+"/"+filename)
	}
}

func postEngineFile(s services.Tournament) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		vars := mux.Vars(req)
		name := vars["name"]
		version := vars["version"]
		osName := vars["os"]

		req.ParseMultipartForm(32 << 20)
		file, handler, err := req.FormFile("uploadfile")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		fmt.Fprintf(w, "%v", handler.Header)
		dirname := "tourney_storage/engineFiles/" + name + "/" + version + "/" + osName
		filename := name + "-" + version + "-" + osName
		err = os.MkdirAll(dirname, os.ModePerm)
		if err != nil {
			panic(err)
		}
		f, err := os.OpenFile(dirname+"/"+filename, os.O_WRONLY|os.O_CREATE, 0755)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		io.Copy(f, file)
	}
}