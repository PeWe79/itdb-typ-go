package assets

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"itdb-backend/internal/common/primitives"
	"itdb-backend/pkg/database"
	"itdb-backend/router/common"
	"mime"
	"mime/multipart"
	"net/http"

	"itdb-backend/internal/service"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

func (a *Router) handleListLocations(w http.ResponseWriter, r *http.Request) {
	rows, e := a.locationWorkflow.List(r.Context(), r.URL.Query().Get("search"))
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	defer rows.Close()
	out, e := database.RowsToMaps(rows)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, out)
}
func (a *Router) handleGetLocation(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	lr, e := a.locationWorkflow.Get(r.Context(), id)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	defer lr.Close()
	rows, e := database.RowsToMaps(lr)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	if len(rows) == 0 {
		common.WriteError(w, http.StatusNotFound, "location not found")
		return
	}
	p := rows[0]
	ar, e := a.locationWorkflow.Areas(r.Context(), id)
	if e == nil {
		defer ar.Close()
		p["areas"], _ = database.RowsToMaps(ar)
	}
	common.WriteJSON(w, http.StatusOK, p)
}

func (a *Router) handleNextLocAreaID(w http.ResponseWriter, r *http.Request) {
	nextID, e := a.locationWorkflow.NextAreaID(r.Context())
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"nextId": nextID})
}

func (a *Router) handleDownloadLocationFloorplan(w http.ResponseWriter, r *http.Request) {
	id, err := primitives.IntParam(chi.URLParam(r, "id"))
	if err != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	floorplan, locationName, err := a.locationWorkflow.Floorplan(r.Context(), id)
	if err != nil {
		if err == sql.ErrNoRows {
			common.WriteError(w, http.StatusNotFound, "floorplan not found")
		} else {
			common.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	fileName := strings.TrimSpace(floorplan)
	if fileName == "" {
		common.WriteError(w, http.StatusNotFound, "floorplan not found")
		return
	}
	if fileName != filepath.Base(fileName) {
		common.WriteError(w, http.StatusBadRequest, "invalid floorplan name")
		return
	}
	path := filepath.Join(a.cfg.UploadDir, fileName)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		common.WriteError(w, http.StatusNotFound, "floorplan not found")
		return
	}

	ext := strings.ToLower(filepath.Ext(fileName))
	displayName := fileName
	if name := strings.TrimSpace(locationName); name != "" {
		displayName = fmt.Sprintf("%s%s", name, ext)
	}
	if contentType := mime.TypeByExtension(ext); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	common.SetContentDispositionHeader(w, "inline", displayName)
	http.ServeFile(w, r, path)
}

func (a *Router) handleCreateLocation(w http.ResponseWriter, r *http.Request) {
	u, _ := common.CurrentUser(r.Context())
	req := locationPayload{}
	floorplan := ""
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/") {
		if e := r.ParseMultipartForm(64 << 20); e != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}
		req.Name = strings.TrimSpace(r.FormValue("name"))
		req.Floor = strings.TrimSpace(r.FormValue("floor"))
		req.ChangeNote = strings.TrimSpace(r.FormValue("changeNote"))
		if raw := strings.TrimSpace(r.FormValue("areas")); raw != "" {
			_ = json.Unmarshal([]byte(raw), &req.Areas)
		}
		h, _ := common.GetMultipartFileHeader(r.MultipartForm, "file")
		if h != nil {
			var e error
			floorplan, e = a.storeFloorplanFile(h, req.Name)
			if e != nil {
				common.WriteError(w, http.StatusBadRequest, e.Error())
				return
			}
		}
	} else {
		if e := json.NewDecoder(r.Body).Decode(&req); e != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid json")
			return
		}
	}
	id, e := a.locationWorkflow.CreateWithFloorplan(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, req, floorplan)
	if e != nil {
		writeLocationError(w, e)
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
func (a *Router) handleUpdateLocation(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	req := locationPayload{}
	var floorplan *string
	if strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "multipart/") {
		if e = r.ParseMultipartForm(64 << 20); e != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid multipart form")
			return
		}
		req.Name = strings.TrimSpace(r.FormValue("name"))
		req.Floor = strings.TrimSpace(r.FormValue("floor"))
		req.ChangeNote = strings.TrimSpace(r.FormValue("changeNote"))
		if raw := strings.TrimSpace(r.FormValue("areas")); raw != "" {
			_ = json.Unmarshal([]byte(raw), &req.Areas)
		}
		h, _ := common.GetMultipartFileHeader(r.MultipartForm, "file")
		if h != nil {
			saved, e := a.storeFloorplanFile(h, req.Name)
			if e != nil {
				common.WriteError(w, http.StatusBadRequest, e.Error())
				return
			}
			floorplan = &saved
		}
	} else {
		if e = json.NewDecoder(r.Body).Decode(&req); e != nil {
			common.WriteError(w, http.StatusBadRequest, "invalid json")
			return
		}
	}
	old, e := a.locationWorkflow.UpdateWithFloorplan(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id, req, floorplan)
	if e != nil {
		writeLocationError(w, e)
		return
	}
	if floorplan != nil && strings.TrimSpace(old) != "" && filepath.Base(old) == old {
		_ = os.Remove(filepath.Join(a.cfg.UploadDir, old))
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": id})
}
func (a *Router) handleDeleteLocation(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	old, e := a.locationWorkflow.Delete(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, id)
	if e != nil {
		if e == sql.ErrNoRows {
			common.WriteError(w, http.StatusNotFound, "location not found")
		} else if conflict := newServiceConflict(e); conflict != "" {
			common.WriteError(w, http.StatusConflict, conflict)
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	if strings.TrimSpace(old) != "" && filepath.Base(old) == old {
		_ = os.Remove(filepath.Join(a.cfg.UploadDir, old))
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}
func writeLocationError(w http.ResponseWriter, e error) {
	if e.Error() == "name and floor are required" || e.Error() == "invalid multipart form" || strings.Contains(e.Error(), "floorplan file") {
		common.WriteError(w, http.StatusBadRequest, e.Error())
		return
	}
	common.WriteError(w, http.StatusInternalServerError, e.Error())
}

func (a *Router) handleListLocAreas(w http.ResponseWriter, r *http.Request) {
	id, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid location id")
		return
	}
	rows, e := a.locationWorkflow.Areas(r.Context(), id)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	defer rows.Close()
	out, e := database.RowsToMaps(rows)
	if e != nil {
		common.WriteError(w, http.StatusInternalServerError, e.Error())
		return
	}
	common.WriteJSON(w, http.StatusOK, out)
}

func (a *Router) handleCreateLocArea(w http.ResponseWriter, r *http.Request) {
	locationID, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid location id")
		return
	}
	var req locAreaPayload
	if e = json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	id, e := a.locationWorkflow.CreateArea(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, locationID, req)
	if e != nil {
		if e.Error() == "areaName is required" {
			common.WriteError(w, http.StatusBadRequest, e.Error())
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusCreated, map[string]int64{"id": id})
}
func (a *Router) handleUpdateLocArea(w http.ResponseWriter, r *http.Request) {
	locationID, e := primitives.IntParam(chi.URLParam(r, "id"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid location id")
		return
	}
	areaID, e := primitives.IntParam(chi.URLParam(r, "areaId"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid area id")
		return
	}
	var req locAreaPayload
	if e = json.NewDecoder(r.Body).Decode(&req); e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.locationWorkflow.UpdateArea(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, areaID, locationID, req); e != nil {
		if e.Error() == "areaName is required" {
			common.WriteError(w, http.StatusBadRequest, e.Error())
		} else if e == sql.ErrNoRows {
			common.WriteError(w, http.StatusNotFound, "area not found")
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]int64{"id": areaID})
}
func (a *Router) handleDeleteLocArea(w http.ResponseWriter, r *http.Request) {
	areaID, e := primitives.IntParam(chi.URLParam(r, "areaId"))
	if e != nil {
		common.WriteError(w, http.StatusBadRequest, "invalid area id")
		return
	}
	u, _ := common.CurrentUser(r.Context())
	if e = a.locationWorkflow.DeleteArea(r.Context(), service.UserActor{Username: u.Username, IP: common.ClientIP(r)}, areaID); e != nil {
		if conflict := newServiceConflict(e); conflict != "" {
			common.WriteError(w, http.StatusConflict, conflict)
		} else if e == sql.ErrNoRows {
			common.WriteError(w, http.StatusNotFound, "area not found")
		} else {
			common.WriteError(w, http.StatusInternalServerError, e.Error())
		}
		return
	}
	common.WriteJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (a *Router) storeFloorplanFile(h *multipart.FileHeader, locationName string) (string, error) {
	if a.files == nil {
		return "", errors.New("file storage service is unavailable")
	}
	return a.files.StoreFloorplanFile(h, locationName)
}
