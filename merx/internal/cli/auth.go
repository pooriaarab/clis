// merx login, merx logout, and merx auth status.
// Credentials come from MERX_USERNAME / MERX_PASSWORD only.
package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"merx-cli/internal/httpx"
	"merx-cli/internal/session"
)

func sessionPath() (string, error) {
	dir, err := cacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, session.JarFile), nil
}

func openSession() (*session.Jar, *httpx.Client, string, error) {
	path, err := sessionPath()
	if err != nil {
		return nil, nil, "", err
	}
	j, err := session.Open(path)
	if err != nil {
		return nil, nil, "", err
	}
	c, err := session.Bind(j)
	if err != nil {
		return nil, nil, "", err
	}
	return j, c, path, nil
}

func loginCmd() *cobra.Command {
	return &cobra.Command{Use: "login", Short: "Log in to MERX with SAML SSO", RunE: func(*cobra.Command, []string) error {
		user, pass := os.Getenv("MERX_USERNAME"), os.Getenv("MERX_PASSWORD")
		if user == "" || pass == "" {
			return fmt.Errorf("missing credentials: export MERX_USERNAME and MERX_PASSWORD")
		}
		j, c, _, err := openSession()
		if err != nil {
			return err
		}
		if err := session.Login(c, session.Production, user, pass); err != nil {
			return err
		}
		if err := j.Save(); err != nil {
			return err
		}
		if flagJSON {
			return emit(map[string]any{"authenticated": true})
		}
		fmt.Println("Login successful.")
		return nil
	}}
}

func logoutCmd() *cobra.Command {
	return &cobra.Command{Use: "logout", Short: "End the MERX server session and drop the local jar", RunE: func(*cobra.Command, []string) error {
		_, c, path, err := openSession()
		if err != nil {
			return err
		}
		reqErr := session.Logout(c, session.Production.Portal)
		if rmErr := os.Remove(path); rmErr != nil && !os.IsNotExist(rmErr) {
			return rmErr
		}
		if reqErr != nil {
			return fmt.Errorf("portal logout request failed (local session cleared): %w", reqErr)
		}
		if flagJSON {
			return emit(map[string]any{"cleared": true})
		}
		fmt.Println("Logged out.")
		return nil
	}}
}

func authCmd() *cobra.Command {
	g := &cobra.Command{Use: "auth", Short: "Show MERX authentication state", RunE: func(cmd *cobra.Command, _ []string) error {
		return cmd.Help()
	}}
	g.AddCommand(&cobra.Command{Use: "status", Short: "Show authentication status", RunE: func(*cobra.Command, []string) error {
		path, err := sessionPath()
		if err != nil {
			return err
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return reportAuth(false, path)
		}
		_, c, path, err := openSession()
		if err != nil {
			return err
		}
		ok, err := session.Valid(c, session.Production.Portal)
		if err != nil {
			return err
		}
		return reportAuth(ok, path)
	}})
	return g
}

func reportAuth(ok bool, path string) error {
	if flagJSON {
		if err := emit(map[string]any{"authenticated": ok, "session_file": path}); err != nil {
			return err
		}
	} else if ok {
		fmt.Println("Authenticated: MERX session is valid.")
	} else {
		fmt.Println("Not authenticated: no valid MERX session.")
	}
	if !ok {
		return fmt.Errorf("not authenticated")
	}
	return nil
}
