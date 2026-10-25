// Package ruledata defines the extracted, native table-rule assets.
// IDs are opaque record identities retained for development evidence. Runtime
// code follows named relationships and never performs pointer arithmetic on IDs.
package ruledata

import (
	"github.com/olivierh59500/pinballdreams/internal/audiodata"
	"image"
)

type Lamp struct {
	ID                           int
	Flags, InitialBlink          byte
	InitialPlayers, InitialPhase byte
	X, Y                         int
	Pixels                       []image.Point
	RetriggerPhase               [16]byte
}

type Group struct {
	ID, Program, Period int
	InitialTimer        int
	InitialDirection    bool
	Lamps               []int
}

type StageKind string

const (
	Collectible StageKind = "collectible"
	Repeatable  StageKind = "repeatable"
)

type Stage struct {
	ID, Priority, OffPriority              int
	Kind                                   StageKind
	Permanent                              bool
	RetainOnBall                           bool
	ConsumeInPath, ConsumeInGroup          bool
	SaveAwardOnArm                         int
	InitialPlayers                         byte
	OnLamp, OffLamp, OnSound, OffSound     int
	OnMessage, OffMessage, Flags, Duration int
	Points, Bonus                          uint64
	Effect                                 Effect
	// Hook is provenance for oracle diagnostics; it is never a callable entry.
	Hook int
}

type EffectKind string

const (
	NoEffect             EffectKind = "none"
	SwitchStages         EffectKind = "switch-stages"
	AnnounceAdvance      EffectKind = "announce-advance"
	CopyLeadingScore     EffectKind = "copy-leading-score"
	ExtraBall            EffectKind = "extra-ball"
	SetMultiplier        EffectKind = "set-multiplier"
	HoldBonus            EffectKind = "hold-bonus"
	ProgressFeature      EffectKind = "progress-feature"
	RaiseJackpot         EffectKind = "raise-jackpot"
	CollectJackpot       EffectKind = "collect-jackpot"
	FixedAward           EffectKind = "fixed-award"
	SavedAward           EffectKind = "saved-award"
	CopyScore            EffectKind = "copy-score"
	CopyBonus            EffectKind = "copy-bonus"
	ArmLocks             EffectKind = "arm-locks"
	ReserveBall          EffectKind = "reserve-ball"
	CollectLockedJackpot EffectKind = "collect-locked-jackpot"
	ChangeGate           EffectKind = "change-gate"
	CountBonus           EffectKind = "count-bonus"
	SelectRoulette       EffectKind = "select-roulette"
	Untranslated         EffectKind = "untranslated"
)

// Effect is an explicit native game action, with already-decoded parameters.
type Effect struct {
	Kind                  EffectKind
	Value, Clear, Arm     int
	Program, Group, Timer int
	Bank, Gate, Text      int
	ClampProgress         bool
	Amount                uint64
	Messages              map[int]int
	CopyText              *TextCopy
}

// TextCopy snapshots a published award label before its running value resets.
type TextCopy struct {
	Source, Destination, Characters int
}

type Timer struct{ ID, Period, Group, Program, Sound int }

type Target struct {
	ID, Kind, Next, X0, Y0, X1, Y1, Lamp int
	InitialPlayers                       byte
	InitialCooldown                      int
	Pixels                               []image.Point
}

type TargetGroup struct {
	Layers             byte
	ID, X0, Y0, X1, Y1 int
	Targets            []int
}

// TargetCycle describes the supplied target retrigger and bank reset cadence.
type TargetCycle struct {
	ResetDelay, ResetPeriod, RetriggerFrames int
	ResetSound                               int
}

type GateAction struct {
	Target, Layer int
	Players       byte
	Conditions    []int
}

type Jackpot struct {
	ID, RaiseHook                         int
	Initial, Increment                    uint64
	AwardSourceInitial, SavedAwardInitial uint64
}

type LockSlot struct{ Object, Lamp int }

type Locks struct {
	ClearEmptyIndicatorsOnRelease       bool
	Slots                               []LockSlot
	CaptureStage, SuperBank, SuperDigit int
	// These identities are extraction/verification provenance only.
	EnableHook, CaptureHook, SuperHook int
}

type Bonus struct {
	Text                                      string
	EntryText                                 string
	EntryFrames                               int
	EntrySounds                               [2]int
	InitialDelay, Period, MultiplierDelay     int
	CountSound, MultiplierSound               int
	BoosterText                               string
	BoosterPosition, BoosterSound             int
	AfterCountDelay, BoosterPeriod, ExitDelay int
	FinalHoldSound, FinalHoldMessage          int
}

type RouletteEntry struct{ Program, Lamp, Text int }

type Roulette struct {
	DefaultText, Limit, Sound, InitialSpeed int
	Lookup                                  [128]byte
	Entries                                 []RouletteEntry
	Hook, Root                              int
}

type Sequences struct {
	Roulettes map[int]Roulette
	Counters  map[int]int
}

type PathMode string

const (
	Sequential PathMode = "sequential"
	Selected   PathMode = "selected"
	Together   PathMode = "together"
)

type StageEntry struct {
	Stage, Path int
	Arm         bool
}

// ProgressStep is a resolved feature transition. Reset/back/jump markers are
// resolved during extraction, rather than interpreted by the game loop.
type ProgressStep struct {
	StageEntry
	Index, Next, Unavailable int
	Stop                     bool
	Selectable               bool
}

type FeaturePath struct {
	Mode        PathMode
	ResetOnBall bool
	Steps       map[int]ProgressStep
	Entries     []StageEntry
}

type Event struct {
	Points, Bonus uint64
	Sound         int
}

type ObjectRole string

const (
	ContactObject     ObjectRole = "contact"
	AcceleratorObject ObjectRole = "accelerator"
	TrapObject        ObjectRole = "trap"
	GateEnterObject   ObjectRole = "gate-enter"
	GateExitObject    ObjectRole = "gate-exit"
)

type Capture struct {
	Period, Count    int
	CenterX, CenterY int
	VX, VY           int16
}

type Object struct {
	Role        ObjectRole
	Lamp, Event int
	Gate        int
	GatePlayers byte
	Capture     Capture
}

type Sensor struct {
	X0, Y0, X1, Y1 int
	Object         int
}

type Layer struct{ Sensors, ExtraSensors []Sensor }

type Sound = audiodata.Command

type Announcement struct {
	ID, Text               int
	Flags, Period, Repeats byte
	Speed                  int16
	Masks                  [][20]byte
	MasksEnabled           bool
	SequenceSource         int
}

type TextSlot struct{ Text, Offset int }

// Definitions contains immutable game data produced by the Go extractor.
type Definitions struct {
	Version, TableID                              int
	SourceSHA256                                  string
	Panel                                         PanelUI
	Relaunch                                      RelaunchPolicy
	Gates                                         []int
	GateActions                                   map[int]GateAction
	BoosterStage                                  int
	BoosterValue                                  uint64
	AdvanceStage, AdvanceLimit, AdvanceOverflow   int
	Groups                                        []Group
	LampDrawBudget                                int
	Lamps                                         map[int]Lamp
	Stages                                        map[int]Stage
	Timers                                        []Timer
	Sequential, Rotating, SkillLamps, TimedStages []int
	SequentialMarkCollected                       bool
	SkillGroup, SkillPath, ExtraBallLamp          int
	BallStartPath                                 int
	Jackpots                                      map[int]Jackpot
	Bonus                                         Bonus
	Locks                                         Locks
	Sequences                                     Sequences
	Targets                                       map[int]Target
	TargetGroups                                  []TargetGroup
	TargetCycle                                   TargetCycle
	Paths                                         map[int]FeaturePath
	Features                                      map[int]int
	Events                                        map[int]Event
	Objects                                       map[int]Object
	Layers                                        []Layer
	FreezeSensors                                 bool
	FreezeTimedStages                             bool
	ResetAtPanEnd                                 bool
	KeepLatchAtYStop                              bool
	CaptureReleaseSound                           int
	Sounds                                        map[int]Sound
	Announcements                                 map[int]Announcement
	Texts                                         map[int][]byte
	TextSlots                                     map[int][]TextSlot
}
