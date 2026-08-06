package cli

import (
	"encoding/json"
	"fmt"

	"buzz-cli/internal/client"
	"buzz-cli/internal/nostr"
	"github.com/spf13/cobra"
)

func socialCommand(opts *rootOptions) *cobra.Command {
	cmd := &cobra.Command{Use: "social", Short: "Social Nostr commands"}

	publish := &cobra.Command{
		Use:   "publish",
		Short: "Publish a text note",
		RunE: func(cmd *cobra.Command, args []string) error {
			if !cmd.Flags().Changed("content") {
				return inputError("--content is required")
			}
			content, _ := cmd.Flags().GetString("content")
			if len(content) > 64*1024 {
				return inputError("content must be 64KiB or smaller")
			}
			replyTo, _ := cmd.Flags().GetString("reply-to")
			tags := nostr.Tags{}
			if replyTo != "" {
				var err error
				replyTo, err = validateHex64(replyTo)
				if err != nil {
					return err
				}
				tags = append(tags, nostr.Tag{"e", replyTo, "", "reply"})
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindTextNote, keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign text note event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	publish.Flags().String("content", "", "note content")
	publish.Flags().String("reply-to", "", "event id to reply to")

	setContacts := &cobra.Command{
		Use:   "set-contacts",
		Short: "Publish a contact list",
		RunE: func(cmd *cobra.Command, args []string) error {
			contactsJSON, err := requiredFlag(cmd, "contacts")
			if err != nil {
				return err
			}
			var contacts []struct {
				Pubkey   string  `json:"pubkey"`
				RelayURL *string `json:"relay_url"`
				Petname  *string `json:"petname"`
			}
			if err := json.Unmarshal([]byte(contactsJSON), &contacts); err != nil {
				return inputWrap("parse contacts JSON", err)
			}
			if len(contacts) > 10000 {
				return inputError("contacts must contain 10000 entries or fewer")
			}
			seen := map[string]struct{}{}
			tags := nostr.Tags{}
			for i, contact := range contacts {
				pubkey, err := validateHex64(contact.Pubkey)
				if err != nil {
					return inputError(fmt.Sprintf("contacts[%d].pubkey must be 64 hex characters", i))
				}
				if _, ok := seen[pubkey]; ok {
					continue
				}
				seen[pubkey] = struct{}{}
				relayURL := ""
				if contact.RelayURL != nil {
					relayURL = *contact.RelayURL
				}
				petname := ""
				if contact.Petname != nil {
					petname = *contact.Petname
				}
				tags = append(tags, nostr.Tag{"p", pubkey, relayURL, petname})
			}
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(nostr.KindContactList, keys.PublicHex(), "", tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign contact list event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	setContacts.Flags().String("contacts", "", "contacts JSON")

	eventCmd := &cobra.Command{
		Use:   "event",
		Short: "Get an event",
		RunE: func(cmd *cobra.Command, args []string) error {
			eventID, err := requiredFlag(cmd, "event")
			if err != nil {
				return err
			}
			eventID, err = validateHex64(eventID)
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"ids": []string{eventID}}})
		},
	}
	eventCmd.Flags().String("event", "", "event id")

	notes := &cobra.Command{
		Use:   "notes",
		Short: "List text notes by pubkey",
		RunE: func(cmd *cobra.Command, args []string) error {
			pubkey, err := requiredFlag(cmd, "pubkey")
			if err != nil {
				return err
			}
			pubkey, err = validateHex64(pubkey)
			if err != nil {
				return err
			}
			limit, _ := cmd.Flags().GetInt("limit")
			if limit <= 0 {
				return inputError("limit must be greater than zero")
			}
			if limit > 100 {
				limit = 100
			}
			before, _ := cmd.Flags().GetInt64("before")
			beforeID, _ := cmd.Flags().GetString("before-id")
			filter := client.Filter{"kinds": []int{nostr.KindTextNote}, "authors": []string{pubkey}, "limit": limit}
			if cmd.Flags().Changed("before") {
				filter["until"] = before
			}
			if beforeID != "" {
				beforeID, err = validateHex64(beforeID)
				if err != nil {
					return err
				}
				filter["before_id"] = beforeID
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	notes.Flags().String("pubkey", "", "author pubkey")
	notes.Flags().Int("limit", 50, "max results")
	notes.Flags().Int64("before", 0, "maximum created_at timestamp")
	notes.Flags().String("before-id", "", "relay pagination cursor")

	contacts := &cobra.Command{
		Use:   "contacts",
		Short: "Get a contact list",
		RunE: func(cmd *cobra.Command, args []string) error {
			pubkey, err := requiredFlag(cmd, "pubkey")
			if err != nil {
				return err
			}
			pubkey, err = validateHex64(pubkey)
			if err != nil {
				return err
			}
			return opts.query(cmd.Context(), []client.Filter{{"kinds": []int{nostr.KindContactList}, "authors": []string{pubkey}, "limit": 1}})
		},
	}
	contacts.Flags().String("pubkey", "", "pubkey")

	setList := &cobra.Command{
		Use:   "set-list",
		Short: "Publish a Nostr list",
		RunE: func(cmd *cobra.Command, args []string) error {
			kind, _ := cmd.Flags().GetUint16("kind")
			if err := validateSocialKind(uint32(kind)); err != nil {
				return err
			}
			tagsJSON, err := requiredFlag(cmd, "tags")
			if err != nil {
				return err
			}
			var tags nostr.Tags
			if err := json.Unmarshal([]byte(tagsJSON), &tags); err != nil {
				return inputWrap("parse tags JSON", err)
			}
			if isParameterizedSocialKind(uint32(kind)) && !tagsContainD(tags) {
				return inputError(fmt.Sprintf("kind %d is parameterized replaceable and requires a d tag", kind))
			}
			content, _ := cmd.Flags().GetString("content")
			resolved, keys, err := opts.resolveKeys(true)
			if err != nil {
				return err
			}
			event := nostr.NewUnsignedEvent(int(kind), keys.PublicHex(), content, tags, 0)
			if err := event.Sign(keys); err != nil {
				return otherWrap("sign list event", err)
			}
			return opts.publish(cmd.Context(), resolved, keys, event)
		},
	}
	setList.Flags().Uint16("kind", 0, "list kind")
	setList.Flags().String("tags", "", "tags JSON array")
	setList.Flags().String("content", "", "list content")

	list := &cobra.Command{
		Use:   "list",
		Short: "Get a Nostr list",
		RunE: func(cmd *cobra.Command, args []string) error {
			pubkey, err := requiredFlag(cmd, "pubkey")
			if err != nil {
				return err
			}
			pubkey, err = validateHex64(pubkey)
			if err != nil {
				return err
			}
			kind, _ := cmd.Flags().GetUint32("kind")
			if err := validateSocialKind(kind); err != nil {
				return err
			}
			dTag, _ := cmd.Flags().GetString("d-tag")
			if dTag != "" && !isParameterizedSocialKind(kind) {
				return inputError(fmt.Sprintf("kind %d is not parameterized; omit --d-tag", kind))
			}
			filter := client.Filter{"kinds": []int{int(kind)}, "authors": []string{pubkey}, "limit": 10}
			if dTag != "" {
				filter["#d"] = []string{dTag}
			}
			return opts.query(cmd.Context(), []client.Filter{filter})
		},
	}
	list.Flags().String("pubkey", "", "pubkey")
	list.Flags().Uint32("kind", 0, "list kind")
	list.Flags().String("d-tag", "", "parameterized d tag")

	cmd.AddCommand(publish, setContacts, eventCmd, notes, contacts, setList, list)
	return cmd
}

func validateSocialKind(kind uint32) error {
	switch kind {
	case 10000, 10001, 10002, 10003, 30000, 30003:
		return nil
	default:
		return inputError("kind must be one of 10000, 10001, 10002, 10003, 30000, 30003")
	}
}

func isParameterizedSocialKind(kind uint32) bool {
	return kind == 30000 || kind == 30003
}

func tagsContainD(tags nostr.Tags) bool {
	for _, tag := range tags {
		if len(tag) > 0 && tag[0] == "d" {
			return true
		}
	}
	return false
}
