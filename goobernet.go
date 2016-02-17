package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/travissimon/goobernet/data"
)

const (
	DISCOVERY_PATH = "/discover/"
)

// For now we're assuming that all environments live on the same server
// This can be extended when/if that no longer holds

func swaggerIndexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "swagger/goobernet.swagger.json")
}

func marshalAndWrite(obj interface{}, w http.ResponseWriter) {
	json, _ := json.Marshal(obj)
	fmt.Fprintf(w, "%s", json)
}

func getProjectsHandler(w http.ResponseWriter, r *http.Request) {
	marshalAndWrite(data.GetProjects(), w)
}

func getEnvironmentsHandler(w http.ResponseWriter, r *http.Request) {
	marshalAndWrite(data.GetEnvironments(), w)
}

func getDeploymentsHandler(w http.ResponseWriter, r *http.Request) {
	marshalAndWrite(data.GetDeployments(), w)
}

func getDiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	environment := string(r.URL.Path[len(DISCOVERY_PATH):])
	deployments, err := data.GetDeploymentsByEnvironmentName(environment)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error with discovery: %s\n", err.Error())
		w.WriteHeader(400)
		w.Write([]byte(err.Error()))
		return
	}
	marshalAndWrite(deployments, w)
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func main() {
	var port = flag.String("port", "7777", "Define which TCP port to bind to")
	flag.Parse()

	fmt.Printf("Jenkins URL: %s\n", data.GetConfig().JenkinsUrl)

	http.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir("swagger"))))
	http.HandleFunc("/v1/projects", getProjectsHandler)
	http.HandleFunc("/v1/environments", getEnvironmentsHandler)
	http.HandleFunc("/v1/deployments", getDeploymentsHandler)
	http.HandleFunc(DISCOVERY_PATH, getDiscoveryHandler)
	http.HandleFunc("/healthz", healthzHandler)

	fmt.Printf("Starting Goobernet server on port %s\n", *port)
	http.ListenAndServe(":"+*port, nil)
}
