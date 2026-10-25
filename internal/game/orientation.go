package game

// OrientationChanged retires the old viewport's gestures before input resumes.
// Table entry and attract own active spring waits even before Playing is true.
func (g *Game) OrientationChanged() {
	if g.Session.Table != nil {
		g.Suspend()
	}
}
