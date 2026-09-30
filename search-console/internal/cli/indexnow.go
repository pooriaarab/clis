package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/pooriaarab/clis/search-console/internal/config"
	"github.com/pooriaarab/clis/search-console/internal/indexnow"
)

func indexnowCmd(env *Env) *cobra.Command {
	cmd := &cobra.Command{Use: "indexnow", Short: "Tell IndexNow about new and changed URLs"}
	cmd.AddCommand(indexnowSubmitCmd(env))
	return cmd
}

func indexnowSubmitCmd(env *Env) *cobra.Command {
	var urlsFile, keyDir, keyLocation string
	cmd := &cobra.Command{
		Use:   "submit <domain>",
		Short: "Post URLs to IndexNow in batches of up to 10,000",
		Long: `Read the key for the domain from the config directory, or make one. Write the key
file <key>.txt to --key-dir and print its path. You must serve that file at
https://<domain>/<key>.txt. The command checks that the file is reachable before it
posts. Give the URLs with --urls, one per line. Every URL must be on <domain>.`,
		Args: cobra.ExactArgs(1),
		RunE: run(func(args []string) error {
			domain, err := normalizeDomain(args[0])
			if err != nil {
				return err
			}
			if urlsFile == "" {
				return &ExitError{Code: ExitUsage, Err: errors.New("give the URLs with --urls <file>")}
			}
			ctx := context.Background()
			urls, err := indexnowURLs(domain, urlsFile)
			if err != nil {
				return err
			}
			key, keyPath, err := indexnowKey(env, domain, keyDir)
			if err != nil {
				return err
			}
			if keyLocation == "" {
				keyLocation = "https://" + domain + "/" + key + ".txt"
			}
			env.P.Warnf("Key file: %s\nServe it at: %s", keyPath, keyLocation)
			api := &indexnow.API{HTTP: env.Client, Base: envOr(env.Getenv, "INDEXNOW_API_BASE", indexnow.DefaultBase)}
			if err := api.CheckKeyFile(ctx, keyLocation, key); err != nil {
				return fmt.Errorf("the key file is not reachable: %w (upload %s, then run the command again)", err, filepath.Base(keyPath))
			}
			batches := indexnow.Batches(urls)
			var statuses []int
			for i, batch := range batches {
				st, err := api.Submit(ctx, domain, key, keyLocation, batch)
				if err != nil {
					return fmt.Errorf("batch %d of %d failed after %d batch(es) were sent: %w", i+1, len(batches), len(statuses), err)
				}
				statuses = append(statuses, st)
			}
			return env.Emit(map[string]any{
				"ok": true, "domain": domain, "key_file": keyPath, "key_location": keyLocation,
				"urls": len(urls), "batches": len(statuses), "statuses": statuses,
			}, func() {
				verb := "Sent"
				if env.DryRun {
					verb = "Would send"
				}
				env.P.Linef("%s %d URL(s) to IndexNow in %d batch(es).", verb, len(urls), len(statuses))
			})
		}),
	}
	cmd.Flags().StringVar(&urlsFile, "urls", "", "file with one URL per line")
	cmd.Flags().StringVar(&keyDir, "key-dir", ".", "directory where the key file is written")
	cmd.Flags().StringVar(&keyLocation, "key-location", "", "URL of the key file (default https://<domain>/<key>.txt)")
	return cmd
}

// indexnowURLs reads the URLs from a file. It drops blank lines, comments and
// duplicates. A URL that is not on the domain is a usage error, because
// IndexNow would refuse the whole batch.
func indexnowURLs(domain, file string) ([]string, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var raw []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		if line := strings.TrimSpace(sc.Text()); line != "" && !strings.HasPrefix(line, "#") {
			raw = append(raw, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var urls, foreign []string
	for _, r := range raw {
		u, err := url.Parse(r)
		switch {
		case seen[r]:
		case err != nil || (u.Scheme != "http" && u.Scheme != "https") || !strings.EqualFold(u.Host, domain):
			foreign = append(foreign, r)
		default:
			seen[r] = true
			urls = append(urls, r)
		}
	}
	if len(foreign) > 0 {
		return nil, &ExitError{Code: ExitUsage, Err: fmt.Errorf("%d URL(s) are not http(s) URLs on %s, for example %s", len(foreign), domain, strings.Join(foreign[:min(3, len(foreign))], ", "))}
	}
	if len(urls) == 0 {
		return nil, &ExitError{Code: ExitUsage, Err: errors.New("there are no URLs to send")}
	}
	return urls, nil
}

// indexnowKey reads the saved key for the domain, or makes and saves one. It
// writes <key>.txt into dir. A dry run saves and writes nothing.
func indexnowKey(env *Env, domain, dir string) (key, path string, err error) {
	cfg, err := config.Dir(env.Getenv)
	if err != nil {
		return "", "", err
	}
	var saved struct {
		Key string `json:"key"`
	}
	if _, err := config.Load(cfg, "indexnow-"+domain, &saved); err != nil {
		return "", "", err
	}
	if key = saved.Key; key == "" {
		if key, err = indexnow.NewKey(); err != nil {
			return "", "", err
		}
		if !env.DryRun {
			if err := config.Save(cfg, "indexnow-"+domain, map[string]string{"key": key}); err != nil {
				return "", "", err
			}
		}
	}
	path = filepath.Join(dir, key+".txt")
	if env.DryRun {
		return key, path, nil
	}
	return key, path, os.WriteFile(path, []byte(key), 0o644)
}
