package rules

// These flags describe game animation and collection modes in the extracted
// assets. Naming them keeps the frame loop expressed in table concepts.
const (
	lampFlash              byte = 2
	lampAnimateDuringPlay  byte = 4
	lampExcludedFromBank   byte = 8
	lampBlinkWhenCollected byte = 16
	lampChase              byte = 32
	lampBlink              byte = 64
	lampPersistent         byte = 128
)

const (
	messageScroll byte = 1
	messageBlink  byte = 4
	messageShort  byte = 8
	messageReveal byte = 16
	messageMasks  byte = 32
)
