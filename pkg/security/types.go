package security

type CSRFConfig struct {
	CheckOrigin  bool
	AllowOrigins []string
	SiteURL      string
}
