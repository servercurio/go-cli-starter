package application

// initializeRouting attaches every registered module to its corresponding
// Echo group. When TLS is enabled, modules are mounted on the TLS server
// (the HTTP server in that case is reserved for the HTTPS-redirect
// middleware) so handlers don't run twice for the same logical request.
func (app *Application) initializeRouting() error {
	server := app.httpServer

	if app.tlsEnabled() {
		server = app.tlsServer
	}

	for _, m := range app.modules {
		g := server.Group(m.Prefix(), m.Middleware()...)
		m.AttachGroup(g)
	}

	return nil
}
