package handlers

// HealthHandler returns basic health info.
func HealthHandler(opts HandlerOpts) error {
	opts.Respond(true, map[string]interface{}{
		"ok":      true,
		"version": opts.Context.Version,
	}, nil, nil)
	return nil
}
