// Copyright (c) 2013-2017 The btcsuite developers
// Copyright (c) 2017 The Decred developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/lbryio/lbcd/addrmgr"
	"github.com/lbryio/lbcd/blockchain"
	"github.com/lbryio/lbcd/blockchain/indexers"
	"github.com/lbryio/lbcd/claimtrie/node"
	"github.com/lbryio/lbcd/connmgr"
	"github.com/lbryio/lbcd/database"
	"github.com/lbryio/lbcd/fees"
	"github.com/lbryio/lbcd/mempool"
	"github.com/lbryio/lbcd/mining"
	"github.com/lbryio/lbcd/mining/cpuminer"
	"github.com/lbryio/lbcd/netsync"
	"github.com/lbryio/lbcd/peer"
	"github.com/lbryio/lbcd/txscript"

	"github.com/btcsuite/btclog"
	"github.com/jrick/logrotate/rotator"
)

// logWriter implements an io.Writer that outputs to both standard output and
// the write-end pipe of an initialized log rotator.
type logWriter struct{}

func (logWriter) Write(p []byte) (n int, err error) {
	os.Stdout.Write(p)
	logRotator.Write(p)
	return len(p), nil
}

// Loggers per subsystem.  A single backend logger is created and all subsytem
// loggers created from it will write to the backend.  When adding new
// subsystems, add the subsystem logger variable here and to the
// subsystemLoggers map.
//
// Loggers can not be used before the log rotator has been initialized with a
// log file.  This must be performed early during application startup by calling
// initLogRotator.
var (
	// backendLog is the logging backend used to create all subsystem loggers.
	// The backend must not be used before the log rotator has been initialized,
	// or data races and/or nil pointer dereferences will occur.
	backendLog = btclog.NewBackend(logWriter{})

	// logRotator is one of the logging outputs.  It should be closed on
	// application shutdown.
	logRotator *rotator.Rotator

	adxrLog = backendLog.Logger("ADXR")
	amgrLog = backendLog.Logger("AMGR")
	bcdbLog = backendLog.Logger("BCDB")
	btcdLog = backendLog.Logger("MAIN")
	chanLog = backendLog.Logger("CHAN")
	cmgrLog = backendLog.Logger("CMGR")
	discLog = backendLog.Logger("DISC")
	feesLog = backendLog.Logger("FEES")
	indxLog = backendLog.Logger("INDX")
	lbryLog = backendLog.Logger("LBRY")
	minrLog = backendLog.Logger("MINR")
	peerLog = backendLog.Logger("PEER")
	rpcsLog = backendLog.Logger("RPCS")
	scrpLog = backendLog.Logger("SCRP")
	srvrLog = backendLog.Logger("SRVR")
	syncLog = backendLog.Logger("SYNC")
	txmpLog = backendLog.Logger("TXMP")
)

// Initialize package-global logger variables.
func init() {
	addrmgr.UseLogger(amgrLog)
	blockchain.UseLogger(chanLog)
	connmgr.UseLogger(cmgrLog)
	cpuminer.UseLogger(minrLog)
	database.UseLogger(bcdbLog)
	fees.UseLogger(feesLog)
	indexers.UseLogger(indxLog)
	mempool.UseLogger(txmpLog)
	mining.UseLogger(minrLog)
	netsync.UseLogger(syncLog)
	node.UseLogger(lbryLog)
	peer.UseLogger(peerLog)
	txscript.UseLogger(scrpLog)
}

// subsystemLoggers maps each subsystem identifier to its associated logger.
var subsystemLoggers = map[string]btclog.Logger{
	"ADXR": adxrLog,
	"AMGR": amgrLog,
	"BCDB": bcdbLog,
	"CHAN": chanLog,
	"CMGR": cmgrLog,
	"DISC": discLog,
	"INDX": indxLog,
	"LBRY": lbryLog,
	"MAIN": btcdLog,
	"MINR": minrLog,
	"PEER": peerLog,
	"RPCS": rpcsLog,
	"SCRP": scrpLog,
	"SRVR": srvrLog,
	"SYNC": syncLog,
	"TXMP": txmpLog,
}

// initLogRotator initializes the logging rotater to write logs to logFile and
// create roll files in the same directory.  It must be called before the
// package-global log rotater variables are used.
func initLogRotator(logFile string) {
	logDir, _ := filepath.Split(logFile)
	err := os.MkdirAll(logDir, 0700)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log directory: %v\n", err)
		os.Exit(1)
	}
	r, err := rotator.New(logFile, 40*1024, false, 3)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create file rotator: %v\n", err)
		os.Exit(1)
	}

	logRotator = r
}

// setLogLevel sets the logging level for provided subsystem.  Invalid
// subsystems are ignored.  Uninitialized subsystems are dynamically created as
// needed.
func setLogLevel(subsystemID string, logLevel string) {
	// Ignore invalid subsystems.
	logger, ok := subsystemLoggers[subsystemID]
	if !ok {
		return
	}

	// Defaults to info if the log level is invalid.
	level, _ := btclog.LevelFromString(logLevel)
	logger.SetLevel(level)
}

// setLogLevels sets the log level for all subsystem loggers to the passed
// level.  It also dynamically creates the subsystem loggers as needed, so it
// can be used to initialize the logging system.
func setLogLevels(logLevel string) {
	// Configure all sub-systems with the new logging level.  Dynamically
	// create loggers as needed.
	for subsystemID := range subsystemLoggers {
		setLogLevel(subsystemID, logLevel)
	}
}

// directionString is a helper function that returns a string that represents
// the direction of a connection (inbound or outbound).
func directionString(inbound bool) string {
	if inbound {
		return "inbound"
	}
	return "outbound"
}

// pickNoun returns the singular or plural form of a noun depending
// on the count n.
func pickNoun(n uint64, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}
