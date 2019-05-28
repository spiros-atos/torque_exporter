// Copyright (c) 2017 MSO4SC - javier.carnero@atos.net
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Queue Slurm collector

package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	// qJOBID      = iota
	// qNAME       = iota
	// qUSERNAME   = iota
	// qPARTITION  = iota
	// qNUMCPUS    = iota
	// qSUBMITTIME = iota
	// qSTARTTIME  = iota
	// qSTATE      = iota
	// qFIELDS     = iota
	qJOBID			= iota
	qS 				= iota
	qCCODE         	= iota
	qPAR  			= iota
	qEFFIC  		= iota
	qXFACTOR  		= iota
	qQ  			= iota
	qUSERNAME    	= iota
	qGROUP          = iota
	qMHOST 			= iota
	qPROCS    		= iota
	qWALLTIME       = iota
	qCOMPLETIONTIME	= iota
	qFIELDS 		= iota
)

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


const (
	slurmLayout   = time.RFC3339
//	queueCommand  = "squeue -h -Ojobid,name,username,partition,numcpus,submittime,starttime,state -P"
	queueCommand  = "showq -c"
	nullStartTime = "N/A"
)

func (sc *SlurmCollector) collectQueue(ch chan<- prometheus.Metric) {
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
	var buffer = sshSession.OutBuffer
	line, error := buffer.ReadString('\n')	// new line
	line, error = buffer.ReadString('\n')	// completed jobs-----
	line, error = buffer.ReadString('\n')	// new line
	line, error = buffer.ReadString('\n')	// header line...
	fmt.Println(line, error)
	// so, can probably look for the last item in the header line
	// and then do a ReadString('COMPLETIONTIME\n') type of thing...
// spiros end

	// remove already registered map memory from sacct when finished
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

		// // parse and send job state
		// status, statusOk := StatusDict[fields[qSTATE]]
		// if statusOk {
		// 	if lastJob != fields[qJOBID] {
		// 		ch <- prometheus.MustNewConstMetric(
		// 			sc.status,
		// 			prometheus.GaugeValue,
		// 			float64(status),
		// 			fields[qJOBID], fields[qNAME], fields[qUSERNAME], fields[qPARTITION],
		// 		)
		// 		lastJob = fields[qJOBID]
		// 		collected++
		// 	}

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
		// }
	}
	log.Infof("%d queued jobs collected", collected)
}

// TODO(emepetres): can be optimised doing at the same time Trim+alloc
func squeueLineParser(line string) []string {
	// check if line is long enough
	fields := [13]string{"JOBID", "S", "CCODE", "PAR", "EFFIC", "XFACTOR", "Q", "USERNAME", "GROUP", "MHOST", "PROCS", "WALLTIME", "COMPLETIONTIME"}
	nchars := [13]int{20, 2, 14, 5, 7, 9, 3, 12, 17, 6, 9, 16, 14}
	count := 0
	for idx, nc := range nchars {
		count += nc
	}
	fmt.Println(count)

	// if len(line) < 20*(qFIELDS-1)+1 {
	// 	log.Warnln("Slurm line not long enough: \"" + line + "\"")
	// 	return nil
	// }
	if len(line) < count {
		log.Warnln("Torque line not long enough: \"" + line + "\"")
		return nil
	}

	// separate fields by 20 chars, trimming them
	result := make([]string, 0, qFIELDS)
	for i := 0; i < qFIELDS-1; i++ {
		result = append(result, strings.TrimSpace(line[20*i:20*(i+1)]))
	}
	result = append(result, strings.TrimSpace(line[20*(qFIELDS-1):]))

	// add + to the end of the name if it is long enough
	if len(result[qNAME]) == 20 {
		result[qNAME] = result[qNAME][:19] + "+"
	}

	return result
}
