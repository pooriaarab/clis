package cli

import (
	"encoding/base64"
	"encoding/json"
	"strings"

	"soloist-pp-cli/internal/client"
)

// BuildFirestoreStringEqualQueryBody builds a Firestore runQuery body with one
// EQUAL string fieldFilter.
func BuildFirestoreStringEqualQueryBody(collectionID, fieldPath, value string) map[string]any {
	return firestoreQueryBody(collectionID, map[string]any{
		"fieldFilter": stringEqualFieldFilter(fieldPath, value),
	})
}

// BuildWebsiteInvitesQueryBody builds the WebsiteInvites runQuery body. When
// websiteIDMode is true it filters by websiteId; otherwise it filters by
// invitedUserEmail and, unless includePending is true, accepted == true.
func BuildWebsiteInvitesQueryBody(websiteID string, websiteIDMode bool, email string, includePending bool) map[string]any {
	if websiteIDMode {
		return BuildFirestoreStringEqualQueryBody("WebsiteInvites", "websiteId", websiteID)
	}

	emailFilter := map[string]any{
		"fieldFilter": stringEqualFieldFilter("invitedUserEmail", email),
	}
	if includePending {
		return firestoreQueryBody("WebsiteInvites", emailFilter)
	}

	return firestoreQueryBody("WebsiteInvites", map[string]any{
		"compositeFilter": map[string]any{
			"op": "AND",
			"filters": []map[string]any{
				emailFilter,
				{
					"fieldFilter": booleanEqualFieldFilter("accepted", true),
				},
			},
		},
	})
}

// ReadSoloistIDTokenClaims decodes uid/email from the loaded bearer token on
// the client. It intentionally does not verify the JWT signature; this is only
// deriving query defaults from credentials already loaded for the request.
func ReadSoloistIDTokenClaims(c *client.Client) (uid, email string) {
	token := soloistBearerToken(c)
	if token == "" {
		return "", ""
	}

	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return "", ""
	}

	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		payload, err = base64.URLEncoding.DecodeString(parts[1])
		if err != nil {
			return "", ""
		}
	}

	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", ""
	}

	uid = stringClaim(claims, "user_id")
	if uid == "" {
		uid = stringClaim(claims, "sub")
	}
	email = stringClaim(claims, "email")
	return uid, email
}

func firestoreQueryBody(collectionID string, where map[string]any) map[string]any {
	return map[string]any{
		"structuredQuery": map[string]any{
			"from": []map[string]any{
				{"collectionId": collectionID},
			},
			"where": where,
		},
	}
}

func stringEqualFieldFilter(fieldPath, value string) map[string]any {
	return map[string]any{
		"field": map[string]any{
			"fieldPath": fieldPath,
		},
		"op": "EQUAL",
		"value": map[string]any{
			"stringValue": value,
		},
	}
}

func booleanEqualFieldFilter(fieldPath string, value bool) map[string]any {
	return map[string]any{
		"field": map[string]any{
			"fieldPath": fieldPath,
		},
		"op": "EQUAL",
		"value": map[string]any{
			"booleanValue": value,
		},
	}
}

func soloistBearerToken(c *client.Client) string {
	if c == nil || c.Config == nil {
		return ""
	}

	authHeader := strings.TrimSpace(c.Config.AuthHeader())
	if scheme, token, ok := strings.Cut(authHeader, " "); ok && strings.EqualFold(scheme, "Bearer") {
		return strings.TrimSpace(token)
	}
	if c.Config.SoloistIdToken != "" {
		return strings.TrimSpace(c.Config.SoloistIdToken)
	}
	if c.Config.AccessToken != "" {
		return strings.TrimSpace(c.Config.AccessToken)
	}
	return ""
}

func stringClaim(claims map[string]any, key string) string {
	value, ok := claims[key].(string)
	if !ok {
		return ""
	}
	return value
}
