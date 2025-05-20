package processscraper

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

// Scraper collects process metrics from procfs.
type Scraper struct {
	pid int
}

// New creates a new Scraper for the given pid.
func New(pid int) *Scraper {
	return &Scraper{pid: pid}
}

// Scrape collects metrics for the configured pid.
func (s *Scraper) Scrape() (pmetric.Metrics, error) {
	md := pmetric.NewMetrics()
	rm := md.ResourceMetrics().AppendEmpty()
	rm.Resource().Attributes().PutInt("process.pid", int64(s.pid))
	sm := rm.ScopeMetrics().AppendEmpty()

	// open fds
	fdDir := fmt.Sprintf("/proc/%d/fd", s.pid)
	entries, err := ioutil.ReadDir(fdDir)
	if err != nil {
		return md, err
	}
	openFDs := len(entries)

	mFDs := sm.Metrics().AppendEmpty()
	mFDs.SetName("process.open_fds")
	g := mFDs.SetEmptyGauge()
	dp := g.DataPoints().AppendEmpty()
	dp.SetIntValue(int64(openFDs))
	dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))

	// threads and io stats from /proc/<pid>/status and io
	threads, err := readThreads(s.pid)
	if err == nil {
		mThreads := sm.Metrics().AppendEmpty()
		mThreads.SetName("process.threads")
		g2 := mThreads.SetEmptyGauge()
		dp2 := g2.DataPoints().AppendEmpty()
		dp2.SetIntValue(int64(threads))
		dp2.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	readBytes, writeBytes := readIOStats(s.pid)
	if readBytes >= 0 {
		mRead := sm.Metrics().AppendEmpty()
		mRead.SetName("process.io.read_bytes")
		sum := mRead.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		dp3 := sum.DataPoints().AppendEmpty()
		dp3.SetIntValue(readBytes)
		dp3.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}
	if writeBytes >= 0 {
		mWrite := sm.Metrics().AppendEmpty()
		mWrite.SetName("process.io.write_bytes")
		sum := mWrite.SetEmptySum()
		sum.SetIsMonotonic(true)
		sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
		dp4 := sum.DataPoints().AppendEmpty()
		dp4.SetIntValue(writeBytes)
		dp4.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	}

	return md, nil
}

func readThreads(pid int) (int, error) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/status", pid))
	if err != nil {
		return 0, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "Threads:") {
			fields := strings.Fields(line)
			if len(fields) == 2 {
				t, err := strconv.Atoi(fields[1])
				if err == nil {
					return t, nil
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}
	return 0, fmt.Errorf("threads not found")
}

func readIOStats(pid int) (int64, int64) {
	path := fmt.Sprintf("/proc/%d/io", pid)
	f, err := os.Open(path)
	if err != nil {
		return -1, -1
	}
	defer f.Close()
	var readBytes, writeBytes int64 = -1, -1
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		parts := strings.Split(scanner.Text(), ":")
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		valStr := strings.TrimSpace(parts[1])
		val, err := strconv.ParseInt(valStr, 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "read_bytes":
			readBytes = val
		case "write_bytes":
			writeBytes = val
		}
	}
	return readBytes, writeBytes
}
