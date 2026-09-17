package service

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
)

type RelationService struct{ uploadDir string }

func NewRelationService(uploadDir string) *RelationService {
	return &RelationService{uploadDir: uploadDir}
}
func (s *RelationService) ReplaceIDLinks(tx *sql.Tx, deleteQuery, insertQuery string, ownerID int64, links []int64) error {
	if _, err := tx.Exec(deleteQuery, ownerID); err != nil {
		return err
	}
	for _, linkedID := range links {
		if linkedID == 0 {
			continue
		}
		if _, err := tx.Exec(insertQuery, ownerID, linkedID); err != nil {
			return err
		}
	}
	return nil
}
func (s *RelationService) ReplaceUndirectedItemLinks(tx *sql.Tx, ownerID int64, links []int64) error {
	if _, err := tx.Exec(`DELETE FROM itemlink WHERE itemid1 = ? OR itemid2 = ?`, ownerID, ownerID); err != nil {
		return err
	}
	seen := make(map[[2]int64]struct{})
	for _, linkedID := range links {
		if linkedID == 0 || linkedID == ownerID {
			continue
		}
		left, right := ownerID, linkedID
		if left > right {
			left, right = right, left
		}
		key := [2]int64{left, right}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		if _, err := tx.Exec(`INSERT INTO itemlink (itemid1, itemid2) VALUES (?, ?)`, left, right); err != nil {
			return err
		}
	}
	return nil
}
func (s *RelationService) CountFileLinks(tx *sql.Tx, fileID int64) (int64, error) {
	var count int64
	err := tx.QueryRow(`SELECT (SELECT COUNT(*) FROM software2file WHERE fileid = ?) + (SELECT COUNT(*) FROM invoice2file WHERE fileid = ?) + (SELECT COUNT(*) FROM item2file WHERE fileid = ?) + (SELECT COUNT(*) FROM contract2file WHERE fileid = ?)`, fileID, fileID, fileID, fileID).Scan(&count)
	return count, err
}
func (s *RelationService) DeleteFile(tx *sql.Tx, fileID int64) error {
	var fname sql.NullString
	_ = tx.QueryRow(`SELECT fname FROM files WHERE id = ?`, fileID).Scan(&fname)
	for _, query := range []string{`DELETE FROM files WHERE id = ?`, `DELETE FROM invoice2file WHERE fileid = ?`, `DELETE FROM software2file WHERE fileid = ?`, `DELETE FROM item2file WHERE fileid = ?`, `DELETE FROM contract2file WHERE fileid = ?`} {
		if _, err := tx.Exec(query, fileID); err != nil {
			return err
		}
	}
	if strings.TrimSpace(fname.String) != "" {
		_ = os.Remove(filepath.Join(s.uploadDir, fname.String))
	}
	return nil
}
func (s *RelationService) CleanupRemovedFileLinks(tx *sql.Tx, previous, current, targets []int64) error {
	if len(targets) == 0 {
		return nil
	}
	previousSet, currentSet := map[int64]struct{}{}, map[int64]struct{}{}
	for _, id := range previous {
		if id > 0 {
			previousSet[id] = struct{}{}
		}
	}
	for _, id := range current {
		if id > 0 {
			currentSet[id] = struct{}{}
		}
	}
	seen := map[int64]struct{}{}
	for _, id := range targets {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := previousSet[id]; !ok {
			continue
		}
		if _, ok := currentSet[id]; ok {
			continue
		}
		links, err := s.CountFileLinks(tx, id)
		if err != nil {
			return err
		}
		if links == 0 {
			if err := s.DeleteFile(tx, id); err != nil {
				return err
			}
		}
	}
	return nil
}
