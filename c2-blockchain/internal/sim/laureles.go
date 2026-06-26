package sim

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

// AccessRecord is one row from the Laureles residential access CSV (lab dataset).
type AccessRecord struct {
	ID            string
	Fecha         string
	Apto          string
	Torre         string
	Nombre        string
	TipoRegistro  string
	MedioAcceso   string
	Hora          string
	TipoTransporte string
	Placa         string
}

// LaurelesFeed replays access-control rows as simulated IoT gate events.
type LaurelesFeed struct {
	mu   sync.Mutex
	rows []AccessRecord
	idx  int
}

const DefaultLaurelesCSV = "data/LAURELES V2 AB 21.csv"

// LoadLaurelesCSV reads semicolon-separated access records (samples up to maxRows for lab).
func LoadLaurelesCSV(path string, maxRows int) (*LaurelesFeed, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.Comma = ';'
	r.LazyQuotes = true
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("csv header: %w", err)
	}
	var rows []AccessRecord
	for len(rows) < maxRows {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}
		if len(rec) < 10 {
			continue
		}
		rows = append(rows, AccessRecord{
			ID:             strings.TrimSpace(rec[0]),
			Fecha:          strings.TrimSpace(rec[1]),
			Apto:           strings.TrimSpace(rec[2]),
			Torre:          strings.TrimSpace(rec[3]),
			Nombre:         strings.TrimSpace(rec[4]),
			TipoRegistro:   strings.TrimSpace(rec[7]),
			MedioAcceso:    strings.TrimSpace(rec[8]),
			Hora:           strings.TrimSpace(rec[9]),
			TipoTransporte: strings.TrimSpace(rec[10]),
			Placa:          pickPlaca(rec),
		})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no rows parsed from %s", path)
	}
	return &LaurelesFeed{rows: rows}, nil
}

func pickPlaca(rec []string) string {
	if len(rec) > 11 {
		return strings.TrimSpace(rec[11])
	}
	return ""
}

// NextEvent returns the next simulated access IoT event from the CSV rotation.
func (f *LaurelesFeed) NextEvent() map[string]interface{} {
	f.mu.Lock()
	defer f.mu.Unlock()
	row := f.rows[f.idx%len(f.rows)]
	f.idx++
	summary := fmt.Sprintf("%s torre %s %s — %s (%s)",
		row.TipoRegistro, row.Torre, row.Apto, truncate(row.Nombre, 40), row.MedioAcceso)
	ev := map[string]interface{}{
		"type":            "iot_event",
		"device_id":       "sensor-access-gate",
		"device_type":     "access_reader",
		"zone":            "porteria-laureles",
		"payload_summary": summary,
		"torre":           row.Torre,
		"apto":            row.Apto,
		"tipo_registro":   row.TipoRegistro,
		"medio_acceso":    row.MedioAcceso,
		"transporte":      row.TipoTransporte,
		"source":          "laureles_csv",
	}
	if row.Placa != "" {
		ev["placa"] = row.Placa
	}
	return ev
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
