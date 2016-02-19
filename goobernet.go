package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/travissimon/goobernet/data"
	"github.com/travissimon/goobernet/docker"
	"github.com/travissimon/goobernet/jenkins"
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

func writeError(w http.ResponseWriter, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, msg)
	w.WriteHeader(400)
	w.Write([]byte(msg))
}

func marshalAndWrite(obj interface{}, w http.ResponseWriter) {
	json, err := json.Marshal(obj)
	if err != nil {
		writeError(w, "Error marshalling object: %s\n", err)
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
		writeError(w, "Error retrieving docker containers: %s\n", err)
		return
	}
	marshalAndWrite(containers, w)
}

func getDiscoveryHandler(w http.ResponseWriter, r *http.Request) {
	environment := string(r.URL.Path[len(DISCOVERY_PATH):])
	deployments, err := data.GetDeploymentsByEnvironmentName(environment)
	if err != nil {
		writeError(w, "Error with discovery: %s\n", err.Error())
		return
	}
	marshalAndWrite(deployments, w)
}

func getJobsHandler(w http.ResponseWriter, r *http.Request) {
	jobs, err := jenkins.GetAllJobNames()
	if err != nil {
		writeError(w, "Error querying Jenkins: %s\n", err.Error())
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

func handleGetJob(jobName string, w http.ResponseWriter) {
	job, err := jenkins.GetJobDetails(jobName)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "Jenkins job '%s' not found\n", jobName)
		return
	}
	fmt.Fprintf(os.Stderr, "Apparently we escaped error trap")
	marshalAndWrite(job, w)
}

func handlePostJob(jobName string, w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var project data.Project
	err := decoder.Decode(&project)
	if err != nil {
		writeError(w, "Error decoding project json: %s\n", err.Error())
		return
	}
	err = jenkins.CreateJob(project)
	if err != nil {
		writeError(w, "Error creating Jenkins job: %s\n", err.Error())
		return
	}
	w.WriteHeader(200)
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
