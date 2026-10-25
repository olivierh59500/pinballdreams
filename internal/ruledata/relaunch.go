package ruledata

type RelaunchMode string

const (
	RelaunchSingle  RelaunchMode = "single"
	RelaunchCounted RelaunchMode = "counted"
)

// RelaunchPolicy defines an awarded extra-ball credit and its play-entry use.
// Counted credits retain the source byte wrap and signed decrement clamp.
type RelaunchPolicy struct {
	Mode      RelaunchMode
	Indicator int
}
