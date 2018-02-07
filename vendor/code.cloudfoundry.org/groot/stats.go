package groot

func (g *Groot) Stats(handle string) (VolumeStats, error) {
	g.Logger = g.Logger.Session("stats")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	return g.Driver.Stats(g.Logger, handle)
}
