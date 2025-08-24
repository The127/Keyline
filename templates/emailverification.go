package templates

import _ "embed"

//go:embed default_email_verification_template.txt
var DefaultEmailVerificationTemplate []byte

type EmailVerificationTemplateData struct {
	VerificationLink string
}
