package user

import "github.com/go-chi/chi/v5"

// RegisterRoutes registers user routes
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Route("/user", func(r chi.Router) {
		r.Get("/profile", h.GetProfile)
		r.Put("/profile", h.UpdateProfile)
		r.Get("/settings", h.GetSettings)
		r.Put("/settings", h.UpdateSettings)
	})
}
