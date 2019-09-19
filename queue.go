package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	qJOBID			= iota
	qS 				= iota
//	qCCODE         	= iota
	qPAR  			= iota
	qEFFIC  		= iota
	qXFACTOR  		= iota
	qQ  			= iota
	qUSERNAME    	= iota
	qGROUP          = iota
	qMHOST 			= iota
	qPROCS    		= iota
//	qWALLTIME       = iota
//	qCOMPLETIONTIME	= iota
	qREMAINING		= iota
	qSTARTTIME		= iota
	qFIELDS 		= iota
)

// showq -c:

// JOBID               (15)+5
// S (1)+1
// CCODE         (9)+5
// PAR  (2)+3
// EFFIC  (2)+5
// XFACTOR  (2)+7
// Q  (2)+1
// USERNAME    (4)+8
// GROUP            (12)+5
// MHOST (1)+5
// PROCS    (4)+5
// WALLTIME        (8)+8
// COMPLETIONTIME

// showq -r:

// JOBID               
// S  
// PAR  
// EFFIC  
// XFACTOR  
// Q  
// USERNAME    
// GROUP            
// MHOST 
// PROCS   
// REMAINING            
// STARTTIME


const (
//	slurmLayout   = time.RFC3339
//	queueCommand  = "squeue -h -Ojobid,name,username,partition,numcpus,submittime,starttime,state -P"
	queueCommand  = "showq -r" // -r = active jobs, -c = completed jobs
	// nullStartTime = "N/A"
)

func (sc *TorqueCollector) collectQueue(ch chan<- prometheus.Metric) {
	log.Debugln("Collecting Queue metrics...")
	var collected uint
	var currentCommand string

	if len(sc.alreadyRegistered) > 0 {
		currentCommand = fmt.Sprintf(queueCommand+" | grep -v '%s' | uniq", strings.Join(sc.alreadyRegistered, "\\|"))
		sc.alreadyRegistered = make([]string, 0) // free memory
	} else {
		currentCommand = queueCommand
	}

	// execute the command
	log.Debugln(currentCommand)
	sshSession, err := sc.executeSSHCommand(currentCommand)
	if sshSession != nil {
		defer sshSession.Close()
	}
	if err != nil {
		if sshSession != nil {
			msg := err.Error()
			possibleExitCode := msg[len(msg)-1:]
			if possibleExitCode != "1" {
				log.Errorln(msg)
			} else {
				log.Debugln("No queued jobs collected")
				//TODO(emepetres) ¿¿supply metrics when no job data is available??
			}
			return
		}
		log.Errorln(err.Error())
		return
	}

	// wait for stdout to fill (it is being filled async by ssh)
	time.Sleep(100 * time.Millisecond)

// spiros start 
	// 	don't know how to return without header line and blank lines,
	// 	so jump over them this way for now
	// TODO: something about this
	// 	so, can probably look for the last item in the header line
	// 	and then do a ReadString('COMPLETIONTIME\n') type of thing...
	var buffer = sshSession.OutBuffer
	line, error := buffer.ReadString('\n')	// new line
	line, error = buffer.ReadString('\n')	// completed jobs-----
	line, error = buffer.ReadString('\n')	// new line
	line, error = buffer.ReadString('\n')	// header line...
	fmt.Println(line, error)
// spiros end

	lastJob := ""
	nextLine := nextLineIterator(sshSession.OutBuffer, squeueLineParser)
	for fields, err := nextLine(); err == nil; fields, err = nextLine() {
		// check the line is correctly parsed
		if err != nil {
			log.Warnln(err.Error())
			continue
		}

		// // parse submittime
		// submittime, stErr := time.Parse(slurmLayout, fields[qSUBMITTIME]+"Z")
		// if stErr != nil {
		// 	log.Warnln(stErr.Error())
		// 	continue
		// }

		if len(fields) < qFIELDS-1 {
			break
		}

		// parse and send job state
		status, statusOk := StatusDict[fields[qS]]
		if statusOk {
			if lastJob != fields[qJOBID] {
				ch <- prometheus.MustNewConstMetric(
					// sc.status,
					sc.queueRunning,
					prometheus.GaugeValue,
					float64(status),
//					fields[qJOBID], fields[qNAME], fields[qUSERNAME], fields[qPARTITION],
					// fields[qJOBID], fields[qUSERNAME], fields[qGROUP], fields[qMHOST],
					fields[qJOBID], 
					fields[qS], 
					fields[qUSERNAME], 
					fields[qREMAINING],
					fields[qSTARTTIME],
				)
				lastJob = fields[qJOBID]
				collected++
			}

		// 	// parse starttime and send wait time
		// 	if fields[qSTARTTIME] != nullStartTime {
		// 		starttime, sstErr := time.Parse(slurmLayout, fields[qSTARTTIME]+"Z")
		// 		if sstErr == nil {
		// 			waitTimestamp := starttime.Unix() - submittime.Unix()
		// 			ch <- prometheus.MustNewConstMetric(
		// 				sc.waitTime,
		// 				prometheus.GaugeValue,
		// 				float64(waitTimestamp),
		// 				fields[qJOBID], fields[qNAME], fields[qUSERNAME],
		// 				fields[qPARTITION], fields[qNUMCPUS], fields[qSTATE],
		// 			)
		// 		} else {
		// 			log.Warn(sstErr.Error())
		// 		}
		// 	}
		// } else {
		// 	log.Warnf("Couldn't parse job status: %s", fields[qSTATE])
		}
	}
	log.Infof("%d queued jobs collected", collected)
}

func squeueLineParser(line string) []string {

	// TODO: this works well for all fields except time which has spaces within the field
	result := strings.Fields(line)	
	return result
}
