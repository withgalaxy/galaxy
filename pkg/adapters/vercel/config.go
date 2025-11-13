package vercel

type VercelConfig struct {
	Version int     `json:"version"`
	Routes  []Route `json:"routes,omitempty"`
}

type Route struct {
	Src     string            `json:"src,omitempty"`
	Dest    string            `json:"dest,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
	Methods []string          `json:"methods,omitempty"`
	Status  int               `json:"status,omitempty"`
	Handle  string            `json:"handle,omitempty"`
}

func NewVercelConfig() *VercelConfig {
	return &VercelConfig{
		Version: 3,
		Routes:  []Route{},
	}
}

func (c *VercelConfig) AddRoute(route Route) {
	c.Routes = append(c.Routes, route)
}
