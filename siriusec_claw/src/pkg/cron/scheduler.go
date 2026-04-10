package cron

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/siriusec/siriusec_claw/pkg/logging"
	"github.com/siriusec/siriusec_claw/pkg/paths"
)

var cronLog = logging.Sub("cron")

// Job represents a scheduled cron job.
type Job struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Schedule   string                 `json:"schedule"` // cron expression (e.g. "*/5 * * * *")
	Message    string                 `json:"message"`  // message to send to agent
	AgentID    string                 `json:"agentId,omitempty"`
	SessionKey string                 `json:"sessionKey,omitempty"`
	Channel    string                 `json:"channel,omitempty"`
	To         string                 `json:"to,omitempty"`
	ChatType   string                 `json:"chatType,omitempty"`
	Enabled    bool                   `json:"enabled"`
	CreatedAt  int64                  `json:"createdAt"`
	UpdatedAt  int64                  `json:"updatedAt"`
	LastRunAt  *int64                 `json:"lastRunAt,omitempty"`
	LastRunID  string                 `json:"lastRunId,omitempty"`
	NextRunAt  *int64                 `json:"nextRunAt,omitempty"`
	RunCount   int                    `json:"runCount"`
	Tags       []string               `json:"tags,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// RunRecord is a single execution record of a cron job.
type RunRecord struct {
	RunID      string `json:"runId"`
	JobID      string `json:"jobId"`
	StartedAt  int64  `json:"startedAt"`
	FinishedAt *int64 `json:"finishedAt,omitempty"`
	Status     string `json:"status"` // "running", "completed", "failed", "aborted"
	Output     string `json:"output,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs *int64 `json:"durationMs,omitempty"`
}

// Store manages persistent cron job storage.
type Store struct {
	mu       sync.RWMutex
	jobs     map[string]*Job
	filePath string
}

// RunFunc is the function called when a cron job fires.
type RunFunc func(job *Job) (string, error)

// Scheduler manages cron job scheduling and execution.
type Scheduler struct {
	store   *Store
	runFunc RunFunc
	mu      sync.Mutex
	running bool
	stopCh  chan struct{}
	ticker  *time.Ticker
}

// ResolveCronDir returns the cron storage directory.
func ResolveCronDir(env func(string) string) string {
	return filepath.Join(paths.ResolveStateDir(env), "cron")
}

// ResolveCronStorePath returns the cron jobs.json path.
func ResolveCronStorePath(env func(string) string) string {
	return filepath.Join(ResolveCronDir(env), "jobs.json")
}

// ResolveCronRunsDir returns the directory for cron run records.
func ResolveCronRunsDir(env func(string) string) string {
	return filepath.Join(ResolveCronDir(env), "runs")
}

// NewStore creates or loads a cron job store.
func NewStore(env func(string) string) (*Store, error) {
	fp := ResolveCronStorePath(env)
	dir := filepath.Dir(fp)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &Store{
		jobs:     make(map[string]*Job),
		filePath: fp,
	}

	data, err := os.ReadFile(fp)
	if err == nil && len(data) > 0 {
		var jobs map[string]*Job
		if json.Unmarshal(data, &jobs) == nil {
			s.jobs = jobs
		}
	}

	return s, nil
}

// List returns all jobs.
func (s *Store) List() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		jobs = append(jobs, j)
	}
	return jobs
}

// Get returns a job by ID.
func (s *Store) Get(id string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jobs[id]
}

// Add creates a new cron job.
func (s *Store) Add(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job.ID == "" {
		job.ID = uuid.New().String()
	}
	now := time.Now().UnixMilli()
	job.CreatedAt = now
	job.UpdatedAt = now
	s.jobs[job.ID] = job
	return s.save()
}

// Update modifies an existing cron job.
func (s *Store) Update(job *Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[job.ID]; !ok {
		return nil
	}
	job.UpdatedAt = time.Now().UnixMilli()
	s.jobs[job.ID] = job
	return s.save()
}

// Remove deletes a cron job.
func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.jobs, id)
	return s.save()
}

// RecordRun marks a job as having run.
func (s *Store) RecordRun(jobID, runID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if j, ok := s.jobs[jobID]; ok {
		now := time.Now().UnixMilli()
		j.LastRunAt = &now
		j.LastRunID = runID
		j.RunCount++
		j.UpdatedAt = now
		_ = s.save()
	}
}

func (s *Store) save() error {
	data, err := json.MarshalIndent(s.jobs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.filePath, data, 0600)
}

// SaveRunRecord persists a run record to disk.
func SaveRunRecord(env func(string) string, record *RunRecord) error {
	dir := ResolveCronRunsDir(env)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	path := filepath.Join(dir, record.RunID+".json")
	return os.WriteFile(path, data, 0600)
}

// LoadRunRecords loads recent run records for a job.
func LoadRunRecords(env func(string) string, jobID string, limit int) ([]*RunRecord, error) {
	dir := ResolveCronRunsDir(env)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var records []*RunRecord
	for i := len(entries) - 1; i >= 0 && len(records) < limit; i-- {
		e := entries[i]
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var rec RunRecord
		if json.Unmarshal(data, &rec) != nil {
			continue
		}
		if jobID != "" && rec.JobID != jobID {
			continue
		}
		records = append(records, &rec)
	}
	return records, nil
}

// NewScheduler creates a new cron scheduler.
func NewScheduler(store *Store, runFunc RunFunc) *Scheduler {
	return &Scheduler{
		store:   store,
		runFunc: runFunc,
		stopCh:  make(chan struct{}),
	}
}

// Start begins the cron scheduler loop (checks every minute).
func (s *Scheduler) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.ticker = time.NewTicker(60 * time.Second)
	s.mu.Unlock()

	go func() {
		// Run initial check
		s.tick()
		for {
			select {
			case <-s.ticker.C:
				s.tick()
			case <-s.stopCh:
				return
			}
		}
	}()

	cronLog.Info("cron scheduler started")
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.running {
		return
	}
	s.running = false
	if s.ticker != nil {
		s.ticker.Stop()
	}
	close(s.stopCh)
	cronLog.Info("cron scheduler stopped")
}

func (s *Scheduler) tick() {
	jobs := s.store.List()
	now := time.Now()

	for _, job := range jobs {
		if !job.Enabled {
			continue
		}
		if !shouldRun(job, now) {
			continue
		}

		runID := uuid.New().String()
		s.store.RecordRun(job.ID, runID)

		go func(j *Job, rid string) {
			startedAt := time.Now().UnixMilli()
			record := &RunRecord{
				RunID:     rid,
				JobID:     j.ID,
				StartedAt: startedAt,
				Status:    "running",
			}

			output, err := s.runFunc(j)
			finishedAt := time.Now().UnixMilli()
			record.FinishedAt = &finishedAt
			dur := finishedAt - startedAt
			record.DurationMs = &dur

			if err != nil {
				record.Status = "failed"
				record.Error = err.Error()
				cronLog.Error("cron job %s failed: %v", j.ID, err)
			} else {
				record.Status = "completed"
				record.Output = output
			}
			_ = SaveRunRecord(os.Getenv, record)
		}(job, runID)
	}
}

// shouldRun checks if a cron job should run based on its schedule.
// Simplified cron: supports "*/N * * * *" (every N minutes) and basic expressions.
func shouldRun(job *Job, now time.Time) bool {
	schedule := job.Schedule
	if schedule == "" {
		return false
	}

	// Simple interval check: if lastRunAt + interval < now
	if job.LastRunAt != nil {
		interval := parseSimpleInterval(schedule)
		if interval > 0 {
			lastRun := time.UnixMilli(*job.LastRunAt)
			return now.Sub(lastRun) >= interval
		}
	} else {
		// Never run before - run now if schedule is valid
		return parseSimpleInterval(schedule) > 0
	}

	return false
}

// parseSimpleInterval parses simple cron-like intervals.
// Supports: "*/5 * * * *" (every 5 minutes), "0 * * * *" (hourly), "0 0 * * *" (daily)
func parseSimpleInterval(schedule string) time.Duration {
	parts := splitFields(schedule)
	if len(parts) < 5 {
		return 0
	}

	minute := parts[0]
	hour := parts[1]

	// "*/N * * * *" - every N minutes
	if len(minute) > 2 && minute[:2] == "*/" {
		n := 0
		for _, c := range minute[2:] {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > 0 {
			return time.Duration(n) * time.Minute
		}
	}

	// "0 * * * *" - every hour
	if minute == "0" && hour == "*" {
		return time.Hour
	}

	// "0 0 * * *" - every day
	if minute == "0" && hour == "0" {
		return 24 * time.Hour
	}

	// "0 */N * * *" - every N hours
	if minute == "0" && len(hour) > 2 && hour[:2] == "*/" {
		n := 0
		for _, c := range hour[2:] {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			}
		}
		if n > 0 {
			return time.Duration(n) * time.Hour
		}
	}

	return 0
}

func splitFields(s string) []string {
	var fields []string
	field := ""
	for _, c := range s {
		if c == ' ' || c == '\t' {
			if field != "" {
				fields = append(fields, field)
				field = ""
			}
		} else {
			field += string(c)
		}
	}
	if field != "" {
		fields = append(fields, field)
	}
	return fields
}
