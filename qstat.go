
package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

const (
	aJOBID     = iota
	aNAME      = iota
	aUSERNAME  = iota
	aPARTITION = iota
	aSTATE     = iota
	aFIELDS    = iota
)

const (
	// acctCommand = "sacct -n -a -X -o \"JobIDRaw,JobName%%20,User%%20,Partition%%20,State%%20\" -S%02d:%02d:%02d -sBF,CA,CD,CF,F,NF,PR,RS,S,TO | grep -v 'PENDING\\|COMPLETING\\|RUNNING' | uniq"
	qstatCommand = "qstat -u xeuspimi"
)

func (sc *TorqueCollector) collectQstat(ch chan<- prometheus.Metric) {
	log.Debugln("Collecting qstat metrics...")
	var collected uint

	hour := sc.lasttime.Hour()
	minute := sc.lasttime.Minute()
	second := sc.lasttime.Second()

	now := time.Now().In(sc.timeZone)
	if now.Hour() < hour {
		hour = 0
		minute = 0
		second = 0
	}

	// currentCommand := fmt.Sprintf(qstatCommand, hour, minute, second)
	currentCommand = qstatCommand
	log.Debugln(currentCommand)

	sshSession, err := sc.executeSSHCommand(currentCommand)
	if sshSession != nil {
		defer sshSession.Close()
	}
	if err != nil {
		log.Errorf("qstat: %s", err.Error())
		return
	}

	// wait for stdout to fill (it is being filled async by ssh)
	time.Sleep(100 * time.Millisecond)
	sc.setLastTime()

	nextLine := nextLineIterator(sshSession.OutBuffer, qstatLineParser)
	for fields, err := nextLine(); err == nil; fields, err = nextLine() {
		// check the line is correctly parsed
		if err != nil {
			log.Warnln(err.Error())
			continue
		}

		// parse and send job state
		status, statusOk := StatusDict[fields[aSTATE]]
		if statusOk {
			if jobIsNotInQueue(status) {
				ch <- prometheus.MustNewConstMetric(
					sc.status,
					prometheus.GaugeValue,
					float64(status),
					fields[aJOBID], fields[aNAME], fields[aUSERNAME], fields[aPARTITION],
				)
				sc.alreadyRegistered = append(sc.alreadyRegistered, fields[aJOBID])
				//log.Debugln("Job " + fields[aJOBID] + " finished with state " + fields[aSTATE])
				collected++
			}
		} else {
			log.Warnf("Couldn't parse job status: '%s', fields '%s'", fields[aSTATE], strings.Join(fields, "|"))
		}
	}

	log.Infof("%d finished jobs collected", collected)
}

func jobIsNotInQueue(state int) bool {
	// return state != sPENDING && state != sRUNNING && state != sCOMPLETING
	return state != sEXITING && state != sQUEUED && state != sRUNNING
}

func qstatLineParser(line string) []string {
	fields := strings.Fields(line)

	if len(fields) < aFIELDS {
		log.Warnf("qstat line parse failed (%s): %d fields expected, %d parsed", line, aFIELDS, len(fields))
		return nil
	}

	return fields
}
