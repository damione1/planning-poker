package templates

// PublicSiteURL is the canonical origin, independent of the request Host header.
const PublicSiteURL = "https://pokerplanning.net/"

const PublicSocialImage = PublicSiteURL + "static/img/planning-poker-social.jpg"

const publicTitle = "Online Planning Poker - No Sign-up | PokerPlanning"
const publicDescription = "Run online planning poker without signing up. Share a room, vote with Fibonacci or custom cards, and reveal story point estimates with your team."

type pageMetadata struct {
	Title       string
	Description string
	Canonical   string
	NoIndex     bool
	Room        bool
}

func homeMetadata(errorParam string) pageMetadata {
	return pageMetadata{
		Title:       publicTitle,
		Description: publicDescription,
		Canonical:   PublicSiteURL,
		NoIndex:     errorParam != "",
	}
}

// The application identity describes visible functionality. No reviews, ratings,
// usage figures or rich-result eligibility are inferred from this markup.
func applicationSchema() map[string]any {
	return map[string]any{
		"@context":            "https://schema.org",
		"@type":               "WebApplication",
		"@id":                 PublicSiteURL + "#application",
		"name":                "PokerPlanning",
		"url":                 PublicSiteURL,
		"description":         publicDescription,
		"applicationCategory": "BusinessApplication",
		"operatingSystem":     "Any",
		"browserRequirements": "Requires JavaScript and a modern web browser",
		"inLanguage":          "en",
	}
}
