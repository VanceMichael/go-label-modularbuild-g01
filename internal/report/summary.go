package report

import (
	"github.com/VanceMichael/go-base-modularbuild-g01/internal/domain"
	"sort"
	"time"
)

type ModuleMoveRow struct {
	Reference string
	Status    domain.ModuleMoveStatus
	WeightKg  int64
	CreatedAt time.Time
}
type Summary struct {
	Total    int
	WeightKg int64
	ByStatus map[domain.ModuleMoveStatus]int
	Latest   []ModuleMoveRow
}

func Build(rows []ModuleMoveRow, limit int) Summary {
	if limit < 1 {
		limit = 10
	}
	out := Summary{ByStatus: map[domain.ModuleMoveStatus]int{}}
	copyRows := append([]ModuleMoveRow(nil), rows...)
	sort.Slice(copyRows, func(i, j int) bool { return copyRows[i].CreatedAt.After(copyRows[j].CreatedAt) })
	for _, r := range copyRows {
		out.Total++
		out.WeightKg += r.WeightKg
		out.ByStatus[r.Status]++
	}
	if len(copyRows) > limit {
		copyRows = copyRows[:limit]
	}
	out.Latest = copyRows
	return out
}
func Window(rows []ModuleMoveRow, start, end time.Time) []ModuleMoveRow {
	out := make([]ModuleMoveRow, 0)
	for _, r := range rows {
		if !r.CreatedAt.Before(start) && r.CreatedAt.Before(end) {
			out = append(out, r)
		}
	}
	return out
}
