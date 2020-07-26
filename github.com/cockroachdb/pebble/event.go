// Copyright 2018 The LevelDB-Go and Pebble Authors. All rights reserved. Use
// of this source code is governed by a BSD-style license that can be found in
// the LICENSE file.

package pebble

import (
	"strings"
	"time"

	"github.com/cockroachdb/pebble/internal/humanize"
	"github.com/cockroachdb/pebble/internal/manifest"
	"github.com/cockroachdb/redact"
)

// TableInfo exports the manifest.TableInfo type.
type TableInfo = manifest.TableInfo

func tablesTotalSize(tables []TableInfo) uint64 {
	var size uint64
	for i := range tables {
		size += tables[i].Size
	}
	return size
}

func formatFileNums(tables []TableInfo) string {
	var buf strings.Builder
	for i := range tables {
		if i > 0 {
			buf.WriteString(" ")
		}
		buf.WriteString(tables[i].FileNum.String())
	}
	return buf.String()
}

// LevelInfo contains info pertaining to a partificular level.
type LevelInfo struct {
	Level  int
	Tables []TableInfo
}

func (i LevelInfo) String() string {
	return string(redact.Sprintfn(i.safeFormat))
}

func (i LevelInfo) safeFormat(w redact.SafePrinter) {
	w.Printf("L%d [%s] (%s)", redact.Safe(i.Level), redact.Safe(formatFileNums(i.Tables)),
		redact.Safe(humanize.Uint64(tablesTotalSize(i.Tables))))
}

func redactSprint(s redact.SafeFormatter) string {
	return string(redact.Sprintfn(func(w redact.SafePrinter) {
		s.SafeFormat(w, 's')
	}))
}

// CompactionInfo contains the info for a compaction event.
type CompactionInfo struct {
	// JobID is the ID of the compaction job.
	JobID int
	// Reason is the reason for the compaction.
	Reason string
	// Input contains the input tables for the compaction organized by level.
	Input []LevelInfo
	// Output contains the output tables generated by the compaction. The output
	// tables are empty for the compaction begin event.
	Output   LevelInfo
	Duration time.Duration
	Done     bool
	Err      error
}

func (i CompactionInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i CompactionInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	if i.Err != nil {
		w.Printf("[JOB %d] compaction to L%d error: %s",
			redact.Safe(i.JobID), redact.Safe(i.Output.Level), i.Err)
		return
	}

	if !i.Done {
		w.Printf("[JOB %d] compacting ", redact.Safe(i.JobID))
		i.safeFormatInputs(w)
		return
	}
	outputSize := tablesTotalSize(i.Output.Tables)
	w.Printf("[JOB %d] compacted ", redact.Safe(i.JobID))
	i.safeFormatInputs(w)
	w.Printf(" -> L%d [%s] (%s), in %.1fs, output rate %s/s",
		redact.Safe(i.Output.Level),
		redact.Safe(formatFileNums(i.Output.Tables)),
		redact.Safe(humanize.Uint64(outputSize)),
		redact.Safe(i.Duration.Seconds()),
		redact.Safe(humanize.Uint64(uint64(float64(outputSize)/i.Duration.Seconds()))))
}

func (i CompactionInfo) safeFormatInputs(w redact.SafePrinter) {
	for j, levelInfo := range i.Input {
		if j > 0 {
			w.Printf(" + ")
		}
		levelInfo.safeFormat(w)
	}
}

// FlushInfo contains the info for a flush event.
type FlushInfo struct {
	// JobID is the ID of the flush job.
	JobID int
	// Reason is the reason for the flush.
	Reason string
	// Input contains the count of input memtables that were flushed.
	Input int
	// Output contains the ouptut table generated by the flush. The output info
	// is empty for the flush begin event.
	Output   []TableInfo
	Duration time.Duration
	Done     bool
	Err      error
}

func (i FlushInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i FlushInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	if i.Err != nil {
		w.Printf("[JOB %d] flush error: %s", redact.Safe(i.JobID), i.Err)
		return
	}

	plural := redact.SafeString("s")
	if i.Input == 1 {
		plural = ""
	}
	if !i.Done {
		w.Printf("[JOB %d] flushing %d memtable", redact.Safe(i.JobID), redact.Safe(i.Input))
		w.SafeString(plural)
		w.Printf(" to L0")
		return
	}

	outputSize := tablesTotalSize(i.Output)
	w.Printf("[JOB %d] flushed %d memtable%s to L0 [%s] (%s), in %.1fs, output rate %s/s",
		redact.Safe(i.JobID), redact.Safe(i.Input), plural,
		redact.Safe(formatFileNums(i.Output)),
		redact.Safe(humanize.Uint64(outputSize)),
		redact.Safe(i.Duration.Seconds()),
		redact.Safe(humanize.Uint64(uint64(float64(outputSize)/i.Duration.Seconds()))))
}

// ManifestCreateInfo contains info about a manifest creation event.
type ManifestCreateInfo struct {
	// JobID is the ID of the job the caused the manifest to be created.
	JobID int
	Path  string
	// The file number of the new Manifest.
	FileNum FileNum
	Err     error
}

func (i ManifestCreateInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i ManifestCreateInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	if i.Err != nil {
		w.Printf("[JOB %d] MANIFEST create error: %s", redact.Safe(i.JobID), i.Err)
		return
	}
	w.Printf("[JOB %d] MANIFEST created %s", redact.Safe(i.JobID), redact.Safe(i.FileNum))
}

// ManifestDeleteInfo contains the info for a Manifest deletion event.
type ManifestDeleteInfo struct {
	// JobID is the ID of the job the caused the Manifest to be deleted.
	JobID   int
	Path    string
	FileNum FileNum
	Err     error
}

func (i ManifestDeleteInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i ManifestDeleteInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	if i.Err != nil {
		w.Printf("[JOB %d] MANIFEST delete error: %s", redact.Safe(i.JobID), i.Err)
		return
	}
	w.Printf("[JOB %d] MANIFEST deleted %s", redact.Safe(i.JobID), redact.Safe(i.FileNum))
}

// TableCreateInfo contains the info for a table creation event.
type TableCreateInfo struct {
	JobID int
	// Reason is the reason for the table creation: "compacting", "flushing", or
	// "ingesting".
	Reason  string
	Path    string
	FileNum FileNum
}

func (i TableCreateInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i TableCreateInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	w.Printf("[JOB %d] %s: sstable created %s",
		redact.Safe(i.JobID), redact.Safe(i.Reason), redact.Safe(i.FileNum))
}

// TableDeleteInfo contains the info for a table deletion event.
type TableDeleteInfo struct {
	JobID   int
	Path    string
	FileNum FileNum
	Err     error
}

func (i TableDeleteInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i TableDeleteInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	if i.Err != nil {
		w.Printf("[JOB %d] sstable delete error %s: %s",
			redact.Safe(i.JobID), redact.Safe(i.FileNum), i.Err)
		return
	}
	w.Printf("[JOB %d] sstable deleted %s", redact.Safe(i.JobID), redact.Safe(i.FileNum))
}

// TableIngestInfo contains the info for a table ingestion event.
type TableIngestInfo struct {
	// JobID is the ID of the job the caused the table to be ingested.
	JobID  int
	Tables []struct {
		TableInfo
		Level int
	}
	// GlobalSeqNum is the sequence number that was assigned to all entries in
	// the ingested table.
	GlobalSeqNum uint64
	Err          error
}

func (i TableIngestInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i TableIngestInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	if i.Err != nil {
		w.Printf("[JOB %d] ingest error: %s", redact.Safe(i.JobID), i.Err)
		return
	}

	w.Printf("[JOB %d] ingested", redact.Safe(i.JobID))
	for j := range i.Tables {
		t := &i.Tables[j]
		if j > 0 {
			w.Printf(",")
		}
		w.Printf(" L%d:%s (%s)", redact.Safe(t.Level), redact.Safe(t.FileNum),
			redact.Safe(humanize.Uint64(t.Size)))
	}
}

// TableStatsInfo contains the info for a table stats loaded event.
type TableStatsInfo struct {
	// JobID is the ID of the job that finished loading the initial tables'
	// stats.
	JobID int
}

func (i TableStatsInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i TableStatsInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	w.Printf("[JOB %d] all initial table stats loaded", redact.Safe(i.JobID))
}

// WALCreateInfo contains info about a WAL creation event.
type WALCreateInfo struct {
	// JobID is the ID of the job the caused the WAL to be created.
	JobID int
	Path  string
	// The file number of the new WAL.
	FileNum FileNum
	// The file number of a previous WAL which was recycled to create this
	// one. Zero if recycling did not take place.
	RecycledFileNum FileNum
	Err             error
}

func (i WALCreateInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i WALCreateInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	if i.Err != nil {
		w.Printf("[JOB %d] WAL create error: %s", redact.Safe(i.JobID), i.Err)
		return
	}

	if i.RecycledFileNum == 0 {
		w.Printf("[JOB %d] WAL created %s", redact.Safe(i.JobID), redact.Safe(i.FileNum))
		return
	}

	w.Printf("[JOB %d] WAL created %s (recycled %s)",
		redact.Safe(i.JobID), redact.Safe(i.FileNum), redact.Safe(i.RecycledFileNum))
}

// WALDeleteInfo contains the info for a WAL deletion event.
type WALDeleteInfo struct {
	// JobID is the ID of the job the caused the WAL to be deleted.
	JobID   int
	Path    string
	FileNum FileNum
	Err     error
}

func (i WALDeleteInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i WALDeleteInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	if i.Err != nil {
		w.Printf("[JOB %d] WAL delete error: %s", redact.Safe(i.JobID), i.Err)
		return
	}
	w.Printf("[JOB %d] WAL deleted %s", redact.Safe(i.JobID), redact.Safe(i.FileNum))
}

// WriteStallBeginInfo contains the info for a write stall begin event.
type WriteStallBeginInfo struct {
	Reason string
}

func (i WriteStallBeginInfo) String() string {
	return redactSprint(i)
}

// SafeFormat implements redact.SafeFormatter.
func (i WriteStallBeginInfo) SafeFormat(w redact.SafePrinter, _ rune) {
	w.Printf("write stall beginning: %s", redact.Safe(i.Reason))
}

// EventListener contains a set of functions that will be invoked when various
// significant DB events occur. Note that the functions should not run for an
// excessive amount of time as they are invoked synchronously by the DB and may
// block continued DB work. For a similar reason it is advisable to not perform
// any synchronous calls back into the DB.
type EventListener struct {
	// BackgroundError is invoked whenever an error occurs during a background
	// operation such as flush or compaction.
	BackgroundError func(error)

	// CompactionBegin is invoked after the inputs to a compaction have been
	// determined, but before the compaction has produced any output.
	CompactionBegin func(CompactionInfo)

	// CompactionEnd is invoked after a compaction has completed and the result
	// has been installed.
	CompactionEnd func(CompactionInfo)

	// FlushBegin is invoked after the inputs to a flush have been determined,
	// but before the flush has produced any output.
	FlushBegin func(FlushInfo)

	// FlushEnd is invoked after a flush has complated and the result has been
	// installed.
	FlushEnd func(FlushInfo)

	// ManifestCreated is invoked after a manifest has been created.
	ManifestCreated func(ManifestCreateInfo)

	// ManifestDeleted is invoked after a manifest has been deleted.
	ManifestDeleted func(ManifestDeleteInfo)

	// TableCreated is invoked when a table has been created.
	TableCreated func(TableCreateInfo)

	// TableDeleted is invoked after a table has been deleted.
	TableDeleted func(TableDeleteInfo)

	// TableIngested is invoked after an externally created table has been
	// ingested via a call to DB.Ingest().
	TableIngested func(TableIngestInfo)

	// TableStatsLoaded is invoked at most once, when the table stats
	// collector has loaded statistics for all tables that existed at Open.
	TableStatsLoaded func(TableStatsInfo)

	// WALCreated is invoked after a WAL has been created.
	WALCreated func(WALCreateInfo)

	// WALDeleted is invoked after a WAL has been deleted.
	WALDeleted func(WALDeleteInfo)

	// WriteStallBegin is invoked when writes are intentionally delayed.
	WriteStallBegin func(WriteStallBeginInfo)

	// WriteStallEnd is invoked when delayed writes are released.
	WriteStallEnd func()
}

// EnsureDefaults ensures that background error events are logged to the
// specified logger if a handler for those events hasn't been otherwise
// specified. Ensure all handlers are non-nil so that we don't have to check
// for nil-ness before invoking.
func (l *EventListener) EnsureDefaults(logger Logger) {
	if l.BackgroundError == nil {
		l.BackgroundError = func(err error) {
			logger.Infof("background error: %s", err)
		}
	}
	if l.CompactionBegin == nil {
		l.CompactionBegin = func(info CompactionInfo) {}
	}
	if l.CompactionEnd == nil {
		l.CompactionEnd = func(info CompactionInfo) {}
	}
	if l.FlushBegin == nil {
		l.FlushBegin = func(info FlushInfo) {}
	}
	if l.FlushEnd == nil {
		l.FlushEnd = func(info FlushInfo) {}
	}
	if l.ManifestCreated == nil {
		l.ManifestCreated = func(info ManifestCreateInfo) {}
	}
	if l.ManifestDeleted == nil {
		l.ManifestDeleted = func(info ManifestDeleteInfo) {}
	}
	if l.TableCreated == nil {
		l.TableCreated = func(info TableCreateInfo) {}
	}
	if l.TableDeleted == nil {
		l.TableDeleted = func(info TableDeleteInfo) {}
	}
	if l.TableIngested == nil {
		l.TableIngested = func(info TableIngestInfo) {}
	}
	if l.TableStatsLoaded == nil {
		l.TableStatsLoaded = func(info TableStatsInfo) {}
	}
	if l.WALCreated == nil {
		l.WALCreated = func(info WALCreateInfo) {}
	}
	if l.WALDeleted == nil {
		l.WALDeleted = func(info WALDeleteInfo) {}
	}
	if l.WriteStallBegin == nil {
		l.WriteStallBegin = func(info WriteStallBeginInfo) {}
	}
	if l.WriteStallEnd == nil {
		l.WriteStallEnd = func() {}
	}
}

// MakeLoggingEventListener creates an EventListener that logs all events to the
// specified logger.
func MakeLoggingEventListener(logger Logger) EventListener {
	if logger == nil {
		logger = DefaultLogger
	}

	return EventListener{
		BackgroundError: func(err error) {
			logger.Infof("background error: %s", err)
		},
		CompactionBegin: func(info CompactionInfo) {
			logger.Infof("%s", info)
		},
		CompactionEnd: func(info CompactionInfo) {
			logger.Infof("%s", info)
		},
		FlushBegin: func(info FlushInfo) {
			logger.Infof("%s", info)
		},
		FlushEnd: func(info FlushInfo) {
			logger.Infof("%s", info)
		},
		ManifestCreated: func(info ManifestCreateInfo) {
			logger.Infof("%s", info)
		},
		ManifestDeleted: func(info ManifestDeleteInfo) {
			logger.Infof("%s", info)
		},
		TableCreated: func(info TableCreateInfo) {
			logger.Infof("%s", info)
		},
		TableDeleted: func(info TableDeleteInfo) {
			logger.Infof("%s", info)
		},
		TableIngested: func(info TableIngestInfo) {
			logger.Infof("%s", info)
		},
		TableStatsLoaded: func(info TableStatsInfo) {
			logger.Infof("%s", info)
		},
		WALCreated: func(info WALCreateInfo) {
			logger.Infof("%s", info)
		},
		WALDeleted: func(info WALDeleteInfo) {
			logger.Infof("%s", info)
		},
		WriteStallBegin: func(info WriteStallBeginInfo) {
			logger.Infof("%s", info)
		},
		WriteStallEnd: func() {
			logger.Infof("write stall ending")
		},
	}
}
