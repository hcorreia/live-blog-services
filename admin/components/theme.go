package components

import "context"

type Theme struct {
	SiteName string
	Lang     string
}

func GetTheme(ctx context.Context) Theme {
	if theme, ok := ctx.Value("theme").(Theme); ok {
		return theme
	}
	return Theme{}
}
