package handlers

import (
	"runtime"
	"time"
)

var processStartTime = time.Now()

// StatusHandler returns system status summary.
func StatusHandler(opts HandlerOpts) error {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	opts.Respond(true, map[string]interface{}{
		"version":    opts.Context.Version,
		"uptimeMs":   time.Since(processStartTime).Milliseconds(),
		"goroutines": runtime.NumGoroutine(),
		"memory": map[string]interface{}{
			"allocMb":      float64(memStats.Alloc) / (1024 * 1024),
			"totalAllocMb": float64(memStats.TotalAlloc) / (1024 * 1024),
			"sysMb":        float64(memStats.Sys) / (1024 * 1024),
			"numGC":        memStats.NumGC,
		},
		"platform": runtime.GOOS + "/" + runtime.GOARCH,
	}, nil, nil)
	return nil
}
