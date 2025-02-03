package utils

import (
	"fmt"
	"time"
)

func FormatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return fmt.Sprintf("%d seconds ago", int(diff.Seconds()))
	case diff < time.Hour:
		return fmt.Sprintf("%d minutes ago", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d h ago", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d d ago", int(diff.Hours()/24))
	case t.Year() == now.Year():
		return t.Format("Jan 2006") 
	default:
		return t.Format("Jan 02 2006") 
	}
}
