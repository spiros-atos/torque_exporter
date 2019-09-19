
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
	aUSERNAME  = iota
	aQUEUE     = iota
	aJOBNAME   = iota
	aSESSID    = iota
	aNDS       = iota
	aTSK       = iota
	aREQDMEM   = iota
	aREQDTIME  = iota
	aS         = iota
	aELAPTIME  = iota
	aFIELDS    = iota
)

const (
	// acctCommand = "sacct -n -a -X -o \"JobIDRaw,JobName%%20,User%%20,Partition%%20,State%%20\" -S%02d:%02d:%02d -sBF,CA,CD,CF,F,NF,PR,RS,S,TO | grep -v 'PENDING\\|COMPLETING\\|RUNNING' | uniq"
	qstatCommand = "qstat -u xeuspimi"
)

func (sc *TorqueCollector) collectQstat(ch chan<- prometheus.Metric) {
	log.Debugln("Collecting qstat metrics...")
	var collected uint

	// hour := sc.lasttime.Hour()
	// minute := sc.lasttime.Minute()
	// second := sc.lasttime.Second()

	// now := time.Now().In(sc.timeZone)
	// if now.Hour() < hour {
	// 	hour = 0
	// 	minute = 0
	// 	second = 0
	// }

	// currentCommand := fmt.Sprintf(qstatCommand, hour, minute, second)
	currentCommand := qstatCommand
	log.Debugln(currentCommand)

	sshSession, err := sc.executeSSHCommand(currentCommand)
	if sshSession != nil {
		defer sshSession.Close()
	}
	if err != nil {
		log.Errorf("qstat: %s ", err.Error())
		return
	}

	// wait for stdout to fill (it is being filled async by ssh)
	time.Sleep(100 * time.Millisecond)
	sc.setLastTime()

// spiros start
// xeuspimi@eslogin004:~> qstat -u xeuspimi
//
// hazelhen-batch.hww.hlrs.de:
//                                                                                   Req'd       Req'd       Elap
// Job ID                  Username    Queue    Jobname          SessID  NDS   TSK   Memory      Time    S   Time
// ----------------------- ----------- -------- ---------------- ------ ----- ------ --------- --------- - ---------
// 2483314.hazelhen-batch  xeuspimi    single   euxduwkde4         8006     1    --        --   01:00:00 C       --
// xeuspimi@eslogin004:~>

	// 	don't know how to return without header line and blank lines,
	// 	so jump over them this way for now
	// TODO: something about this
	// 	so, can probably look for the last item in the header line
	// 	and then do a ReadString('COMPLETIONTIME\n') type of thing...
	var buffer = sshSession.OutBuffer
	line, error := buffer.ReadString('\n')	// new line
	line, error = buffer.ReadString('\n')	// hazelhen-batch.hww.hlrs.de:
	line, error = buffer.ReadString('\n')	// new line
	line, error = buffer.ReadString('\n')	// header line: "Job ID..."
	line, error = buffer.ReadString('\n')	// dashes: "------..."
	fmt.Println(line, error)
// spiros end

	nextLine := nextLineIterator(sshSession.OutBuffer, qstatLineParser)
	for fields, err := nextLine(); err == nil; fields, err = nextLine() {
		// check the line is correctly parsed
		if err != nil {
			log.Warnln(err.Error())
			continue
		}

		// parse and send job state
		// status, statusOk := StatusDict[fields[aSTATE]]
		status, statusOk := StatusDict[fields[aS]]
		if statusOk {
			// if jobIsNotInQueue(status) {
			ch <- prometheus.MustNewConstMetric(
				sc.userJobs,
				prometheus.GaugeValue,
				float64(status),
				// fields[aJOBID], fields[aNAME], fields[aUSERNAME], fields[aPARTITION],
				// fields[aJOBID], fields[aJOBNAME], fields[aUSERNAME], fields[aQUEUE],
				fields[aJOBID], 
				fields[aUSERNAME], 
				fields[aJOBNAME], 
				fields[aS],
			)
			sc.alreadyRegistered = append(sc.alreadyRegistered, fields[aJOBID])
			//log.Debugln("Job " + fields[aJOBID] + " finished with state " + fields[aSTATE])
			collected++
			// }
		} else {
			// log.Warnf("Couldn't parse job status: '%s', fields '%s'", fields[aSTATE], strings.Join(fields, "|"))
			log.Warnf("Couldn't parse job status: '%s', fields '%s'", fields[aS], strings.Join(fields, "|"))
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
