package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"buzz-cli/internal/client"
	"buzz-cli/internal/config"
	"buzz-cli/internal/nostr"
	"buzz-cli/internal/types"
	"github.com/spf13/cobra"
)

const (
	ExitOK       = 0
	ExitInput    = 1
	ExitRelay    = 2
	ExitAuth     = 3
	ExitOther    = 4
	ExitConflict = 5
)

type rootOptions struct {
	ConfigPath string
	RelayURL   string
	Identity   string
	PrivateKey string
	AuthTag    string
	OwnerKey   string
	Format     string
	Stdout     io.Writer
	Stderr     io.Writer
}

type ExitError struct {
	Code    int
	Message string
	Err     error
}

func (e ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return e.Message
}

func Execute() int {
	root, opts := NewRootCommand()
	if err := root.Execute(); err != nil {
		return writeError(opts.stderr(), err)
	}
	return ExitOK
}

func NewRootCommand() (*cobra.Command, *rootOptions) {
	opts := &rootOptions{Format: "json", Stdout: os.Stdout, Stderr: os.Stderr}
	root := &cobra.Command{
		Use:           "buzz",
		Short:         "Buzz relay CLI",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if opts.Format != "json" && opts.Format != "compact" {
				return inputError("format must be json or compact")
			}
			return nil
		},
	}
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return ExitError{Code: ExitInput, Message: err.Error(), Err: err}
	})
	root.PersistentFlags().StringVar(&opts.RelayURL, "relay", "", "relay URL")
	root.PersistentFlags().StringVar(&opts.Identity, "identity", "", "named identity from config")
	root.PersistentFlags().StringVar(&opts.PrivateKey, "key", "", "private key as nsec or 64 hex")
	root.PersistentFlags().StringVar(&opts.PrivateKey, "private-key", "", "private key as nsec or 64 hex")
	root.PersistentFlags().StringVar(&opts.AuthTag, "auth-tag", "", "NIP-OA auth tag JSON")
	root.PersistentFlags().StringVar(&opts.OwnerKey, "owner-key", "", "owner private key as nsec or 64 hex")
	root.PersistentFlags().StringVar(&opts.ConfigPath, "config", "", "config file path")
	root.PersistentFlags().StringVar(&opts.Format, "format", "json", "output format: compact or json")

	root.AddCommand(usersCommand(opts))
	root.AddCommand(channelsCommand(opts))
	root.AddCommand(messagesCommand(opts))
	agents := agentsCommand(opts)
	root.AddCommand(agents)
	root.AddCommand(fleetCommand(opts))

	root.AddCommand(canvasCommand(opts))
	root.AddCommand(reactionsCommand(opts))
	root.AddCommand(emojiCommand(opts))
	root.AddCommand(dmsCommand(opts))
	root.AddCommand(stubGroup("workflows", "list", "get", "create", "update", "delete", "trigger", "runs", "approve"))
	root.AddCommand(feedCommand(opts))
	root.AddCommand(socialCommand(opts))
	root.AddCommand(notesCommand(opts))
	root.AddCommand(reposCommand(opts))
	root.AddCommand(projectsCommand(opts))
	root.AddCommand(patchesCommand(opts))
	root.AddCommand(issuesCommand(opts))
	root.AddCommand(prCommand(opts))
	root.AddCommand(stubGroup("media", "get"))
	root.AddCommand(stubGroup("upload", "file"))
	root.AddCommand(stubGroup("mem", "ls", "get", "hash", "set", "patch", "rm"))
	root.AddCommand(stubGroup("pack", "validate", "inspect"))
	root.AddCommand(stubGroup("moderation", "reports", "resolve", "ban", "unban", "timeout", "untimeout", "restricted", "audit"))
	root.AddCommand(inviteCommand(opts))
	root.AddCommand(settingsCommand(opts))

	return root, opts
}

func usersCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "users", Short: "User profile and presence commands"}

	get := &cobra.Command{
		Use:   "get",
		Short: "Get user profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			pubkey, _ := cmd.Flags().GetString("pubkey")
			name, _ := cmd.Flags().GetString("name")
			owner, _ := cmd.Flags().GetString("owner")
			filter := client.Filter{"kinds": []int{nostr.KindProfile}, "limit": 50}
			authors := compactStrings([]string{pubkey, owner})
			if len(authors) > 0 {
				filter["authors"] = authors
			}
			if name != "" {
				filter["search"] = name
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	get.Flags().String("pubkey", "", "user pubkey")
	get.Flags().String("name", "", "profile search text")
	get.Flags().String("owner", "", "owner pubkey")

	setProfile := &cobra.Command{
		Use:   "set-profile",
		Short: "Publish a kind 0 profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, _ := cmd.Flags().GetString("name")
			displayName, _ := cmd.Flags().GetString("display-name")
			avatar, _ := cmd.Flags().GetString("avatar")
			about, _ := cmd.Flags().GetString("about")
			nip05, _ := cmd.Flags().GetString("nip05")
			if displayName == "" {
				displayName = name
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			var authTags []nostr.Tag
			if resolved.AuthTag != "" {
				tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
				if err != nil {
					return inputWrap("parse auth tag", err)
				}
				authTags = append(authTags, tag)
			}
			event, err := nostr.BuildProfileEvent(keys, displayName, name, avatar, about, nip05, authTags...)
			if err != nil {
				return otherWrap("build profile event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	setProfile.Flags().String("name", "", "profile name")
	setProfile.Flags().String("display-name", "", "profile display name")
	setProfile.Flags().String("avatar", "", "avatar URL")
	setProfile.Flags().String("about", "", "profile about text")
	setProfile.Flags().String("nip05", "", "NIP-05 identifier")

	presence := &cobra.Command{
		Use:   "presence",
		Short: "Query user presence",
		RunE: func(cmd *cobra.Command, args []string) error {
			pubkeys, _ := cmd.Flags().GetStringSlice("pubkeys")
			filter := client.Filter{"kinds": []int{nostr.KindPresence}, "limit": 100}
			if len(pubkeys) > 0 {
				filter["authors"] = pubkeys
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	presence.Flags().StringSlice("pubkeys", nil, "pubkeys to query")

	setPresence := &cobra.Command{
		Use:   "set-presence",
		Short: "Publish user presence",
		RunE: func(cmd *cobra.Command, args []string) error {
			status, err := requiredFlag(cmd, "status")
			if err != nil {
				return err
			}
			if status != "online" && status != "away" && status != "offline" {
				return inputError("status must be online, away, or offline")
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindPresence, keys.PublicHex(), status, nostr.Tags{{"status", status}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign presence event", err)
			}
			return opts.publishWS(cmd.Context(), resolved, keys, event)
		},
	}
	setPresence.Flags().String("status", "", "presence status")
	setStatus := &cobra.Command{
		Use:   "set-status",
		Short: "Publish user status",
		RunE: func(cmd *cobra.Command, args []string) error {
			text, _ := cmd.Flags().GetString("text")
			emoji, _ := cmd.Flags().GetString("emoji")
			clear, _ := cmd.Flags().GetBool("clear")
			tags := nostr.Tags{{"d", "general"}}
			content := strings.TrimSpace(text)
			if !clear && strings.TrimSpace(emoji) != "" {
				tags = append(tags, nostr.Tag{"emoji", strings.TrimSpace(emoji)})
			}
			if clear {
				content = ""
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindStatus, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign status event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	setStatus.Flags().String("text", "", "status text")
	setStatus.Flags().String("emoji", "", "status emoji")
	setStatus.Flags().Bool("clear", false, "clear status")

	cmd.AddCommand(get, setProfile, presence, setPresence, setStatus)
	return cmd
}

func channelsCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "channels", Short: "Channel commands"}

	list := &cobra.Command{
		Use:   "list",
		Short: "List channels",
		RunE: func(cmd *cobra.Command, args []string) error {
			visibility, _ := cmd.Flags().GetString("visibility")
			member, _ := cmd.Flags().GetString("member")
			limit, _ := cmd.Flags().GetInt("limit")
			filter := client.Filter{"kinds": []int{nostr.KindChannelMetadata}, "limit": positiveOr(limit, 100)}
			if visibility != "" {
				filter["#visibility"] = []string{visibility}
			}
			if member != "" {
				filter["#p"] = []string{member}
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	list.Flags().String("visibility", "", "visibility filter")
	list.Flags().String("member", "", "member pubkey")
	list.Flags().Int("limit", 100, "max results")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"kinds": []int{nostr.KindChannelMetadata}, "#h": []string{channel}, "limit": 1}})
		},
	}
	get.Flags().String("channel", "", "channel id")

	create := &cobra.Command{
		Use:   "create",
		Short: "Create a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			name, err := requiredFlag(cmd, "name")
			if err != nil {
				return err
			}
			channelType, _ := cmd.Flags().GetString("type")
			visibility, _ := cmd.Flags().GetString("visibility")
			description, _ := cmd.Flags().GetString("description")
			ttl, _ := cmd.Flags().GetInt64("ttl")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event, err := types.BuildChannelCreateEvent(keys.PublicHex(), name, channelType, visibility, description, ttl)
			if err != nil {
				return inputWrap("build channel event", err)
			}
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign channel event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	create.Flags().String("name", "", "channel name")
	create.Flags().String("type", "", "channel type")
	create.Flags().String("visibility", "", "channel visibility")
	create.Flags().String("description", "", "channel description")
	create.Flags().Int64("ttl", 0, "ttl seconds")
	create.Flags().String("template", "", "template name")
	create.Flags().String("templates-file", "", "templates file")

	join := &cobra.Command{
		Use:   "join",
		Short: "Join a channel",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindNIP29JoinRequest, keys.PublicHex(), "", nostr.Tags{{"h", channel}}, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign join event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	join.Flags().String("channel", "", "channel id")

	addMember := &cobra.Command{
		Use:   "add-member",
		Short: "Add a channel member",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.publishMembership(cmd, nostr.KindNIP29PutUser)
		},
	}
	addMember.Flags().String("channel", "", "channel id")
	addMember.Flags().String("pubkey", "", "member pubkey")
	addMember.Flags().String("role", "", "member role")

	removeMember := &cobra.Command{
		Use:   "remove-member",
		Short: "Remove a channel member",
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.publishMembership(cmd, nostr.KindNIP29RemoveUser)
		},
	}
	removeMember.Flags().String("channel", "", "channel id")
	removeMember.Flags().String("pubkey", "", "member pubkey")

	members := &cobra.Command{
		Use:   "members",
		Short: "List channel members",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"kinds": []int{nostr.KindChannelMembers, nostr.KindNIP29PutUser}, "#h": []string{channel}, "limit": 1000}})
		},
	}
	members.Flags().String("channel", "", "channel id")

	cmd.AddCommand(list, get, create, join, addMember, removeMember, members)
	cmd.AddCommand(stubCommand("search", "Search channels"))
	cmd.AddCommand(stubCommand("update", "Update channel metadata"))
	cmd.AddCommand(stubCommand("topic", "Set channel topic"))
	cmd.AddCommand(stubCommand("purpose", "Set channel purpose"))
	cmd.AddCommand(stubCommand("leave", "Leave a channel"))
	cmd.AddCommand(stubCommand("archive", "Archive a channel"))
	cmd.AddCommand(stubCommand("unarchive", "Unarchive a channel"))
	cmd.AddCommand(stubCommand("delete", "Delete a channel"))
	cmd.AddCommand(stubCommand("set-add-policy", "Set channel add policy"))
	return cmd
}

func messagesCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "messages", Short: "Message commands"}
	send := &cobra.Command{
		Use:   "send",
		Short: "Send a channel message",
		RunE: func(cmd *cobra.Command, args []string) error {
			channel, err := requiredFlag(cmd, "channel")
			if err != nil {
				return err
			}
			content, _ := cmd.Flags().GetString("content")
			file, _ := cmd.Flags().GetString("file")
			if file != "" {
				b, err := os.ReadFile(file)
				if err != nil {
					return inputWrap("read message file", err)
				}
				content = string(b)
			}
			if strings.TrimSpace(content) == "" {
				return inputError("message content is required")
			}
			kind, _ := cmd.Flags().GetInt("kind")
			if kind == 0 {
				kind = nostr.KindChannelMessage
			}
			replyTo, _ := cmd.Flags().GetString("reply-to")
			mentions, _ := cmd.Flags().GetStringSlice("mention")
			tags := nostr.Tags{{"h", channel}}
			if replyTo != "" {
				tags = append(tags, nostr.Tag{"e", replyTo})
			}
			for _, mention := range compactStrings(mentions) {
				tags = append(tags, nostr.Tag{"p", mention})
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(kind, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign message event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	send.Flags().String("channel", "", "channel id")
	send.Flags().String("content", "", "message content")
	send.Flags().Int("kind", nostr.KindChannelMessage, "event kind")
	send.Flags().String("reply-to", "", "parent event id")
	send.Flags().Bool("broadcast", false, "broadcast message")
	send.Flags().String("file", "", "read message content from file")
	send.Flags().StringSlice("mention", nil, "mention pubkey")

	get := &cobra.Command{
		Use:   "get",
		Short: "Get a message",
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := requiredFlag(cmd, "id")
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"ids": []string{id}, "limit": 1}})
		},
	}
	get.Flags().String("id", "", "event id")

	thread := &cobra.Command{
		Use:   "thread",
		Short: "Get a message thread",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := requiredFlag(cmd, "root")
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"kinds": []int{nostr.KindChannelMessage}, "#e": []string{root}, "limit": 200}})
		},
	}
	thread.Flags().String("root", "", "root event id")

	cmd.AddCommand(send, get, thread)
	cmd.AddCommand(stubCommand("send-diff", "Send a diff message"))
	cmd.AddCommand(stubCommand("edit", "Edit a message"))
	cmd.AddCommand(stubCommand("delete", "Delete a message"))
	cmd.AddCommand(stubCommand("search", "Search messages"))
	cmd.AddCommand(stubCommand("vote", "Vote on a message"))
	return cmd
}

type agentCreateOptions struct {
	Name             string
	SystemPrompt     string
	SystemPromptFile string
	Avatar           string
	Runtime          string
	RuntimeArgs      string
	Model            string
	Provider         string
	Channels         []string
	RespondTo        string
	PersonaDir       string
}

type createdAgent struct {
	Name       string   `json:"name"`
	PubKey     string   `json:"pubkey"`
	Nsec       string   `json:"nsec"`
	AuthTag    string   `json:"auth_tag"`
	ConfigPath string   `json:"config_path"`
	EventIDs   []string `json:"event_ids"`
}

func agentsCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "agents", Short: "Managed agent commands"}
	createOpts := &agentCreateOptions{}
	create := &cobra.Command{
		Use:   "create",
		Short: "Create a managed agent",
		RunE: func(cmd *cobra.Command, args []string) error {
			created, err := opts.createAgent(cmd.Context(), *createOpts, false)
			if err != nil {
				return err
			}
			return opts.writeJSON(created)
		},
	}
	addAgentCreateFlags(create, createOpts)

	list := &cobra.Command{
		Use:   "list",
		Short: "List managed agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			resolved, owner, err := opts.resolveOwnerKey(false)
			if err != nil {
				return err
			}
			filter := client.Filter{"kinds": []int{nostr.KindManagedAgent}, "limit": 100}
			if owner != nil {
				filter["authors"] = []string{owner.PublicHex()}
			}
			return opts.queryResolved(cmd.Context(), resolved, nil, []client.Filter{filter})
		},
	}

	get := &cobra.Command{
		Use:   "get <name|pubkey>",
		Short: "Get a managed agent",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := args[0]
			resolved, owner, err := opts.resolveOwnerKey(false)
			if err != nil {
				return err
			}
			if keys, ok := resolved.File.Identities[target]; ok {
				keyPair, err := nostr.ParsePrivateKey(keys)
				if err != nil {
					return inputWrap("parse configured identity", err)
				}
				target = keyPair.PublicHex()
			}
			filter := client.Filter{"kinds": []int{nostr.KindManagedAgent}, "#d": []string{target}, "limit": 1}
			if owner != nil {
				filter["authors"] = []string{owner.PublicHex()}
			}
			return opts.queryResolved(cmd.Context(), resolved, nil, []client.Filter{filter})
		},
	}

	run := &cobra.Command{
		Use:   "run <name|pubkey>",
		Short: "Run an agent runtime",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			detach, _ := cmd.Flags().GetBool("detach")
			pidfile, _ := cmd.Flags().GetString("pidfile")
			command, _ := cmd.Flags().GetString("acp-command")
			harness, _ := cmd.Flags().GetString("harness-command")
			return opts.runAgent(cmd.Context(), args[0], command, harness, detach, pidfile)
		},
	}
	run.Flags().Bool("detach", false, "run in the background")
	run.Flags().String("pidfile", "", "pidfile path")
	run.Flags().String("acp-command", "buzz-acp", "ACP runner command")
	run.Flags().String("harness-command", "", "agent harness command")

	stop := &cobra.Command{
		Use:   "stop",
		Short: "Stop a detached agent runtime",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, _ := cmd.Flags().GetInt("pid")
			pidfile, _ := cmd.Flags().GetString("pidfile")
			return stopProcess(pid, pidfile)
		},
	}
	stop.Flags().Int("pid", 0, "process id")
	stop.Flags().String("pidfile", "", "pidfile path")

	delete := &cobra.Command{
		Use:   "delete <name|pubkey>",
		Short: "Delete a managed agent projection",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.deleteAgent(cmd.Context(), args[0])
		},
	}

	cmd.AddCommand(create, list, get, run, stop, delete, fleetCommand(opts))
	cmd.AddCommand(stubCommand("update", "Update a managed agent"))
	cmd.AddCommand(stubCommand("draft-create", "Create an agent draft"))
	cmd.AddCommand(stubCommand("draft-update", "Update an agent draft"))
	cmd.AddCommand(stubCommand("archive", "Archive an agent"))
	cmd.AddCommand(stubCommand("unarchive", "Unarchive an agent"))
	cmd.AddCommand(stubCommand("archived", "List archived agents"))
	return cmd
}

func fleetCommand(opts *rootOptions) *cobra.Command {
	fleetOpts := &agentCreateOptions{}
	var count int
	var namePrefix string
	var run bool
	var maxConcurrent int
	cmd := &cobra.Command{
		Use:   "fleet",
		Short: "Create a managed agent fleet",
		RunE: func(cmd *cobra.Command, args []string) error {
			if count <= 0 {
				return inputError("count must be greater than zero")
			}
			if strings.TrimSpace(namePrefix) == "" {
				return inputError("name-prefix is required")
			}
			if maxConcurrent <= 0 {
				maxConcurrent = count
			}
			created := make([]createdAgent, 0, count)
			for i := 1; i <= count; i++ {
				current := *fleetOpts
				current.Name = fmt.Sprintf("%s-%d", namePrefix, i)
				if current.PersonaDir != "" {
					personaPath := filepath.Join(current.PersonaDir, current.Name+".txt")
					if b, err := os.ReadFile(personaPath); err == nil {
						current.SystemPrompt = string(b)
						current.SystemPromptFile = ""
					}
				}
				agent, err := opts.createAgent(cmd.Context(), current, true)
				if err != nil {
					return err
				}
				created = append(created, agent)
			}
			result := map[string]any{
				"count":          len(created),
				"max_concurrent": maxConcurrent,
				"agents":         created,
				"run":            run,
			}
			if run {
				result["message"] = "fleet runtime launch is not implemented in this increment"
			}
			return opts.writeJSON(result)
		},
	}
	cmd.Flags().IntVar(&count, "count", 0, "number of agents")
	cmd.Flags().StringVar(&namePrefix, "name-prefix", "", "agent name prefix")
	cmd.Flags().BoolVar(&run, "run", false, "run agents after creation")
	cmd.Flags().IntVar(&maxConcurrent, "max-concurrent", 0, "maximum concurrent runtimes")
	addAgentCreateFlags(cmd, fleetOpts)
	cmd.Flags().Lookup("name").Hidden = true
	return cmd
}

func addAgentCreateFlags(cmd *cobra.Command, opts *agentCreateOptions) {
	cmd.Flags().StringVar(&opts.Name, "name", "", "agent name")
	cmd.Flags().StringVar(&opts.SystemPrompt, "system-prompt", "", "system prompt")
	cmd.Flags().StringVar(&opts.SystemPromptFile, "system-prompt-file", "", "system prompt file")
	cmd.Flags().StringVar(&opts.Avatar, "avatar", "", "avatar URL")
	cmd.Flags().StringVar(&opts.Runtime, "runtime", "", "runtime command")
	cmd.Flags().StringVar(&opts.RuntimeArgs, "runtime-args", "", "runtime args")
	cmd.Flags().StringVar(&opts.Model, "model", "", "model")
	cmd.Flags().StringVar(&opts.Provider, "provider", "", "provider")
	cmd.Flags().StringSliceVar(&opts.Channels, "channels", nil, "channel ids")
	cmd.Flags().StringVar(&opts.RespondTo, "respond-to", types.RespondToOwnerOnly, "respond-to mode")
	cmd.Flags().StringVar(&opts.PersonaDir, "persona-dir", "", "persona directory")
}

func (opts *rootOptions) createAgent(ctx context.Context, input agentCreateOptions, allowDuplicate bool) (createdAgent, error) {
	if strings.TrimSpace(input.Name) == "" {
		return createdAgent{}, inputError("agent name is required")
	}
	prompt := input.SystemPrompt
	if input.SystemPromptFile != "" {
		b, err := os.ReadFile(input.SystemPromptFile)
		if err != nil {
			return createdAgent{}, inputWrap("read system prompt file", err)
		}
		prompt = string(b)
	}
	if strings.TrimSpace(prompt) == "" {
		return createdAgent{}, inputError("system prompt is required")
	}
	resolved, owner, err := opts.resolveOwnerKey(true)
	if err != nil {
		return createdAgent{}, err
	}
	if resolved.RelayURL == "" {
		return createdAgent{}, inputError("relay URL is required")
	}
	if !allowDuplicate {
		if _, exists := resolved.File.Identities[input.Name]; exists {
			return createdAgent{}, ExitError{Code: ExitConflict, Message: "identity already exists"}
		}
	}
	agentKeys, err := nostr.NewKeyPair()
	if err != nil {
		return createdAgent{}, otherWrap("generate agent key", err)
	}
	authTag, err := nostr.MintAuthTag(owner, agentKeys.PublicHex(), "")
	if err != nil {
		return createdAgent{}, authWrap("mint auth tag", err)
	}
	authTagJSON, err := nostr.AuthTagJSON(authTag)
	if err != nil {
		return createdAgent{}, otherWrap("encode auth tag", err)
	}

	events, err := types.BuildManagedAgentCreateEvents(types.ManagedAgentCreateInput{
		AgentPubKey:  agentKeys.PublicHex(),
		OwnerPubKey:  owner.PublicHex(),
		Name:         input.Name,
		SystemPrompt: prompt,
		AvatarURL:    input.Avatar,
		Runtime:      combineRuntime(input.Runtime, input.RuntimeArgs),
		Model:        input.Model,
		Provider:     input.Provider,
		Parallelism:  1,
		RespondTo:    input.RespondTo,
		Channels:     input.Channels,
		AuthTag:      authTag,
	})
	if err != nil {
		return createdAgent{}, inputWrap("build managed agent events", err)
	}

	signed := []nostr.Event{events.Profile, events.Persona, events.ManagedAgent}
	signed = append(signed, events.ChannelMemberships...)
	if err := signed[0].Sign(agentKeys); err != nil {
		return createdAgent{}, otherWrap("sign profile event", err)
	}
	for i := 1; i < len(signed); i++ {
		if err := signed[i].Sign(owner); err != nil {
			return createdAgent{}, otherWrap("sign owner event", err)
		}
	}

	relayClient := client.New(resolved.RelayURL, owner, nil)
	eventIDs := make([]string, 0, len(signed))
	for _, event := range signed {
		if _, err := relayClient.PostEvent(ctx, event); err != nil {
			return createdAgent{}, ExitError{Code: ExitRelay, Message: "publish event failed", Err: err}
		}
		eventIDs = append(eventIDs, event.ID)
	}

	nsec, err := agentKeys.Nsec()
	if err != nil {
		return createdAgent{}, otherWrap("encode agent nsec", err)
	}
	file := resolved.File
	if file.RelayURL == "" {
		file.RelayURL = resolved.RelayURL
	}
	if file.OwnerKey == "" && resolved.OwnerKey != "" {
		file.OwnerKey = resolved.OwnerKey
	}
	if err := file.SaveIdentity(resolved.ConfigPath, input.Name, nsec, authTagJSON); err != nil {
		return createdAgent{}, otherWrap("save identity", err)
	}
	return createdAgent{
		Name:       input.Name,
		PubKey:     agentKeys.PublicHex(),
		Nsec:       nsec,
		AuthTag:    authTagJSON,
		ConfigPath: resolved.ConfigPath,
		EventIDs:   eventIDs,
	}, nil
}

func (opts *rootOptions) runAgent(ctx context.Context, target, acpCommand, harnessCommand string, detach bool, pidfile string) error {
	resolved, err := config.Resolve(config.Options{
		ConfigPath: opts.ConfigPath,
		RelayURL:   opts.RelayURL,
		Identity:   target,
		PrivateKey: opts.PrivateKey,
		AuthTag:    opts.AuthTag,
		OwnerKey:   opts.OwnerKey,
	})
	if err != nil {
		return otherWrap("resolve config", err)
	}
	if resolved.PrivateKey == "" {
		if key, ok := resolved.File.Identities[target]; ok {
			resolved.PrivateKey = key
			resolved.AuthTag = resolved.File.AuthTags[target]
		}
	}
	if resolved.PrivateKey == "" {
		return inputError("agent identity key is required")
	}
	if resolved.RelayURL == "" {
		return inputError("relay URL is required")
	}
	if acpCommand == "" {
		return inputError("acp-command is required")
	}
	env := os.Environ()
	env = append(env,
		config.EnvPrivateKey+"="+resolved.PrivateKey,
		config.EnvRelayURL+"="+resolved.RelayURL,
		config.EnvAuthTag+"="+resolved.AuthTag,
		"BUZZ_ACP_AGENT_COMMAND="+harnessCommand,
	)
	proc := exec.CommandContext(ctx, acpCommand)
	proc.Env = env
	if !detach {
		proc.Stdout = opts.stdout()
		proc.Stderr = opts.stderr()
		proc.Stdin = os.Stdin
		if err := proc.Run(); err != nil {
			return ExitError{Code: ExitOther, Message: "agent runtime exited with error", Err: err}
		}
		return nil
	}
	if err := proc.Start(); err != nil {
		return ExitError{Code: ExitOther, Message: "start detached agent runtime failed", Err: err}
	}
	if pidfile != "" {
		if err := os.MkdirAll(filepath.Dir(pidfile), 0o700); err != nil {
			return otherWrap("create pidfile directory", err)
		}
		if err := os.WriteFile(pidfile, []byte(strconv.Itoa(proc.Process.Pid)+"\n"), 0o600); err != nil {
			return otherWrap("write pidfile", err)
		}
	}
	_ = proc.Process.Release()
	return opts.writeJSON(map[string]any{"pid": proc.Process.Pid, "pidfile": pidfile})
}

func (opts *rootOptions) deleteAgent(ctx context.Context, target string) error {
	resolved, owner, err := opts.resolveOwnerKey(true)
	if err != nil {
		return err
	}
	if resolved.RelayURL == "" {
		return inputError("relay URL is required")
	}
	if key, ok := resolved.File.Identities[target]; ok {
		agent, err := nostr.ParsePrivateKey(key)
		if err != nil {
			return inputWrap("parse configured identity", err)
		}
		target = agent.PublicHex()
	}
	event := nostr.NewUnsignedEvent(
		nostr.KindDeletion,
		owner.PublicHex(),
		"",
		nostr.Tags{{"a", fmt.Sprintf("%d:%s:%s", nostr.KindManagedAgent, owner.PublicHex(), target)}},
		0,
	)
	if err := event.Sign(owner); err != nil {
		return otherWrap("sign delete event", err)
	}
	return opts.publish(ctx, resolved, owner, event)
}

func stopProcess(pid int, pidfile string) error {
	if pid == 0 && pidfile != "" {
		b, err := os.ReadFile(pidfile)
		if err != nil {
			return inputWrap("read pidfile", err)
		}
		parsed, err := strconv.Atoi(strings.TrimSpace(string(b)))
		if err != nil {
			return inputWrap("parse pidfile", err)
		}
		pid = parsed
	}
	if pid <= 0 {
		return inputError("pid or pidfile is required")
	}
	process, err := os.FindProcess(pid)
	if err != nil {
		return otherWrap("find process", err)
	}
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return otherWrap("signal process", err)
	}
	return nil
}

func (opts *rootOptions) publishMembership(cmd *cobra.Command, kind int) error {
	channel, err := requiredFlag(cmd, "channel")
	if err != nil {
		return err
	}
	pubkey, err := requiredFlag(cmd, "pubkey")
	if err != nil {
		return err
	}
	resolved, keys, err := opts.resolveKeys(true)
	if err != nil {
		return err
	}
	tags := nostr.Tags{{"h", channel}, {"p", strings.ToLower(pubkey)}}
	if kind == nostr.KindNIP29PutUser {
		role, _ := cmd.Flags().GetString("role")
		if role != "" {
			tags = append(tags, nostr.Tag{"role", role})
		}
	}
	event := nostr.NewUnsignedEvent(kind, keys.PublicHex(), "", tags, 0)
	if err := event.Sign(keys); err != nil {
		return otherWrap("sign membership event", err)
	}
	return opts.publish(cmd.Context(), resolved, keys, event)
}

func (opts *rootOptions) query(ctx context.Context, filters []client.Filter) error {
	resolved, err := config.Resolve(config.Options{
		ConfigPath: opts.ConfigPath,
		RelayURL:   opts.RelayURL,
		Identity:   opts.Identity,
		PrivateKey: opts.PrivateKey,
		AuthTag:    opts.AuthTag,
		OwnerKey:   opts.OwnerKey,
	})
	if err != nil {
		return otherWrap("resolve config", err)
	}
	return opts.queryResolved(ctx, resolved, nil, filters)
}

func (opts *rootOptions) queryResolved(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, filters []client.Filter) error {
	raw, err := opts.fetchQuery(ctx, resolved, keys, filters)
	if err != nil {
		return err
	}
	return opts.writeRawJSON(raw)
}

func (opts *rootOptions) fetchQuery(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, filters []client.Filter) (json.RawMessage, error) {
	if resolved.RelayURL == "" {
		return nil, inputError("relay URL is required")
	}
	tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
	if err != nil {
		return nil, inputWrap("parse auth tag", err)
	}
	relayClient := client.New(resolved.RelayURL, keys, tag)
	raw, err := relayClient.Query(ctx, filters)
	if err != nil {
		return nil, ExitError{Code: ExitRelay, Message: "query relay failed", Err: err}
	}
	return raw, nil
}

func (opts *rootOptions) publish(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event) error {
	if resolved.RelayURL == "" {
		return inputError("relay URL is required")
	}
	tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
	if err != nil {
		return inputWrap("parse auth tag", err)
	}
	relayClient := client.New(resolved.RelayURL, keys, tag)
	raw, err := relayClient.PostEvent(ctx, event)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "publish event failed", Err: err}
	}
	if len(raw) == 0 {
		return opts.writeJSON(map[string]any{"ok": true, "event_id": event.ID})
	}
	return opts.writeRawJSON(raw)
}

func (opts *rootOptions) publishWS(ctx context.Context, resolved config.Resolved, keys *nostr.KeyPair, event nostr.Event) error {
	if resolved.RelayURL == "" {
		return inputError("relay URL is required")
	}
	tag, err := nostr.ParseAuthTagJSON(resolved.AuthTag)
	if err != nil {
		return inputWrap("parse auth tag", err)
	}
	ws, err := client.DialWS(ctx, resolved.RelayURL, keys, tag)
	if err != nil {
		return ExitError{Code: ExitRelay, Message: "connect relay websocket failed", Err: err}
	}
	defer ws.Close(1000, "done")
	if err := ws.Publish(ctx, event); err != nil {
		return ExitError{Code: ExitRelay, Message: "publish websocket event failed", Err: err}
	}
	return opts.writeJSON(map[string]any{"ok": true, "event_id": event.ID})
}

func (opts *rootOptions) resolveKeys(required bool) (config.Resolved, *nostr.KeyPair, error) {
	resolved, err := config.Resolve(config.Options{
		ConfigPath: opts.ConfigPath,
		RelayURL:   opts.RelayURL,
		Identity:   opts.Identity,
		PrivateKey: opts.PrivateKey,
		AuthTag:    opts.AuthTag,
		OwnerKey:   opts.OwnerKey,
	})
	if err != nil {
		return config.Resolved{}, nil, otherWrap("resolve config", err)
	}
	if resolved.PrivateKey == "" {
		if required {
			return config.Resolved{}, nil, inputError("private key is required")
		}
		return resolved, nil, nil
	}
	keys, err := nostr.ParsePrivateKey(resolved.PrivateKey)
	if err != nil {
		return config.Resolved{}, nil, inputWrap("parse private key", err)
	}
	return resolved, keys, nil
}

func (opts *rootOptions) resolveOwnerKey(required bool) (config.Resolved, *nostr.KeyPair, error) {
	resolved, err := config.Resolve(config.Options{
		ConfigPath: opts.ConfigPath,
		RelayURL:   opts.RelayURL,
		Identity:   opts.Identity,
		PrivateKey: opts.PrivateKey,
		AuthTag:    opts.AuthTag,
		OwnerKey:   opts.OwnerKey,
	})
	if err != nil {
		return config.Resolved{}, nil, otherWrap("resolve config", err)
	}
	if resolved.OwnerKey == "" {
		if required {
			return config.Resolved{}, nil, inputError("owner key is required")
		}
		return resolved, nil, nil
	}
	keys, err := nostr.ParsePrivateKey(resolved.OwnerKey)
	if err != nil {
		return config.Resolved{}, nil, inputWrap("parse owner key", err)
	}
	return resolved, keys, nil
}

func (opts *rootOptions) writeJSON(value any) error {
	if opts.Format == "compact" {
		switch v := value.(type) {
		case string:
			_, err := fmt.Fprintln(opts.stdout(), v)
			return err
		}
	}
	enc := json.NewEncoder(opts.stdout())
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func (opts *rootOptions) writeRawJSON(raw json.RawMessage) error {
	if len(raw) == 0 {
		return opts.writeJSON(map[string]any{"ok": true})
	}
	if opts.Format == "compact" {
		var value any
		if err := json.Unmarshal(raw, &value); err == nil {
			return opts.writeJSON(value)
		}
	}
	_, err := opts.stdout().Write(append(raw, '\n'))
	return err
}

func (opts *rootOptions) stdout() io.Writer {
	if opts.Stdout == nil {
		return os.Stdout
	}
	return opts.Stdout
}

func (opts *rootOptions) stderr() io.Writer {
	if opts.Stderr == nil {
		return os.Stderr
	}
	return opts.Stderr
}

func stubGroup(name string, subcommands ...string) *cobra.Command {
	cmd := &cobra.Command{Use: name}
	for _, sub := range subcommands {
		cmd.AddCommand(stubCommand(sub, ""))
	}
	return cmd
}

func stubCommand(name, short string) *cobra.Command {
	return &cobra.Command{
		Use:   name,
		Short: short,
		RunE: func(cmd *cobra.Command, args []string) error {
			return ExitError{
				Code:    ExitOther,
				Message: fmt.Sprintf("%s is not implemented in this increment", cmd.CommandPath()),
			}
		},
	}
}

func requiredFlag(cmd *cobra.Command, name string) (string, error) {
	value, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", inputWrap("read flag "+name, err)
	}
	if strings.TrimSpace(value) == "" {
		return "", inputError("--" + name + " is required")
	}
	return strings.TrimSpace(value), nil
}

func compactStrings(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		for _, split := range strings.Split(value, ",") {
			split = strings.TrimSpace(split)
			if split != "" {
				out = append(out, split)
			}
		}
	}
	return out
}

func positiveOr(value, fallback int) int {
	if value > 0 {
		return value
	}
	return fallback
}

func combineRuntime(runtime, args string) string {
	runtime = strings.TrimSpace(runtime)
	args = strings.TrimSpace(args)
	if args == "" {
		return runtime
	}
	if runtime == "" {
		return args
	}
	return runtime + " " + args
}

func inputError(message string) error {
	return ExitError{Code: ExitInput, Message: message}
}

func inputWrap(message string, err error) error {
	return ExitError{Code: ExitInput, Message: message, Err: fmt.Errorf("%s: %w", message, err)}
}

func authWrap(message string, err error) error {
	return ExitError{Code: ExitAuth, Message: message, Err: fmt.Errorf("%s: %w", message, err)}
}

func otherWrap(message string, err error) error {
	return ExitError{Code: ExitOther, Message: message, Err: fmt.Errorf("%s: %w", message, err)}
}

func writeError(stderr io.Writer, err error) int {
	code := ExitOther
	message := err.Error()
	var exitErr ExitError
	if errors.As(err, &exitErr) {
		code = exitErr.Code
		if exitErr.Message != "" {
			message = exitErr.Message
		}
	}
	if stderr == nil {
		stderr = os.Stderr
	}
	_ = json.NewEncoder(stderr).Encode(map[string]string{
		"error":   "error",
		"message": message,
	})
	return code
}
