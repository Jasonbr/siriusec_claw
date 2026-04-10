package handlers

import (
	"os"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/cron"
)

var cronStore *cron.Store

func getCronStore() *cron.Store {
	if cronStore == nil {
		var err error
		cronStore, err = cron.NewStore(os.Getenv)
		if err != nil {
			return nil
		}
	}
	return cronStore
}

// --- cron.list ---

func CronListHandler(opts HandlerOpts) error {
	store := getCronStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("cron store not available"), nil)
		return nil
	}

	jobs := store.List()
	opts.Respond(true, map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	}, nil, nil)
	return nil
}

// --- cron.status ---

func CronStatusHandler(opts HandlerOpts) error {
	store := getCronStore()
	if store == nil {
		opts.Respond(true, map[string]interface{}{
			"enabled": false,
			"jobs":    0,
		}, nil, nil)
		return nil
	}

	jobs := store.List()
	enabled := 0
	for _, j := range jobs {
		if j.Enabled {
			enabled++
		}
	}

	opts.Respond(true, map[string]interface{}{
		"enabled":     true,
		"totalJobs":   len(jobs),
		"enabledJobs": enabled,
	}, nil, nil)
	return nil
}

// --- cron.add ---

func CronAddHandler(opts HandlerOpts) error {
	store := getCronStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("cron store not available"), nil)
		return nil
	}

	name := stringParam(opts.Params, "name", "")
	schedule := stringParam(opts.Params, "schedule", "")
	message := stringParam(opts.Params, "message", "")

	if schedule == "" || message == "" {
		opts.Respond(false, nil, errInvalidParams("schedule and message required"), nil)
		return nil
	}

	job := &cron.Job{
		Name:       name,
		Schedule:   schedule,
		Message:    message,
		AgentID:    stringParam(opts.Params, "agentId", "main"),
		SessionKey: stringParam(opts.Params, "sessionKey", ""),
		Channel:    stringParam(opts.Params, "channel", ""),
		To:         stringParam(opts.Params, "to", ""),
		ChatType:   stringParam(opts.Params, "chatType", ""),
		Enabled:    true,
	}

	if err := store.Add(job); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":  true,
		"job": job,
	}, nil, nil)
	return nil
}

// --- cron.remove ---

func CronRemoveHandler(opts HandlerOpts) error {
	store := getCronStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("cron store not available"), nil)
		return nil
	}

	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	if err := store.Remove(id); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"ok": true, "id": id, "removed": true}, nil, nil)
	return nil
}

// --- cron.update ---

func CronUpdateHandler(opts HandlerOpts) error {
	store := getCronStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("cron store not available"), nil)
		return nil
	}

	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	job := store.Get(id)
	if job == nil {
		opts.Respond(false, nil, errInvalidParams("job not found: "+id), nil)
		return nil
	}

	if v, ok := opts.Params["name"]; ok {
		if s, ok := v.(string); ok {
			job.Name = s
		}
	}
	if v, ok := opts.Params["schedule"]; ok {
		if s, ok := v.(string); ok {
			job.Schedule = s
		}
	}
	if v, ok := opts.Params["message"]; ok {
		if s, ok := v.(string); ok {
			job.Message = s
		}
	}
	if v, ok := opts.Params["enabled"]; ok {
		if b, ok := v.(bool); ok {
			job.Enabled = b
		}
	}

	if err := store.Update(job); err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{"ok": true, "job": job}, nil, nil)
	return nil
}

// --- cron.run ---

func CronRunHandler(opts HandlerOpts) error {
	store := getCronStore()
	if store == nil {
		opts.Respond(false, nil, errInternal("cron store not available"), nil)
		return nil
	}

	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	job := store.Get(id)
	if job == nil {
		opts.Respond(false, nil, errInvalidParams("job not found: "+id), nil)
		return nil
	}

	// Trigger manual run via chat.send
	if opts.Context != nil && opts.Context.InvokeMethod != nil {
		sessionKey := job.SessionKey
		if sessionKey == "" {
			sessionKey = "agent:" + job.AgentID + ":cron:" + job.ID
		}
		opts.Context.InvokeMethod("chat.send", map[string]interface{}{
			"sessionKey": sessionKey,
			"message":    job.Message,
			"channel":    job.Channel,
			"to":         job.To,
			"chatType":   job.ChatType,
		})
	}

	now := time.Now().UnixMilli()
	store.RecordRun(job.ID, "manual-"+job.ID)

	opts.Respond(true, map[string]interface{}{
		"ok":          true,
		"id":          id,
		"triggeredAt": now,
	}, nil, nil)
	return nil
}

// --- cron.runs ---

func CronRunsHandler(opts HandlerOpts) error {
	jobID := stringParam(opts.Params, "id", "")
	limit := intParam(opts.Params, "limit", 50)

	records, err := cron.LoadRunRecords(os.Getenv, jobID, limit)
	if err != nil {
		opts.Respond(false, nil, errInternal(err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"runs":  records,
		"count": len(records),
	}, nil, nil)
	return nil
}
