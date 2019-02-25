/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Note: the example only works with the code within the same release/branch.
// This is based on the example from the official k8s golang client repository:
// k8s.io/client-go/examples/create-update-delete-deployment/
package mmforc

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/open-match/config"
	"github.com/GoogleCloudPlatform/open-match/internal/logging"
	"github.com/GoogleCloudPlatform/open-match/internal/metrics"
	redisHelpers "github.com/GoogleCloudPlatform/open-match/internal/statestorage/redis"
	"github.com/tidwall/gjson"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/gomodule/redigo/redis"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	batchv1 "k8s.io/api/batch/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	//"k8s.io/kubernetes/pkg/api"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	// Uncomment the following line to load the gcp plugin (only required to authenticate against GKE clusters).
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	// Logrus structured logging setup
	mmforcLogFields = log.Fields{
		"app":       "openmatch",
		"component": "mmforc",
	}
	mmforcLog = log.WithFields(mmforcLogFields)

	// Default kubernetes namespace
	namespace = apiv1.NamespaceDefault

	// Viper config management setup
	cfg = viper.New()
	err = errors.New("")
)

func initializeApplication() {
	// Add a hook to the logger to auto-count log lines for metrics output thru OpenCensus
	log.AddHook(metrics.NewHook(MmforcLogLines, KeySeverity))

	// Viper config management initialization
	cfg, err = config.Read()
	if err != nil {
		mmforcLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to load config file")
	}

	// Configure open match logging defaults
	logging.ConfigureLogging(cfg)

	metaNamespace := os.Getenv("METADATA_NAMESPACE")
	if len(metaNamespace) != 0 {
		namespace = metaNamespace
	}

	// Configure OpenCensus exporter to Prometheus
	// metrics.ConfigureOpenCensusPrometheusExporter expects that every OpenCensus view you
	// want to register is in an array, so append any views you want from other
	// packages to a single array here.
	ocMmforcViews := DefaultMmforcViews // mmforc OpenCensus views.
	// Waiting on https://github.com/opencensus-integrations/redigo/pull/1
	// ocMmforcViews = append(ocMmforcViews, redis.ObservabilityMetricViews...) // redis OpenCensus views.
	mmforcLog.WithFields(log.Fields{"viewscount": len(ocMmforcViews)}).Info("Loaded OpenCensus views")
	metrics.ConfigureOpenCensusPrometheusExporter(cfg, ocMmforcViews)

}

// RunApplication is a hook for the main() method in the main executable.
func RunApplication() {
	initializeApplication()

	pool := redisHelpers.ConnectionPool(cfg)
	redisConn := pool.Get()
	defer redisConn.Close()

	// Get k8s credentials so we can starts k8s Jobs
	mmforcLog.Info("Attempting to acquire k8s credentials")
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	mmforcLog.Info("K8s credentials acquired")

	start := time.Now()
	checkProposals := true

	// main loop; kick off matchmaker functions for profiles in the profile
	// queue and an evaluator when proposals are in the proposals queue
	for {
		ctx, cancel := context.WithCancel(context.Background())
		_ = cancel

		// Get profiles and kick off a job for each
		mmforcLog.WithFields(log.Fields{
			"profileQueueName": cfg.GetString("queues.profiles.name"),
			"pullCount":        cfg.GetInt("queues.profiles.pullCount"),
			"query":            "SPOP",
			"component":        "statestorage",
		}).Debug("Retreiving match profiles")

		results, err := redis.Strings(redisConn.Do("SPOP",
			cfg.GetString("queues.profiles.name"), cfg.GetInt("queues.profiles.pullCount")))
		if err != nil {
			panic(err)
		}

		if len(results) > 0 {
			mmforcLog.WithFields(log.Fields{
				"numProfiles": len(results),
			}).Info("Starting MMF jobs...")

			for _, profile := range results {
				// Kick off the job asynchrnously
				go mmfunc(ctx, profile, cfg, clientset, pool)
				// Count the number of jobs running
				redisHelpers.Increment(context.Background(), pool, "concurrentMMFs")
			}
		} else {
			mmforcLog.WithFields(log.Fields{
				"profileQueueName": cfg.GetString("queues.profiles.name"),
			}).Info("Unable to retreive match profiles from statestorage - have you entered any?")
		}

		// Check to see if we should run the evaluator.
		// Get number of running MMFs
		r, err := redisHelpers.Retrieve(context.Background(), pool, "concurrentMMFs")

		if err != nil {
			if err.Error() == "redigo: nil returned" {
				// No MMFs have run since we last evaluated; reset timer and loop
				mmforcLog.Debug("Number of concurrentMMFs is nil")
				start = time.Now()
				time.Sleep(1000 * time.Millisecond)
			}
			continue
		}
		numRunning, err := strconv.Atoi(r)
		if err != nil {
			mmforcLog.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Issue retrieving number of currently running MMFs")
		}

		// We are ready to evaluate either when all MMFs are complete, or the
		// timeout is reached.
		//
		// Tuning how frequently the evaluator runs is a complex topic and
		// probably only of interest to users running large-scale production
		// workloads with many concurrently running matchmaking functions,
		// which have some overlap in the matchmaking player pools. Suffice to
		// say that under load, this switch should almost always trigger the
		// timeout interval code path.  The concurrentMMFs check to see how
		// many are still running is meant as a deadman's switch to prevent
		// waiting to run the evaluator when all your MMFs are already
		// finished.
		switch {
		case time.Since(start).Seconds() >= float64(cfg.GetInt("evaluator.interval")):
			mmforcLog.WithFields(log.Fields{
				"interval": cfg.GetInt("evaluator.interval"),
			}).Info("Maximum evaluator interval exceeded")
			checkProposals = true

			// Opencensus tagging
			ctx, _ = tag.New(ctx, tag.Insert(KeyEvalReason, "interval_exceeded"))
		case numRunning <= 0:
			mmforcLog.Info("All MMFs complete")
			checkProposals = true
			numRunning = 0
			ctx, _ = tag.New(ctx, tag.Insert(KeyEvalReason, "mmfs_completed"))
		}

		if checkProposals {
			// Make sure there are proposals in the queue. No need to run the
			// evaluator if there are none.
			checkProposals = false
			mmforcLog.Info("Checking statestorage for match object proposals")
			results, err := redisHelpers.Count(context.Background(), pool, cfg.GetString("queues.proposals.name"))
			switch {
			case err != nil:
				mmforcLog.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Couldn't retrieve the length of the proposal queue from statestorage!")
			case results == 0:
				mmforcLog.WithFields(log.Fields{}).Warn("No proposals in the queue!")
			default:
				mmforcLog.WithFields(log.Fields{
					"numProposals": results,
				}).Info("Proposals available, evaluating!")
				go evaluator(ctx, cfg, clientset)
			}
			err = redisHelpers.Delete(context.Background(), pool, "concurrentMMFs")
			if err != nil {
				mmforcLog.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Error deleting concurrent MMF counter!")
			}
			start = time.Now()
		}

		// TODO: Make this tunable via config.
		// A sleep here is not critical but just a useful safety valve in case
		// things are broken, to keep the main loop from going all-out and spamming the log.
		mainSleep := 1000
		mmforcLog.WithFields(log.Fields{
			"ms": mainSleep,
		}).Info("Sleeping...")
		time.Sleep(time.Duration(mainSleep) * time.Millisecond)
	} // End main for loop
}

// mmfunc generates a k8s job that runs the specified mmf container image.
// resultsID is the redis key that the Backend API is monitoring for results; we can 'short circuit' and write errors directly to this key if we can't run the MMF for some reason.
func mmfunc(ctx context.Context, resultsID string, cfg *viper.Viper, clientset *kubernetes.Clientset, pool *redis.Pool) {

	// Generate the various keys/names, some of which must be populated to the k8s job.
	imageName := cfg.GetString("defaultImages.mmf.name") + ":" + cfg.GetString("defaultImages.mmf.tag")
	jobType := "mmf"
	ids := strings.Split(resultsID, ".") // comes in as dot-concatinated moID and profID.
	moID := ids[0]
	profID := ids[1]
	timestamp := strconv.Itoa(int(time.Now().Unix()))
	jobName := timestamp + "." + moID + "." + profID + "." + jobType
	propID := "proposal." + timestamp + "." + moID + "." + profID

	// Extra fields for structured logging
	lf := log.Fields{"jobName": jobName}
	if cfg.GetBool("debug") { // Log a lot more info.
		lf = log.Fields{
			"jobType":             jobType,
			"backendMatchObject":  moID,
			"profile":             profID,
			"jobTimestamp":        timestamp,
			"jobName":             jobName,
			"profileImageJSONKey": cfg.GetString("jsonkeys.mmfImage"),
		}
	}
	mmfuncLog := mmforcLog.WithFields(lf)

	// Read the full profile from redis and access any keys that are important to deciding how MMFs are run.
	// TODO: convert this to using redispb and directly access the protobuf message instead of retrieving as a map?
	profile, err := redisHelpers.RetrieveAll(ctx, pool, profID)
	if err != nil {
		// Log failure to read this profile and return - won't run an MMF for an unreadable profile.
		mmfuncLog.WithFields(log.Fields{"error": err.Error()}).Error("Failure retreiving profile from statestorage")
		return
	}

	// Got profile from state storage, make sure it is valid
	if !gjson.Valid(profile["properties"]) {
		mmforcLog.WithFields(log.Fields{
			"jobName": jobName,
		}).Warn("Profile JSON was invalid")
		return
	}

	// Determine what kind of job orchestration the profile requires
	profileHostName := gjson.Get(profile["properties"], cfg.GetString("jsonkeys.mmfHostName"))
	profileImage := gjson.Get(profile["properties"], cfg.GetString("jsonkeys.mmfImage"))

	// If a hostname was provided, try making a restful POST to the existing endpoint
	if profileHostName.Exists() && len(profileHostName.String()) > 0 {
		port := "80"
		profilePort := gjson.Get(profile["properties"], cfg.GetString("jsonkeys.mmfPort"))
		if profilePort.Exists() {
			port = profilePort.String()
		} else {
			mmfuncLog.Debug("No port specified in configured properties json key, using default port instead")
		}

		mmforcLog.WithFields(log.Fields{
			"jobName":  jobName,
			"hostName": profileHostName,
			"port":     port,
		}).Debug("Profile specifies a host name for running the match function as a POST rest service call")

		// Make a rest service call
		err = callRestFunction(profileHostName.String(), port, jobName, profID, moID, propID, resultsID, timestamp)

	} else {
		// Otherwise, if a profile image is available, use a k8s job
		if profileImage.Exists() && len(profileImage.String()) > 0 {
			imageName = profileImage.String()
		} else {
			mmfuncLog.Warn("Failed to read image name from profile at configured json key, using default image instead")
		}

		mmfuncLog = mmfuncLog.WithFields(log.Fields{"containerImage": imageName})
		mmfuncLog.Info("Attempting to create mmf k8s job")

		// Kick off k8s job
		envvars := []apiv1.EnvVar{
			{Name: "MMF_PROFILE_ID", Value: profID},
			{Name: "MMF_PROPOSAL_ID", Value: propID},
			{Name: "MMF_REQUEST_ID", Value: moID},
			{Name: "MMF_ERROR_ID", Value: resultsID},
			{Name: "MMF_TIMESTAMP", Value: timestamp},
			// Deprecated: 0.1.0 compatibility config vars.
			{Name: "DEBUG", Value: cfg.GetString("debug")},
			{Name: "JSONKEYS_ROSTERS", Value: cfg.GetString("jsonkeys.rosters")},
			{Name: "JSONKEYS_MMFIMAGE", Value: cfg.GetString("jsonkeys.mmfImage")},
			{Name: "JSONKEYS_POOLS", Value: cfg.GetString("jsonkeys.pools")},
		}
		err = submitJob(clientset, jobType, jobName, imageName, envvars)
	}

	if err != nil {
		// Record failure & log
		stats.Record(ctx, mmforcMmfFailures.M(1))
		mmfuncLog.WithFields(log.Fields{"error": err.Error()}).Error("MMF submission failure!")
	} else {
		// Record Success
		stats.Record(ctx, mmforcMmfs.M(1))
	}
}

// callRestFunction will lookup the provided hostname on the network, then execute a POST to the http /api/function endpoint hosted there
// This method uses a non-optimized, synchronous, on-demand creation of the http client
// Historically, this is a prototype for enabling knative match functions which temporarily requires http/1.1 communication
func callRestFunction(hostName string, strPort string, jobName string, profID string, moID string, propID string, resultsID string, timestamp string) error {
	// TODO: Better define this service contract in an official capacity
	type Profile struct {
		JobName   string
		ProfId    string
		MoId      string
		PropId    string
		ResultsId string
		Timestamp string
	}

	profile := &Profile{
		JobName:   jobName,
		ProfId:    profID,
		MoId:      moID,
		PropId:    propID,
		ResultsId: resultsID,
		Timestamp: timestamp,
	}
	b, err := json.Marshal(profile)
	if err != nil {
		mmforcLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Unable to marshal the profile into json for the MMF rest job call")
		return err
	}
	body := strings.NewReader(string(b))

	// TODO: This is designed to include service discovery from within this network (kubernetes or internal dns)
	// How this is constructed and discovered should be more configurable by the external scheduling mechanism
	host, err := net.LookupHost(hostName)
	if err != nil {
		return err
	}

	// TODO: Re-use a pool'd cache of host-specific http clients to save on creation cost every cycle
	// TODO: Make the endpoint itself configurable to the specific request being produced by the external scheduling mechanism
	// TODO: Configurable timeout and canceling (in-step with the evalutor cycling)
	resp, err := http.Post("http://"+host[0]+":"+strPort+"/api/function", "application/json", body)
	if err != nil {
		// Don't panic, the process is fine, the match function is just erroring
		mmforcLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("MMF rest job call failure!")
		return err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// The function returned an erroring
		mmforcLog.WithFields(log.Fields{
			"status":     resp.StatusCode,
			"host":       resp.Request.Host,
			"requestURI": resp.Request.RequestURI,
		}).Error("MMF rest job call failure!")
	}

	return nil
}

// evaluator generates a k8s job that runs the specified evaluator container image.
func evaluator(ctx context.Context, cfg *viper.Viper, clientset *kubernetes.Clientset) {

	imageName := cfg.GetString("defaultImages.evaluator.name") + ":" + cfg.GetString("defaultImages.evaluator.tag")
	// Generate the job name
	timestamp := strconv.Itoa(int(time.Now().Unix()))
	jobType := "evaluator"
	jobName := timestamp + "." + jobType

	mmforcLog.WithFields(log.Fields{
		"jobName":        jobName,
		"containerImage": imageName,
	}).Info("Attempting to create evaluator k8s job")

	// Kick off k8s job
	envvars := []apiv1.EnvVar{{Name: "MMF_TIMESTAMP", Value: timestamp}}
	err = submitJob(clientset, jobType, jobName, imageName, envvars)
	if err != nil {
		// Record failure & log
		stats.Record(ctx, mmforcEvalFailures.M(1))
		mmforcLog.WithFields(log.Fields{
			"error":          err.Error(),
			"jobName":        jobName,
			"containerImage": imageName,
		}).Error("Evaluator job submission failure!")
	} else {
		// Record success
		stats.Record(ctx, mmforcEvals.M(1))
	}
}

// submitJob submits a job to kubernetes
func submitJob(clientset *kubernetes.Clientset, jobType string, jobName string, imageName string, envvars []apiv1.EnvVar) error {

	// DEPRECATED: will be removed in a future vrsion.  Please switch to using the 'MMF_*' environment variables.
	v := strings.Split(jobName, ".")
	envvars = append(envvars, apiv1.EnvVar{Name: "PROFILE", Value: strings.Join(v[:len(v)-1], ".")})

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: jobName,
		},
		Spec: batchv1.JobSpec{
			Completions: int32Ptr(1),
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": jobType,
					},
					Annotations: map[string]string{
						// Unused; here as an example.
						// Later we can put things more complicated than
						// env vars here and read them using k8s downward API
						// volumes
						"profile": jobName,
					},
				},
				Spec: apiv1.PodSpec{
					RestartPolicy: "Never",
					Containers: []apiv1.Container{
						{
							Name:            jobType,
							Image:           imageName,
							ImagePullPolicy: "Always",
							Env:             envvars,
						},
					},
				},
			},
		},
	}

	// Submit kubernetes job
	jobsClient := clientset.BatchV1().Jobs(namespace)
	result, err := jobsClient.Create(job)
	if err != nil {
		// TODO: replace queued profiles if things go south
		mmforcLog.WithFields(log.Fields{
			"error": err.Error(),
		}).Error("Couldn't create k8s job!")
	}

	mmforcLog.WithFields(log.Fields{
		"jobName": result.GetObjectMeta().GetName(),
	}).Info("Created job.")

	return err
}

// readability functions used by generateJobSpec
func int32Ptr(i int32) *int32 { return &i }
func strPtr(i string) *string { return &i }
