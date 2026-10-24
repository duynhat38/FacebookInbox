package facebook

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// uploadFile uploads imagePath to Facebook and returns the file/image ID.
// Mirrors Python FileUploader.upload() including MIME type detection and
// the file_id/fbid/image_id fallback order.
func uploadFile(ctx context.Context, client *http.Client, cookie string, tok messengerTokens, imagePath string) (string, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("messenger: open image: %w", err)
	}
	defer f.Close()

	// Detect MIME type — mirrors Python mimetypes.guess_type(filepath.name)
	contentType := mime.TypeByExtension(filepath.Ext(imagePath))
	if contentType == "" {
		contentType = "application/octet-stream"
		log.Printf("messenger: cannot guess MIME for %s, using %s", filepath.Base(imagePath), contentType)
	}

	// Multipart form with correct Content-Type per part
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="farr"; filename="%s"`, filepath.Base(imagePath))}
	h["Content-Type"] = []string{contentType}
	part, err := mw.CreatePart(h)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(part, f); err != nil {
		return "", err
	}
	if err := mw.Close(); err != nil {
		return "", err
	}

	// Query params — mirrors Python FileUploader._query_params()
	params := url.Values{
		"__aaid":      {"0"},
		"__user":      {tok.uid},
		"__a":         {"1"},
		"__req":       {tok.req()},
		"dpr":         {"1"},
		"__ccg":       {"EXCELLENT"},
		"__rev":       {tok.clientRevision},
		"__hsi":       {tok.hasteSession},
		"__comet_req": {"15"},
		"fb_dtsg":     {tok.dtsg},
		"jazoest":     {jazoest(tok.dtsg)},
		"lsd":         {tok.lsd},
		"__spin_r":    {tok.clientRevision},
		"__spin_b":    {"trunk"},
		"__spin_t":    {strconv.FormatInt(time.Now().Unix(), 10)},
	}
	if tok.dyn != "" {
		params.Set("__dyn", tok.dyn)
	}
	if tok.csr != "" {
		params.Set("__csr", tok.csr)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://www.facebook.com/ajax/mercury/upload.php?"+params.Encode(), &body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	tok.profile.setHeaders(req)
	tok.profile.setDeviceHints(req)
	req.Header.Set("Origin", "https://www.facebook.com")
	req.Header.Set("Referer", "https://www.facebook.com/messages/")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Priority", "u=1, i")
	// Fetch metadata — AJAX upload, not navigation
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("x-fb-lsd", tok.lsd)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("messenger: upload: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	log.Printf("messenger: upload HTTP %d (%d bytes): %.600s", resp.StatusCode, len(raw), raw)

	data, err := parseFBJSON(string(raw))
	if err != nil {
		return "", fmt.Errorf("messenger: upload response not JSON: %w", err)
	}

	// Detect account checkpoint — payload is {} when FB blocks the upload
	if cpURL := detectCheckpoint(data); cpURL != "" {
		return "", fmt.Errorf("messenger: tài khoản bị checkpoint — cần xác minh tại %s", cpURL)
	}

	payload, _ := data["payload"].(map[string]interface{})
	if payload == nil {
		// HTTP 200 but Facebook returned an error body (e.g. error 1357054).
		// Extract error code + summary for a meaningful error message.
		if errCode, ok := data["error"].(float64); ok && errCode != 0 {
			summary, _ := data["errorSummary"].(string)
			desc, _ := data["errorDescription"].(string)
			if desc == "" {
				desc = summary
			}
			return "", fmt.Errorf("messenger: upload rejected by Facebook (error %.0f): %s", errCode, desc)
		}
		return "", fmt.Errorf("messenger: upload response has no payload. raw: %.400s", raw)
	}

	// metadata can be {"0":{...}} or [{...}] depending on FB server version
	var meta0 map[string]interface{}
	switch m := payload["metadata"].(type) {
	case map[string]interface{}:
		meta0, _ = m["0"].(map[string]interface{})
	case []interface{}:
		if len(m) > 0 {
			meta0, _ = m[0].(map[string]interface{})
		}
	}
	if meta0 == nil {
		log.Printf("messenger: upload metadata missing. payload keys: %v, raw: %.600s", mapKeys(payload), raw)
		return "", errors.New("messenger: upload response missing metadata")
	}

	// File → file_id; Image → fbid or image_id (Python verified .jpg returns fbid)
	// JSON numbers unmarshal as float64; must format as integer — fmt.Sprintf("%v")
	// produces scientific notation (1.07e+15) which Facebook rejects as an invalid ID.
	for _, key := range []string{"file_id", "fbid", "image_id"} {
		if v, ok := meta0[key]; ok && v != nil {
			var id string
			if f, ok := v.(float64); ok {
				id = strconv.FormatInt(int64(f), 10)
			} else {
				id = fmt.Sprintf("%v", v)
			}
			log.Printf("messenger: upload success, %s=%s", key, id)
			return id, nil
		}
	}
	log.Printf("messenger: upload metadata missing id. meta keys: %v, raw: %.300s", mapKeys(meta0), raw)
	return "", errors.New("messenger: upload response has no file_id/fbid/image_id")
}
