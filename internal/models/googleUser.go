package models


// GoogleUser represents the Google user data.
type GoogleUser struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	VerifiedEmail bool   `json:"verified_email"`
	Name         string `json:"name"`
	GivenName    string `json:"given_name"`
	FamilyName   string `json:"family_name"`
}