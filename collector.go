package main

import (
	"bytes"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/spiros-atos/torque_exporter/ssh"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	sCOMPLETED	= iota
	sEXITING	= iota
	sHELD		= iota
	sQUEUED		= iota
	sRUNNING	= iota
	sMOVING		= iota
	sWAITING	= iota
	sSUSPENDED	= iota
)

/*
from man qstat:
 -  the job state:
       C -     Job is completed after having run/
       E -  Job is exiting after having run.
       H -  Job is held.
       Q -  job is queued, eligible to run or routed.
       R -  job is running.
       T -  job is being moved to new location.
       W -  job is waiting for its execution time
            (-a option) to be reached.
       S -  (Unicos only) job is suspend.
*/

// StatusDict maps string status with its int values
var StatusDict = map[string]int{
	"C":    sCOMPLETED,
	"E":	sEXITING,
	"H":	sHELD,
	"Q":	sQUEUED,
	"R":    sRUNNING,
	"T":    sMOVING,
	"W":    sWAITING,
	"S":    sSUSPENDED,
}

type TorqueCollector struct {
	// waitTime          *prometheus.Desc
	// status            *prometheus.Desc
	queueRunning      *prometheus.Desc
	userJobs          *prometheus.Desc
	// jobDetails        *prometheus.Desc
	partitionNodes    *prometheus.Desc
	sshConfig         *ssh.SSHConfig
	sshClient         *ssh.SSHClient
	timeZone          *time.Location
	alreadyRegistered []string
	lasttime          time.Time
}

// func NewTorqueCollector(host, sshUser, sshPass, timeZone string) *TorqueCollector {
// 	newTorqueCollector := &TorqueCollector{
// 		waitTime: prometheus.NewDesc(
// 			"job_wait_time",
// 			"Time that the job waited, or is estimated to wait",
// 			[]string{"jobid", "name", "username", "partition", "numcpus", "state"},
// 			nil,
// 		),
// 		status: prometheus.NewDesc(
// 			"job_status",
// 			"Status of the job",
// 			[]string{"jobid", "name", "username", "partition"},
// 			nil,
// 		),
// 		partitionNodes: prometheus.NewDesc(
// 			"partition_nodes",
// 			"Nodes of the partition",
// 			[]string{"partition", "availability", "state"},
// 			nil,
// 		),
// 		sshConfig: ssh.NewSSHConfigByPassword(
// 			sshUser,
// 			sshPass,
// 			host,
// 			22,
// 		),
// 		sshClient:         nil,
// 		alreadyRegistered: make([]string, 0),
// 	}
// 	var err error
// 	newTorqueCollector.timeZone, err = time.LoadLocation(timeZone)
// 	if err != nil {
// 		log.Fatalln(err.Error())
// 	}
// 	newTorqueCollector.setLastTime()
// 	return newTorqueCollector
// }

func NewerTorqueCollector(host, sshUser, sshPass, timeZone string) *TorqueCollector {
	newerTorqueCollector := &TorqueCollector{
		queueRunning: prometheus.NewDesc(
			"te_showq_r",
			"torque's queue",
			[]string{"jobid", "state", "username", "remaining", "starttime"},
			nil,
		),
		userJobs: prometheus.NewDesc(
			"te_qstat_u",
			"user's jobs",
			[]string{"jobid", "username", "jobname", "status"},
			nil,
		),
		// jobDetails: prometheus.NewDesc(
		// 	"te_qstat_f",
		// 	"job details",
		// 	[]string{"job_name", 
		// 			 "job_owner", 
		// 			 "job_state",
		// 			 "ctime",
		// 			 "mtime",
		// 			 "output_path",
		// 			 "qtime",
		// 			 "euser",
		// 			 "queue_type",
		// 			 "etime",
		// 			 "submit_args"},
		// 	nil,
		// ),
		partitionNodes: prometheus.NewDesc(
			"te_qstat_f",
			"job details",
			[]string{"partition", "availability", "state"},
			nil,
		),
		sshConfig: ssh.NewSSHConfigByPassword(
			sshUser,
			sshPass,
			host,
			22,
		),
		sshClient:         nil,
		alreadyRegistered: make([]string, 0),
	}
	var err error
	newerTorqueCollector.timeZone, err = time.LoadLocation(timeZone)
	if err != nil {
		log.Fatalln(err.Error())
	}
	newerTorqueCollector.setLastTime()
	return newerTorqueCollector
}

// Describe sends metrics descriptions of this collector
// through the ch channel.
// It implements collector interface
func (sc *TorqueCollector) Describe(ch chan<- *prometheus.Desc) {
	// ch <- sc.waitTime
	// ch <- sc.status
	ch <- sc.queueRunning
	ch <- sc.userJobs
	// ch <- sc.jobDetails
	ch <- sc.partitionNodes
}

// Collect read the values of the metrics and
// passes them to the ch channel.
// It implements collector interface
func (sc *TorqueCollector) Collect(ch chan<- prometheus.Metric) {
	var err error
	sc.sshClient, err = sc.sshConfig.NewClient()
	if err != nil {
		log.Errorf("Creating SSH client: %s", err.Error())
		return
	}

	sc.collectQstat(ch)
	sc.collectQueue(ch)
	// sc.collectInfo(ch)

	err = sc.sshClient.Close()
	if err != nil {
		log.Errorf("Closing SSH client: %s", err.Error())
	}
}

func (sc *TorqueCollector) executeSSHCommand(cmd string) (*ssh.SSHSession, error) {
	command := &ssh.SSHCommand{
		Path: cmd,
		// Env:    []string{"LC_DIR=/usr"},
	}

	var outb, errb bytes.Buffer
	session, err := sc.sshClient.OpenSession(nil, &outb, &errb)
	if err == nil {
		err = session.RunCommand(command)
		return session, err
	}
	return nil, err
}

func (sc *TorqueCollector) setLastTime() {
	sc.lasttime = time.Now().In(sc.timeZone).Add(-1 * time.Minute)
}

func parseTorqueTime(field string) (uint64, error) {
	var days, hours, minutes, seconds uint64
	var err error

	toParse := field
	haveDays := false

	// get days
	slice := strings.Split(toParse, "-")
	if len(slice) == 1 {
		toParse = slice[0]
	} else if len(slice) == 2 {
		days, err = strconv.ParseUint(slice[0], 10, 64)
		if err != nil {
			return 0, err
		}
		toParse = slice[1]
		haveDays = true
	} else {
		err = errors.New("Torque time could not be parsed: " + field)
		return 0, err
	}

	// get hours, minutes and seconds
	slice = strings.Split(toParse, ":")
	if len(slice) == 3 {
		hours, err = strconv.ParseUint(slice[0], 10, 64)
		if err == nil {
			minutes, err = strconv.ParseUint(slice[1], 10, 64)
			if err == nil {
				seconds, err = strconv.ParseUint(slice[1], 10, 64)
			}
		}
		if err != nil {
			return 0, err
		}
	} else if len(slice) == 2 {
		if haveDays {
			hours, err = strconv.ParseUint(slice[0], 10, 64)
			if err == nil {
				minutes, err = strconv.ParseUint(slice[1], 10, 64)
			}
		} else {
			minutes, err = strconv.ParseUint(slice[0], 10, 64)
			if err == nil {
				seconds, err = strconv.ParseUint(slice[1], 10, 64)
			}
		}
		if err != nil {
			return 0, err
		}
	} else if len(slice) == 1 {
		if haveDays {
			hours, err = strconv.ParseUint(slice[0], 10, 64)
		} else {
			minutes, err = strconv.ParseUint(slice[0], 10, 64)
		}
		if err != nil {
			return 0, err
		}
	} else {
		err = errors.New("Torque time could not be parsed: " + field)
		return 0, err
	}

	return days*24*60*60 + hours*60*60 + minutes*60 + seconds, nil
}

// nextLineIterator returns a function that iterates
// over an io.Reader object returning each line  parsed
// in fields following the parser method passed as argument
func nextLineIterator(buf io.Reader, parser func(string) []string) func() ([]string, error) {
	var buffer = buf.(*bytes.Buffer)
	var parse = parser
	return func() ([]string, error) {
		// get next line in buffer
		line, err := buffer.ReadString('\n')
		if err != nil {
			return nil, err
		}
		// fmt.Print(line)

		// parse the line and return
		parsed := parse(line)
		if parsed == nil {
			return nil, errors.New("not able to parse line")
		}
		return parsed, nil
	}
}
