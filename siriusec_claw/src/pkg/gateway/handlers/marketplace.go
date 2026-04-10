package handlers

import (
	"context"
	"time"

	"github.com/siriusec/siriusec_claw/pkg/config"
	"github.com/siriusec/siriusec_claw/pkg/marketplace"
)

const defaultMarketplaceURL = "https://openocta.ai"

// getMarketplaceClient creates a marketplace client with the following priority:
// 1. SKILL_API_URL environment variable
// 2. SKILL_API_TOKEN environment variable for auth
// 3. Marketplace config in config file
func getMarketplaceClient() *marketplace.Client {
	env := envGetter()
	url := defaultMarketplaceURL
	var token string

	// Priority 1: Environment variables
	if envURL := env("SKILL_API_URL"); envURL != "" {
		url = envURL
	}
	if envToken := env("SKILL_API_TOKEN"); envToken != "" {
		token = envToken
	}

	// Priority 2: Config file (only if env not set)
	marketCfg := config.GetMarketplaceConfig(env)
	if marketCfg != nil {
		if env("SKILL_API_URL") == "" && marketCfg.URL != "" {
			url = marketCfg.URL
		}
		if token == "" {
			token = marketCfg.Token
		}
	}

	client := marketplace.NewClient(url)
	if token != "" {
		client.SetToken(token)
	}
	return client
}

// --- marketplace.skills ---

func MarketplaceSkillsHandler(opts HandlerOpts) error {
	query := stringParam(opts.Params, "query", "")

	client := getMarketplaceClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.ListSkills(ctx, query)
	if err != nil {
		opts.Respond(false, nil, errInternal("failed to fetch skills: "+err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"skills": result.Skills,
		"total":  result.Total,
	}, nil, nil)
	return nil
}

// --- marketplace.skill.get ---

func MarketplaceSkillGetHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	client := getMarketplaceClient()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := client.GetSkill(ctx, id)
	if err != nil {
		opts.Respond(false, nil, errInternal("failed to fetch skill: "+err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"skill": result,
	}, nil, nil)
	return nil
}

// --- marketplace.skill.install ---

func MarketplaceSkillInstallHandler(opts HandlerOpts) error {
	id := stringParam(opts.Params, "id", "")
	if id == "" {
		opts.Respond(false, nil, errInvalidParams("id required"), nil)
		return nil
	}

	client := getMarketplaceClient()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Get skill details
	detail, err := client.GetSkill(ctx, id)
	if err != nil {
		opts.Respond(false, nil, errInternal("failed to fetch skill: "+err.Error()), nil)
		return nil
	}

	// Save skill config if variables are defined
	if len(detail.Variables) > 0 {
		env := envGetter()
		cfg := config.SkillConfig{
			Config: make(map[string]interface{}),
		}
		// Apply provided config values
		if providedConfig, ok := opts.Params["config"].(map[string]interface{}); ok {
			cfg.Config = providedConfig
		}
		// Apply provided env values
		if providedEnv, ok := opts.Params["env"].(map[string]interface{}); ok {
			cfg.Env = make(map[string]string)
			for k, v := range providedEnv {
				if s, ok := v.(string); ok {
					cfg.Env[k] = s
				}
			}
		}
		if err := config.SetSkillConfig(detail.Name, cfg, env); err != nil {
			opts.Respond(false, nil, errInternal("failed to save skill config: "+err.Error()), nil)
			return nil
		}
	}

	opts.Respond(true, map[string]interface{}{
		"ok":      true,
		"id":      id,
		"name":    detail.Name,
		"content": detail.Content,
		"message": "Skill installed successfully",
	}, nil, nil)
	return nil
}

// --- marketplace.getConfig ---

func MarketplaceGetConfigHandler(opts HandlerOpts) error {
	marketCfg := config.GetMarketplaceConfig(envGetter())

	result := map[string]interface{}{
		"url":   defaultMarketplaceURL,
		"token": "",
	}

	if marketCfg != nil {
		if marketCfg.URL != "" {
			result["url"] = marketCfg.URL
		}
		result["token"] = marketCfg.Token
	}

	opts.Respond(true, result, nil, nil)
	return nil
}

// --- marketplace.setConfig ---

func MarketplaceSetConfigHandler(opts HandlerOpts) error {
	url := stringParam(opts.Params, "url", defaultMarketplaceURL)
	token := stringParam(opts.Params, "token", "")

	marketCfg := &config.MarketplaceConfig{
		URL:   url,
		Token: token,
	}

	if err := config.SetMarketplaceConfig(marketCfg, envGetter()); err != nil {
		opts.Respond(false, nil, errInternal("failed to save marketplace config: "+err.Error()), nil)
		return nil
	}

	opts.Respond(true, map[string]interface{}{
		"ok":      true,
		"url":     url,
		"message": "Marketplace configuration saved",
	}, nil, nil)
	return nil
}
