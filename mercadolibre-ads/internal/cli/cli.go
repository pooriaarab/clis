package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"mercadolibre-ads-pp-cli/internal/client"
	"mercadolibre-ads-pp-cli/internal/config"
)

type spec struct {
	Name, Display, Env, BaseURL, Mode                                            string
	Accounts, Campaigns, CampaignCreate, Audiences, AudienceCreate, Report, Todo string
}

var platform = spec{
	Name: "mercadolibre-ads", Display: "Mercado Ads", Env: "MERCADOLIBRE_ADS", BaseURL: "https://api.mercadolibre.com", Mode: "stub",
	Accounts: "", Campaigns: "", CampaignCreate: "", Audiences: "", AudienceCreate: "", Report: "https://developers.mercadolibre.com/|https://developers.mercadolibre.com/en/authentication-and-authorization", Todo: "undefined",
}

func Execute() error { return root().Execute() }

func root() *cobra.Command {
	cfg := config.Load(platform.Env, platform.BaseURL, platform.Name)
	c := client.New(cfg)
	root := &cobra.Command{Use: platform.Name + "-pp-cli", Short: platform.Display + " command line client"}
	root.AddCommand(doctor(cfg), auth(cfg))
	if platform.Mode == "api" {
		root.AddCommand(accounts(c), campaigns(c), audiences(c), reporting(c))
	}
	return root
}

func doctor(cfg *config.Config) *cobra.Command {
	return &cobra.Command{Use: "doctor", Short: "Show local configuration", RunE: func(cmd *cobra.Command, _ []string) error {
		return emit(map[string]any{"platform": platform.Display, "base_url": cfg.BaseURL, "token_configured": cfg.HasToken(), "mode": platform.Mode, "todo": platform.Todo})
	}}
}

func auth(cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{Use: "auth", Short: "Manage the access token"}
	cmd.AddCommand(&cobra.Command{Use: "status", RunE: func(*cobra.Command, []string) error {
		return emit(map[string]any{"token_configured": cfg.HasToken(), "env": cfg.Env + "_ACCESS_TOKEN"})
	}})
	cmd.AddCommand(&cobra.Command{Use: "login", Short: "Show token setup instructions", RunE: func(*cobra.Command, []string) error {
		return emit(map[string]any{"message": "Create or approve an app in the official console, then set the returned access token.", "environment": cfg.Env + "_ACCESS_TOKEN", "next": platform.Todo})
	}})
	cmd.AddCommand(&cobra.Command{Use: "set-token TOKEN", Args: cobra.ExactArgs(1), RunE: func(_ *cobra.Command, args []string) error {
		if err := config.SaveToken(platform.Name, args[0]); err != nil {
			return err
		}
		return emit(map[string]any{"saved": true, "path": "~/.config/" + platform.Name + "/token"})
	}})
	return cmd
}

func accounts(c *client.Client) *cobra.Command {
	return &cobra.Command{Use: "accounts list", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, _ []string) error {
		if platform.Accounts == "" {
			return errors.New(platform.Todo)
		}
		return call(c, cmd, "GET", platform.Accounts, nil, nil)
	}}
}

func campaigns(c *client.Client) *cobra.Command {
	group := &cobra.Command{Use: "campaigns"}
	group.AddCommand(&cobra.Command{Use: "list [ACCOUNT]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		path, err := resolvePath(platform.Campaigns, args)
		if err != nil {
			return err
		}
		return call(c, cmd, "GET", path, nil, nil)
	}})
	create := &cobra.Command{Use: "create [ACCOUNT]", Args: cobra.MaximumNArgs(1)}
	var body string
	create.Flags().StringVar(&body, "body", "", "JSON file path, or - for stdin")
	create.RunE = func(cmd *cobra.Command, args []string) error {
		path, err := resolvePath(platform.CampaignCreate, args)
		if err != nil {
			return err
		}
		data, err := readBody(body)
		if err != nil {
			return err
		}
		return call(c, cmd, "POST", path, data, nil)
	}
	group.AddCommand(create)
	return group
}

func audiences(c *client.Client) *cobra.Command {
	group := &cobra.Command{Use: "audiences"}
	group.AddCommand(&cobra.Command{Use: "list [ACCOUNT]", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		path, err := resolvePath(platform.Audiences, args)
		if err != nil {
			return err
		}
		return call(c, cmd, "GET", path, nil, nil)
	}})
	upload := &cobra.Command{Use: "upload [ACCOUNT]", Args: cobra.MaximumNArgs(1)}
	var body string
	upload.Flags().StringVar(&body, "body", "", "JSON file path, or - for stdin")
	upload.RunE = func(cmd *cobra.Command, args []string) error {
		path, err := resolvePath(platform.AudienceCreate, args)
		if err != nil {
			return err
		}
		data, err := readBody(body)
		if err != nil {
			return err
		}
		return call(c, cmd, "POST", path, data, nil)
	}
	group.AddCommand(upload)
	return group
}

func reporting(c *client.Client) *cobra.Command {
	cmd := &cobra.Command{Use: "reporting get [ACCOUNT]", Args: cobra.MaximumNArgs(1)}
	var body string
	cmd.Flags().StringVar(&body, "body", "", "JSON file path, or - for stdin")
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		path, err := resolvePath(platform.Report, args)
		if err != nil {
			return err
		}
		method := "GET"
		var data []byte
		if body != "" {
			method = "POST"
			data, err = readBody(body)
			if err != nil {
				return err
			}
		}
		return call(c, cmd, method, path, data, nil)
	}
	return cmd
}

func resolvePath(template string, args []string) (string, error) {
	if template == "" {
		return "", errors.New(platform.Todo)
	}
	path := template
	if strings.Contains(path, "{account}") {
		if len(args) != 1 || strings.TrimSpace(args[0]) == "" {
			return "", errors.New("account is required")
		}
		path = strings.ReplaceAll(path, "{account}", args[0])
	}
	return path, nil
}

func call(c *client.Client, cmd *cobra.Command, method, path string, body []byte, headers map[string]string) error {
	if !c.Config.HasToken() {
		return fmt.Errorf("no access token; run %s auth set-token TOKEN or set %s_ACCESS_TOKEN", platform.Name+"-pp-cli", platform.Env)
	}
	data, err := c.Request(cmd.Context(), method, path, body, "application/json", headers)
	if err != nil {
		return err
	}
	return emitJSON(data)
}

func readBody(path string) ([]byte, error) {
	if path == "" {
		return nil, errors.New("--body is required")
	}
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func emit(value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func emitJSON(data []byte) error {
	var value any
	if json.Unmarshal(data, &value) == nil {
		return emit(value)
	}
	fmt.Println(string(data))
	return nil
}
