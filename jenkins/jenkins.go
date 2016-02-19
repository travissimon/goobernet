package jenkins

import (
	"bytes"
	"fmt"
	"text/template"
	"time"

	"github.com/bndr/gojenkins"
	"github.com/travissimon/goobernet/data"
)

type JenkinsJob struct {
	Name  string `json:"name"`
	Url   string `json:"url"`
	Color string `json:"color"`
}

type JenkinsJobDetails struct {
	Name        string       `json:"name"`
	Url         string       `json:"url"`
	Description string       `json:"description"`
	LastBuild   JenkinsBuild `json:"lastBuild"`
	Downstream  []JenkinsJob `json:"downstreamBuilds"`
}

type JenkinsBuild struct {
	Number    int64     `json:"buildNumber"`
	Duration  int64     `json:"duration"`
	Result    string    `json:"result"`
	Timestamp time.Time `json:"timestamp"`
	Url       string    `json:"url"`
	IsGood    bool      `json:isGood`
}

var client *gojenkins.Jenkins

func init() {
	var cfg = data.GetConfig()
	fmt.Printf("Connecting to Jenkins instance: %s\n", cfg.JenkinsUrl)
	j, err := gojenkins.CreateJenkins(cfg.JenkinsUrl, cfg.JenkinsUsername, cfg.JenkinsPassword).Init()

	if err != nil {
		panic("Could not connect to Jenkins server")
	}

	client = j
}

func GetAllJobNames() ([]JenkinsJob, error) {
	jobNames, err := client.GetAllJobNames()
	if err != nil {
		return nil, err
	}

	var jobs []JenkinsJob
	jobs = make([]JenkinsJob, 0, len(jobNames))

	for _, j := range jobNames {
		jobs = append(jobs, JenkinsJob{j.Name, j.Url, j.Color})
	}

	return jobs, nil
}

func GetJobDetails(jobName string) (*JenkinsJobDetails, error) {
	job, err := client.GetJob(jobName)
	if err != nil {
		return nil, err
	}

	downStr, _ := job.GetDownstreamJobs()
	var jobs []JenkinsJob
	jobs = make([]JenkinsJob, 0, len(downStr))

	for _, j := range downStr {
		jobs = append(jobs, JenkinsJob{j.GetName(), j.Raw.URL, j.Raw.Color})
	}

	lb, _ := job.GetLastBuild()
	lastBuild := JenkinsBuild{
		Number:    lb.GetBuildNumber(),
		Duration:  lb.GetDuration(),
		Result:    lb.GetResult(),
		Timestamp: lb.GetTimestamp(),
		Url:       lb.GetUrl(),
		IsGood:    lb.IsGood(),
	}

	return &JenkinsJobDetails{
		Name:        job.GetName(),
		Url:         job.Raw.URL,
		Description: job.GetDescription(),
		LastBuild:   lastBuild,
		Downstream:  jobs,
	}, nil
}

func CreateJob(newProject data.Project) error {
	var templ data.JenkinsTemplate
	var err error

	templ = newProject.BuildTemplate
	/*
		if templ, err = data.GetTemplateByName(newProject.BuildTemplate); err != nil {
			return err
		}
	*/

	var t *template.Template
	if t, err = template.New("Build template").Parse(templ.Content); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	err = t.Execute(buf, newProject)
	if err != nil {
		return err
	}

	xml := buf.String()
	// fmt.Printf("xml: \n%s\n", xml)

	if _, err := client.CreateJob(xml, newProject.ShortName); err != nil {
		return err
	}

	if err := data.AddProject(newProject); err != nil {
		return err
	}

	return nil
}
