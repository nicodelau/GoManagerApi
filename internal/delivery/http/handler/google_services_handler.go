package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"time"

	"gomanager/internal/domain/user"
	"gomanager/internal/infrastructure/config"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

// GoogleServicesHandler handles Google Calendar and Tasks API calls
type GoogleServicesHandler struct {
	oauthConfig *oauth2.Config
	userRepo    user.Repository
}

// NewGoogleServicesHandler creates a new Google services handler
func NewGoogleServicesHandler(cfg *config.Config, userRepo user.Repository) *GoogleServicesHandler {
	oauthConfig := &oauth2.Config{
		ClientID:     cfg.GoogleClientID,
		ClientSecret: cfg.GoogleClientSecret,
		RedirectURL:  cfg.BaseURL + "/api/auth/google/callback",
		Scopes: []string{
			"https://www.googleapis.com/auth/userinfo.email",
			"https://www.googleapis.com/auth/userinfo.profile",
			"https://www.googleapis.com/auth/calendar.readonly",
			"https://www.googleapis.com/auth/calendar.events",
			"https://www.googleapis.com/auth/tasks.readonly",
			"https://www.googleapis.com/auth/tasks",
			"https://www.googleapis.com/auth/drive",
			"https://www.googleapis.com/auth/drive.file",
			"https://www.googleapis.com/auth/adwords",
		},
		Endpoint: google.Endpoint,
	}

	return &GoogleServicesHandler{
		oauthConfig: oauthConfig,
		userRepo:    userRepo,
	}
}

// CalendarEvent represents a Google Calendar event
type CalendarEvent struct {
	ID          string    `json:"id"`
	Summary     string    `json:"summary"`
	Description string    `json:"description,omitempty"`
	Location    string    `json:"location,omitempty"`
	Start       EventTime `json:"start"`
	End         EventTime `json:"end"`
	HtmlLink    string    `json:"htmlLink,omitempty"`
	Status      string    `json:"status,omitempty"`
}

// EventTime represents a time for an event
type EventTime struct {
	DateTime string `json:"dateTime,omitempty"`
	Date     string `json:"date,omitempty"`
	TimeZone string `json:"timeZone,omitempty"`
}

// Task represents a Google Task
type Task struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Notes     string `json:"notes,omitempty"`
	Status    string `json:"status"`
	Due       string `json:"due,omitempty"`
	Completed string `json:"completed,omitempty"`
	Links     []struct {
		Type string `json:"type"`
		Link string `json:"link"`
	} `json:"links,omitempty"`
}

// TaskList represents a Google Task List
type TaskList struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// getOAuthClient creates an OAuth2 client for the user
func (h *GoogleServicesHandler) getOAuthClient(u *user.User) (*http.Client, error) {
	if u.GoogleToken == "" {
		return nil, ErrNoGoogleToken
	}

	token := &oauth2.Token{
		RefreshToken: u.GoogleToken,
		TokenType:    "Bearer",
	}

	tokenSource := h.oauthConfig.TokenSource(context.Background(), token)
	return oauth2.NewClient(context.Background(), tokenSource), nil
}

// ListCalendars handles GET /api/google/calendars
func (h *GoogleServicesHandler) ListCalendars(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	resp, err := client.Get("https://www.googleapis.com/calendar/v3/users/me/calendarList")
	if err != nil {
		SendError(w, "Failed to fetch calendars", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Items []struct {
			ID              string `json:"id"`
			Summary         string `json:"summary"`
			Description     string `json:"description"`
			BackgroundColor string `json:"backgroundColor"`
			Primary         bool   `json:"primary"`
		} `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		SendError(w, "Failed to parse calendars", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "", result.Items)
}

// ListEvents handles GET /api/google/calendar/events
func (h *GoogleServicesHandler) ListEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	// Get query params
	calendarID := r.URL.Query().Get("calendarId")
	if calendarID == "" {
		calendarID = "primary"
	}

	// Default to next 30 days
	timeMin := time.Now().Format(time.RFC3339)
	timeMax := time.Now().AddDate(0, 0, 30).Format(time.RFC3339)

	if tm := r.URL.Query().Get("timeMin"); tm != "" {
		timeMin = tm
	}
	if tm := r.URL.Query().Get("timeMax"); tm != "" {
		timeMax = tm
	}

	maxResults := r.URL.Query().Get("maxResults")
	if maxResults == "" {
		maxResults = "50"
	}

	apiURL := "https://www.googleapis.com/calendar/v3/calendars/" + url.PathEscape(calendarID) + "/events"
	apiURL += "?timeMin=" + url.QueryEscape(timeMin)
	apiURL += "&timeMax=" + url.QueryEscape(timeMax)
	apiURL += "&maxResults=" + maxResults
	apiURL += "&singleEvents=true"
	apiURL += "&orderBy=startTime"

	resp, err := client.Get(apiURL)
	if err != nil {
		SendError(w, "Failed to fetch events", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Items []CalendarEvent `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		SendError(w, "Failed to parse events", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "", result.Items)
}

// CreateEvent handles POST /api/google/calendar/events
func (h *GoogleServicesHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	calendarID := r.URL.Query().Get("calendarId")
	if calendarID == "" {
		calendarID = "primary"
	}

	// Read the event from request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	apiURL := "https://www.googleapis.com/calendar/v3/calendars/" + url.PathEscape(calendarID) + "/events"

	req, _ := http.NewRequest("POST", apiURL, io.NopCloser(io.Reader(nil)))
	req.Header.Set("Content-Type", "application/json")

	// Create a new request with the body
	resp, err := client.Post(apiURL, "application/json", io.NopCloser(jsonReader(body)))
	if err != nil {
		SendError(w, "Failed to create event", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		SendError(w, "Failed to create event: "+string(respBody), resp.StatusCode)
		return
	}

	var event CalendarEvent
	json.Unmarshal(respBody, &event)

	SendSuccess(w, "Event created", event)
}

// ListTaskLists handles GET /api/google/tasks/lists
func (h *GoogleServicesHandler) ListTaskLists(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	resp, err := client.Get("https://www.googleapis.com/tasks/v1/users/@me/lists")
	if err != nil {
		SendError(w, "Failed to fetch task lists", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Items []TaskList `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		SendError(w, "Failed to parse task lists", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "", result.Items)
}

// ListTasks handles GET /api/google/tasks
func (h *GoogleServicesHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	taskListID := r.URL.Query().Get("taskListId")
	if taskListID == "" {
		taskListID = "@default"
	}

	showCompleted := r.URL.Query().Get("showCompleted")
	if showCompleted == "" {
		showCompleted = "false"
	}

	apiURL := "https://www.googleapis.com/tasks/v1/lists/" + url.PathEscape(taskListID) + "/tasks"
	apiURL += "?showCompleted=" + showCompleted
	apiURL += "&maxResults=100"

	resp, err := client.Get(apiURL)
	if err != nil {
		SendError(w, "Failed to fetch tasks", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Items []Task `json:"items"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		SendError(w, "Failed to parse tasks", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "", result.Items)
}

// CreateTask handles POST /api/google/tasks
func (h *GoogleServicesHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	taskListID := r.URL.Query().Get("taskListId")
	if taskListID == "" {
		taskListID = "@default"
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	apiURL := "https://www.googleapis.com/tasks/v1/lists/" + url.PathEscape(taskListID) + "/tasks"

	resp, err := client.Post(apiURL, "application/json", jsonReader(body))
	if err != nil {
		SendError(w, "Failed to create task", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		SendError(w, "Failed to create task", resp.StatusCode)
		return
	}

	var task Task
	json.Unmarshal(respBody, &task)

	SendSuccess(w, "Task created", task)
}

// UpdateTask handles PUT /api/google/tasks/{taskId}
func (h *GoogleServicesHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPatch {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	taskListID := r.URL.Query().Get("taskListId")
	if taskListID == "" {
		taskListID = "@default"
	}

	taskID := r.URL.Query().Get("taskId")
	if taskID == "" {
		SendError(w, "Task ID required", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	apiURL := "https://www.googleapis.com/tasks/v1/lists/" + url.PathEscape(taskListID) + "/tasks/" + url.PathEscape(taskID)

	req, _ := http.NewRequest("PUT", apiURL, jsonReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		SendError(w, "Failed to update task", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		SendError(w, "Failed to update task", resp.StatusCode)
		return
	}

	var task Task
	json.Unmarshal(respBody, &task)

	SendSuccess(w, "Task updated", task)
}

// CompleteTask handles POST /api/google/tasks/{taskId}/complete
func (h *GoogleServicesHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	taskListID := r.URL.Query().Get("taskListId")
	if taskListID == "" {
		taskListID = "@default"
	}

	taskID := r.URL.Query().Get("taskId")
	if taskID == "" {
		SendError(w, "Task ID required", http.StatusBadRequest)
		return
	}

	// Update task status to completed
	updateBody := `{"status": "completed"}`

	apiURL := "https://www.googleapis.com/tasks/v1/lists/" + url.PathEscape(taskListID) + "/tasks/" + url.PathEscape(taskID)

	req, _ := http.NewRequest("PATCH", apiURL, jsonReader([]byte(updateBody)))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		SendError(w, "Failed to complete task", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		SendError(w, "Failed to complete task", resp.StatusCode)
		return
	}

	SendSuccess(w, "Task completed", nil)
}

// GoogleConnectionStatus handles GET /api/google/status
func (h *GoogleServicesHandler) GoogleConnectionStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	connected := u.GoogleToken != ""

	SendSuccess(w, "", map[string]interface{}{
		"connected":    connected,
		"authProvider": u.AuthProvider,
		"hasCalendar":  connected,
		"hasTasks":     connected,
		"hasDrive":     connected,
		"hasAds":       connected,
	})
}

// ListDriveFiles handles GET /api/google/drive/files
func (h *GoogleServicesHandler) ListDriveFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	// Get query parameters
	folderID := r.URL.Query().Get("folderId")
	pageSize := r.URL.Query().Get("pageSize")
	if pageSize == "" {
		pageSize = "50"
	}
	pageToken := r.URL.Query().Get("pageToken")

	// Build API URL
	apiURL := "https://www.googleapis.com/drive/v3/files"
	apiURL += "?pageSize=" + pageSize
	if pageToken != "" {
		apiURL += "&pageToken=" + url.QueryEscape(pageToken)
	}

	// If folder ID specified, search within that folder
	if folderID != "" {
		apiURL += "&q=" + url.QueryEscape("'"+folderID+"' in parents")
	}

	apiURL += "&fields=nextPageToken,files(id,name,mimeType,size,parents,createdTime,modifiedTime,webViewLink)"

	resp, err := client.Get(apiURL)
	if err != nil {
		SendError(w, "Failed to fetch files", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Files         []DriveFile `json:"files"`
		NextPageToken string      `json:"nextPageToken"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		SendError(w, "Failed to parse files", http.StatusInternalServerError)
		return
	}

	SendSuccess(w, "", result)
}

// CreateDriveFolder handles POST /api/google/drive/folders
func (h *GoogleServicesHandler) CreateDriveFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	var request struct {
		Name     string `json:"name"`
		ParentID string `json:"parentId,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		SendError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Create folder metadata
	folderMetadata := map[string]interface{}{
		"name":     request.Name,
		"mimeType": "application/vnd.google-apps.folder",
	}

	if request.ParentID != "" {
		folderMetadata["parents"] = []string{request.ParentID}
	}

	body, _ := json.Marshal(folderMetadata)

	resp, err := client.Post("https://www.googleapis.com/drive/v3/files", "application/json", jsonReader(body))
	if err != nil {
		SendError(w, "Failed to create folder", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		SendError(w, "Failed to create folder", resp.StatusCode)
		return
	}

	var folder DriveFile
	json.Unmarshal(respBody, &folder)

	SendSuccess(w, "Folder created", folder)
}

// UploadDriveFile handles POST /api/google/drive/upload
func (h *GoogleServicesHandler) UploadDriveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	// Parse multipart form
	err = r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		SendError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		SendError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get folder ID from form
	folderID := r.FormValue("folderId")

	// Read file content
	content, err := io.ReadAll(file)
	if err != nil {
		SendError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Create file metadata
	fileMetadata := map[string]interface{}{
		"name": header.Filename,
	}

	if folderID != "" {
		fileMetadata["parents"] = []string{folderID}
	}

	metadataJSON, _ := json.Marshal(fileMetadata)

	// Use multipart upload for files
	boundary := "boundary123456789"
	var uploadBody bytes.Buffer

	// Write metadata part
	uploadBody.WriteString("--" + boundary + "\r\n")
	uploadBody.WriteString("Content-Type: application/json; charset=UTF-8\r\n\r\n")
	uploadBody.Write(metadataJSON)
	uploadBody.WriteString("\r\n")

	// Write file content part
	uploadBody.WriteString("--" + boundary + "\r\n")
	uploadBody.WriteString("Content-Type: " + header.Header.Get("Content-Type") + "\r\n\r\n")
	uploadBody.Write(content)
	uploadBody.WriteString("\r\n--" + boundary + "--")

	req, err := http.NewRequest("POST", "https://www.googleapis.com/upload/drive/v3/files?uploadType=multipart", &uploadBody)
	if err != nil {
		SendError(w, "Failed to create upload request", http.StatusInternalServerError)
		return
	}

	req.Header.Set("Content-Type", "multipart/related; boundary="+boundary)

	resp, err := client.Do(req)
	if err != nil {
		SendError(w, "Failed to upload file", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		SendError(w, "Upload failed: "+string(respBody), resp.StatusCode)
		return
	}

	var uploadedFile DriveFile
	json.Unmarshal(respBody, &uploadedFile)

	SendSuccess(w, "File uploaded successfully", uploadedFile)
}

// DeleteDriveFile handles DELETE /api/google/drive/files/{fileId}
func (h *GoogleServicesHandler) DeleteDriveFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		SendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	u := GetUserFromContext(r.Context())
	if u == nil {
		SendError(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	client, err := h.getOAuthClient(u)
	if err != nil {
		SendError(w, "Google account not connected", http.StatusBadRequest)
		return
	}

	fileID := r.URL.Query().Get("fileId")
	if fileID == "" {
		SendError(w, "File ID required", http.StatusBadRequest)
		return
	}

	req, err := http.NewRequest("DELETE", "https://www.googleapis.com/drive/v3/files/"+url.PathEscape(fileID), nil)
	if err != nil {
		SendError(w, "Failed to create delete request", http.StatusInternalServerError)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		SendError(w, "Failed to delete file", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		SendError(w, "Failed to delete file", resp.StatusCode)
		return
	}

	SendSuccess(w, "File deleted successfully", nil)
}

// DriveFile represents a Google Drive file
type DriveFile struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	MimeType     string   `json:"mimeType"`
	Size         string   `json:"size,omitempty"`
	Parents      []string `json:"parents,omitempty"`
	CreatedTime  string   `json:"createdTime"`
	ModifiedTime string   `json:"modifiedTime"`
	WebViewLink  string   `json:"webViewLink,omitempty"`
}

// Error for missing Google token
var ErrNoGoogleToken = &googleError{"Google account not connected"}

type googleError struct {
	message string
}

func (e *googleError) Error() string {
	return e.message
}

// Helper to create a reader from bytes
func jsonReader(data []byte) io.Reader {
	return io.NopCloser(readerFromBytes(data))
}

type bytesReader struct {
	data []byte
	pos  int
}

func readerFromBytes(data []byte) *bytesReader {
	return &bytesReader{data: data, pos: 0}
}

func (r *bytesReader) Read(p []byte) (n int, err error) {
	if r.pos >= len(r.data) {
		return 0, io.EOF
	}
	n = copy(p, r.data[r.pos:])
	r.pos += n
	return n, nil
}
