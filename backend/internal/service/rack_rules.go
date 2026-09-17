package service

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

func ValidateRackPlacement(ctx context.Context, tx *sql.Tx, rackID, excludeItemID, position, units int64) (string, error) {
	var totalUnits, revNums int64
	err := tx.QueryRowContext(ctx, `SELECT COALESCE(usize, 0), COALESCE(revnums, 0) FROM racks WHERE id = ?`, rackID).Scan(&totalUnits, &revNums)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	spanOf := func(start, count int64) []int64 {
		out := make([]int64, 0, count)
		for offset := int64(0); offset < count; offset++ {
			if revNums == 1 {
				out = append(out, start+offset)
			} else {
				out = append(out, start-offset)
			}
		}
		return out
	}
	span := spanOf(position, units)
	for _, unit := range span {
		if unit < 1 || unit > totalUnits {
			return "机架位置超出所选机架范围", nil
		}
	}
	occupied := make(map[int64]bool)
	rows, err := tx.QueryContext(ctx, `SELECT COALESCE(rackposition, 0), COALESCE(usize, 0) FROM items WHERE rackid = ? AND id <> ?`, rackID, excludeItemID)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var otherPosition, otherUnits int64
		if err := rows.Scan(&otherPosition, &otherUnits); err != nil {
			return "", err
		}
		if otherPosition <= 0 || otherUnits <= 0 {
			continue
		}
		for _, unit := range spanOf(otherPosition, otherUnits) {
			occupied[unit] = true
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	conflicts := make([]int64, 0, len(span))
	seen := make(map[int64]bool)
	for _, unit := range span {
		if occupied[unit] && !seen[unit] {
			conflicts = append(conflicts, unit)
			seen[unit] = true
		}
	}
	sort.Slice(conflicts, func(i, j int) bool { return conflicts[i] < conflicts[j] })
	if len(conflicts) > 0 {
		texts := make([]string, 0, len(conflicts))
		for _, unit := range conflicts {
			texts = append(texts, strconv.FormatInt(unit, 10))
		}
		return fmt.Sprintf("机架行 %s 已被其他硬件占用", strings.Join(texts, "、")), nil
	}
	return "", nil
}
