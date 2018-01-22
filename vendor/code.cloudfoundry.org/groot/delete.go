package groot

func (g *Groot) Delete(handle string) error {
	g.Logger = g.Logger.Session("delete")
	g.Logger.Debug("starting")
	defer g.Logger.Debug("ending")

	return g.Driver.Delete(g.Logger, handle)
}
