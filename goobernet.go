package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/travissimon/goobernet/ci"
	"github.com/travissimon/goobernet/data"
	"github.com/travissimon/goobernet/docker"
)

const (
	DISCOVERY_PATH = "/v1/discover/"
	JOB_PATH       = "/v1/job/"
)

// For now we're assuming that all environments live on the same server
// This can be extended when/if that no longer holds

func swaggerIndexHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "swagger/goobernet.swagger.json")
}

func writeError(w http.ResponseWriter, code int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, msg)
	http.Error(w, msg, code)
}

func marshalAndWrite(obj interface{}, w http.ResponseWriter) {
	json, err := json.Marshal(obj)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error marshalling object: %s\n", err)
		return
	}
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

func getTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	marshalAndWrite(data.GetTemplates(), w)
}

func getContainersHandler(w http.ResponseWriter, r *http.Request) {
	containers, err := docker.GetContainers()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error retrieving docker containers: %s\n", err)
		return
	}
	marshalAndWrite(containers, w)
}

func getDiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	environment := string(r.URL.Path[len(DISCOVERY_PATH):])
	deployments, err := data.GetDeploymentsByEnvironmentName(environment)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error with discovery: %s\n", err.Error())
		return
	}
	marshalAndWrite(deployments, w)
}

func getJobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs, err := ci.Proxy.GetTasks()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error querying ci server: %s\n", err.Error())
		return
	}
	marshalAndWrite(jobs, w)
}

// handles requests for /job/(job-name)
func getJobHandler(w http.ResponseWriter, r *http.Request) {
	jobName := string(r.URL.Path[len(JOB_PATH):])
	if r.Method == "PUT" || r.Method == "POST" {
		handlePostJob(jobName, w, r)
	} else {
		handleGetJob(jobName, w)
	}
}

func handleGetJob(taskName string, w http.ResponseWriter) {
	task, err := ci.Proxy.GetTaskDetails(taskName)
	if err != nil {
		writeError(w, http.StatusNotFound, "Build task '%s' not found\n", taskName)
		return
	}
	marshalAndWrite(task, w)
}

func handlePostJob(taskName string, w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var project data.Project
	err := decoder.Decode(&project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error decoding project json: %s\n", err.Error())
		return
	}
	err = ci.Proxy.CreateTask(project)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error creating ci task: %s\n", err.Error())
		return
	}
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK")
}

func main() {
	var port = flag.String("port", "7777", "Define which TCP port to bind to")
	flag.Parse()

	http.Handle("/swagger/", http.StripPrefix("/swagger/", http.FileServer(http.Dir("swagger"))))
	http.HandleFunc("/v1/projects", getProjectsHandler)
	http.HandleFunc("/v1/environments", getEnvironmentsHandler)
	http.HandleFunc("/v1/deployments", getDeploymentsHandler)
	http.HandleFunc("/v1/containers", getContainersHandler)
	http.HandleFunc("/v1/templates", getTemplatesHandler)
	http.HandleFunc(JOB_PATH, getJobHandler)
	http.HandleFunc("/v1/jobs", getJobsHandler)
	http.HandleFunc(DISCOVERY_PATH, getDiscoveryHandler)
	http.HandleFunc("/healthz", healthzHandler)

	fmt.Printf("Starting Goobernet server on port %s\n", *port)
	http.ListenAndServe(":"+*port, nil)
}
