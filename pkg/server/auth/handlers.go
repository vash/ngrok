package auth

import (
	"context"
	"fmt"
	"html/template"
	"mime"
	"net/http"
	"ngrok/pkg/log"
	"ngrok/pkg/server/assets"
	"ngrok/pkg/server/config"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var tmpl *template.Template
var serverAssetsPrefix = "assets/server"

// Server struct embedding config
type Handler struct {
	Config *config.Config
}

/* var funcMap = template.FuncMap{
	"equal": func(n int) bool { return n == 5 },
	"inc":   func(n int) int { return n + 1 },
} */

func init() {
	tmpl = template.New("root")
	// Get all embedded asset names
	assetNames := assets.AssetNames()

	for _, assetName := range assetNames {
		if filepath.Ext(assetName) != ".html" {
			continue
		}
		if !strings.HasPrefix(assetName, "assets/server/") {
			continue
		}

		content := assets.MustAsset(assetName)
		name := strings.TrimPrefix(assetName, "assets/server/") // Adjust if needed

		template.Must(tmpl.New(name).Parse(string(content)))
	}
}

func (h *Handler) HomePage(w http.ResponseWriter, r *http.Request) {
	year := time.Now().Year()

	data := map[string]any{
		"Title": "Go & HTMx Demo",
		"Year":  year,
	}

	err := tmpl.ExecuteTemplate(w, "views/index.html", data)
	if err != nil {
		log.Error("Failed to execute template: %v", err)
	}
}

func (h *Handler) GetAPIKeys(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// fmt.Println("Time Zone: ", r.Header.Get("X-TimeZone"))
	var intPage int
	intPage, _ = strconv.Atoi(r.URL.Query().Get("page"))
	if intPage == 0 {
		intPage = 1
	}

	offset := (intPage - 1) * 5

	apikeysSlice, err := ListAuthTokens(ctx, h.Config.Database, offset)
	if err != nil {
		log.Error("something went wrong: %s", err.Error())
	}
	err = tmpl.ExecuteTemplate(w, "key-list", apikeysSlice)
	if err != nil {
		log.Error("GetAPIKeys: Failed to execute template: %v, %+v", err, apikeysSlice)
	}
}

func (h *Handler) AddAPIKey(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	description := strings.Trim(r.PostFormValue("description"), " ")
	if len(description) == 0 {
		var errDescription string
		if len(description) == 0 {
			errDescription = "Please enter a description in this field"
		}

		data := map[string]string{
			"FormDescription": description,
			"ErrDescription":  errDescription,
		}

		w.Header().Set("HX-Retarget", "form")
		w.Header().Set("HX-Reswap", "innerHTML")
		err := tmpl.ExecuteTemplate(w, "new-key-form", data)
		if err != nil {
			log.Error("Failed to execute template: %v", err)
		}

		return
	}

	err := CreateAuthToken(ctx, h.Config.Database, description)
	if err != nil {
		var message string
		if strings.Contains(err.Error(), "CHECK constraint failed") {
			message = "The description is longer than 255 characters "
		} else {
			message = fmt.Sprintf("Error occurred: %s", err)
		}
		http.Error(w, "Bad Request", http.StatusBadRequest)

		w.Header().Set("HX-Retarget", "body")
		w.Header().Set("HX-Reswap", "beforeend")
		err := tmpl.ExecuteTemplate(w, "modal", message)
		if err != nil {
			log.Error("Failed to execute template: %v", err)
		}

		return
	}
	if err != nil {
		log.Error("Failed to execute template: %v", err)
	}
	w.Header().Set("HX-Location", "/")
}

func (h *Handler) RemoveAPIKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	id := r.URL.Query().Get("id")
	if id == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Error("No key specified for deletion")
	}
	log.Info("Deleting key ID %s", id)

	err := DeleteAuthToken(ctx, h.Config.Database, id)
	if err != nil {
		w.Header().Set("HX-Retarget", "body")
		w.Header().Set("HX-Reswap", "beforeend")
		err := tmpl.ExecuteTemplate(w, "modal", "Requested apikey was not found!")
		if err != nil {
			log.Error("Failed to execute template: %v", err)
		}

		return
	}

	w.Header().Set("HX-Location", "/")
}

func (h *Handler) ServeStaticFiles(w http.ResponseWriter, r *http.Request) {
	fileName := serverAssetsPrefix + r.URL.Path

	fileData, err := assets.Asset(fileName)
	if err != nil {
		log.Error("Failed to serve static file: %v", err)
		http.NotFound(w, r)
		return
	}

	// Determine the content type based on the file extension using the mime package
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set the content type header
	w.Header().Set("Content-Type", contentType)

	// Serve the embedded file
	w.Write(fileData)
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status": "OK", "message": "Web handler is working"}`)
}
