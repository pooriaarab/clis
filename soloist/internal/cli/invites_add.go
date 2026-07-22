// Hand-authored (not generated). `invites add` — semantic wrapper over the raw
// WebsiteInvites create that builds a correct WebsiteInvite document (creating
// the doc is the invite).

package cli

import (
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
)

const websiteInvitesCreatePath = "/v1/projects/moz-ocho-solo-prod/databases/(default)/documents/WebsiteInvites"

var websiteUserRoles = map[string]bool{"Viewer": true, "Collaborator": true, "Admin": true, "Owner": true}

func newInvitesAddCmd(flags *rootFlags) *cobra.Command {
	var role string
	cmd := &cobra.Command{
		Use:     "add <websiteId> <email>",
		Short:   "Invite a collaborator to a website (creates a WebsiteInvite).",
		Example: "  soloist-pp-cli invites add <websiteId> jane@example.com --role Collaborator",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 && cmd.Flags().NFlag() == 0 {
				return cmd.Help()
			}
			if len(args) < 2 {
				return missingSitesArgument(cmd, flags, "<websiteId> <email>")
			}
			if !websiteUserRoles[role] {
				_ = cmd.Usage()
				return usageErr(fmt.Errorf("--role must be one of Viewer, Collaborator, Admin, Owner"))
			}
			websiteID, email := args[0], args[1]
			id := uuid.NewString()
			fields := map[string]any{
				"id":               fsEncodeString(id),
				"websiteId":        fsEncodeString(websiteID),
				"invitedUserEmail": fsEncodeString(email),
				"role":             fsEncodeString(role),
				"accepted":         fsEncodeBool(false),
				"createdAt":        map[string]any{"timestampValue": time.Now().UTC().Format(time.RFC3339)},
			}
			body := map[string]any{"fields": fields}
			if dryRunOK(flags) {
				return writeSitesMutationResult(cmd, flags, map[string]any{
					"action": "invites-add", "dry_run": true, "inviteId": id,
					"websiteId": websiteID, "email": email, "role": role, "fields": fields,
				}, fmt.Sprintf("would invite %s (%s) to %s", email, role, websiteID))
			}
			c, err := flags.newClient()
			if err != nil {
				return err
			}
			path := websiteInvitesCreatePath + "?documentId=" + url.QueryEscape(id)
			resp, status, err := c.Post(cmd.Context(), path, body)
			if err != nil {
				return classifyAPIError(err, flags)
			}
			if status < 200 || status >= 300 {
				return fmt.Errorf("invite failed: HTTP %d: %s", status, string(resp))
			}
			return writeSitesMutationResult(cmd, flags, map[string]any{
				"action": "invites-add", "status": status, "inviteId": id,
				"email": email, "role": role, "response": parseJSONForOutput(resp),
			}, fmt.Sprintf("invited %s (%s)", email, role))
		},
	}
	cmd.Flags().StringVar(&role, "role", "Collaborator", "Viewer | Collaborator | Admin | Owner")
	return cmd
}
