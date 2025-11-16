package meals

import "github.com/go-chi/chi/v5"

// RegisterMealsRoutes registers all meal routes with the router
func (h *Handler) RegisterMealsRoutes(r chi.Router) {
	r.Route("/meals", func(r chi.Router) {
		r.Post("/parse", h.ParseMeal)
		r.Post("/confirm", h.ConfirmMeal)
		r.Get("/", h.ListMeals)
		r.Get("/suggestions", h.GetMealSuggestions)
		r.Get("/{id}", h.GetMeal)
		r.Put("/{id}", h.UpdateMeal)
		r.Delete("/{id}", h.DeleteMeal)
		r.Post("/{id}/copy", h.CopyMeal)
	})
}
