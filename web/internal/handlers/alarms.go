package handlers

import (
	"net/http"
)

// HandleAlarms renders the alarms page.
// GET /alarms
func HandleAlarms(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Histórico de Alarmes",
	}
	renderTemplate(w, "alarms.html", data, r)
}
