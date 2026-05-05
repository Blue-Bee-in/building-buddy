package pdf

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/phpdave11/gofpdf"
)

// MaintenanceRequestData holds the fields rendered into a maintenance PDF.
type MaintenanceRequestData struct {
	ReferenceID    string    // e.g. "MR-2026-042"
	SocietyName    string    // e.g. "Ashoka Heights Co-op Hsg. Society Ltd."
	Title          string
	ContractorName string
	QuotedAmount   float64
	ApprovedAmount float64
	Description    string
	CreatedByName  string
	ApprovedByName string
	ApprovedAt     time.Time
	GeneratedAt    time.Time
}

// MaintenanceRequest generates a maintenance work authorisation PDF and
// returns the raw bytes. Uses Helvetica (no TTF embedding required for Phase 1).
func MaintenanceRequest(d *MaintenanceRequestData) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 0, 20)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()

	// ── Header block ──────────────────────────────────────────────────────────
	// Background: primary #1E2A3A = (30,42,58)
	pdf.SetFillColor(30, 42, 58)
	pdf.Rect(0, 0, 210, 50, "F")

	// Society name (caps, small)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(180, 186, 195)
	pdf.SetXY(20, 12)
	pdf.CellFormat(170, 5, strings.ToUpper(d.SocietyName), "", 2, "L", false, 0, "")

	// Document title
	pdf.SetFont("Helvetica", "B", 17)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetXY(20, 21)
	pdf.CellFormat(170, 9, "Maintenance Work Authorisation", "", 2, "L", false, 0, "")

	// Reference ID and date on same line
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(140, 150, 165)
	pdf.SetXY(20, 34)
	pdf.CellFormat(85, 5, fmt.Sprintf("Reference: %s", d.ReferenceID), "", 0, "L", false, 0, "")
	pdf.SetXY(105, 34)
	pdf.CellFormat(85, 5, fmt.Sprintf("Approved: %s", d.ApprovedAt.Format("02 Jan 2006")), "", 0, "R", false, 0, "")

	// ── Body: warm paper surface ──────────────────────────────────────────────
	// Surface #F5F4F0 = (245,244,240)
	pdf.SetFillColor(245, 244, 240)
	pdf.Rect(0, 50, 210, 207, "F")

	// Section header
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetTextColor(92, 102, 118)
	pdf.SetXY(20, 58)
	pdf.CellFormat(85, 5, "FIELD", "", 0, "L", false, 0, "")
	pdf.CellFormat(85, 5, "DETAILS", "", 0, "L", false, 0, "")

	// Divider line
	pdf.SetDrawColor(221, 217, 210)
	pdf.Line(20, 65, 190, 65)

	rows := [][]string{
		{"Description", d.Title},
		{"Contractor", d.ContractorName},
		{"Quoted Amount", formatINR(d.QuotedAmount)},
		{"Approved Amount", formatINR(d.ApprovedAmount)},
		{"Submitted By", d.CreatedByName},
		{"Approved By", d.ApprovedByName},
		{"Approval Date", d.ApprovedAt.Format("02 Jan 2006")},
		{"Status", "APPROVED"},
	}
	if d.Description != "" {
		rows = append([][]string{{"Work Notes", d.Description}}, rows...)
	}

	y := 68.0
	for i, row := range rows {
		if i%2 == 0 {
			pdf.SetFillColor(245, 244, 240)
		} else {
			pdf.SetFillColor(238, 237, 233)
		}
		pdf.Rect(20, y, 170, 9, "F")

		// Label column
		pdf.SetFont("Helvetica", "", 9)
		pdf.SetTextColor(92, 102, 118)
		pdf.SetXY(22, y+2)
		pdf.CellFormat(60, 5, row[0], "", 0, "L", false, 0, "")

		// Value column
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetTextColor(30, 42, 58)
		pdf.SetXY(85, y+2)
		pdf.CellFormat(103, 5, row[1], "", 0, "L", false, 0, "")

		pdf.Line(20, y+9, 190, y+9)
		y += 9
	}

	// ── Signature section ─────────────────────────────────────────────────────
	sigY := y + 16
	pdf.SetFillColor(245, 244, 240)

	pdf.SetDrawColor(30, 42, 58)
	pdf.SetLineWidth(0.3)

	// Left sig block
	pdf.Line(22, sigY+15, 90, sigY+15)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetTextColor(142, 154, 166)
	pdf.SetXY(22, sigY+17)
	pdf.CellFormat(68, 5, "SECRETARY SIGNATURE", "", 0, "L", false, 0, "")

	// Right sig block
	pdf.Line(110, sigY+15, 188, sigY+15)
	pdf.SetXY(110, sigY+17)
	pdf.CellFormat(78, 5, "CONTRACTOR ACKNOWLEDGEMENT", "", 0, "L", false, 0, "")

	// ── Footer ────────────────────────────────────────────────────────────────
	pdf.SetFillColor(255, 255, 255)
	pdf.Rect(0, 257, 210, 40, "F")
	pdf.SetDrawColor(221, 217, 210)
	pdf.SetLineWidth(0.2)
	pdf.Line(0, 258, 210, 258)

	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetTextColor(30, 42, 58)
	pdf.SetXY(20, 262)
	pdf.CellFormat(85, 5, "building\xB7buddy  \xe2\x80\x94  Secretary's Command Center", "", 0, "L", false, 0, "")

	pdf.SetFont("Helvetica", "", 7)
	pdf.SetTextColor(142, 154, 166)
	pdf.SetXY(105, 262)
	pdf.CellFormat(85, 5, fmt.Sprintf("Generated: %s", d.GeneratedAt.UTC().Format("2006-01-02T15:04:05Z")), "", 0, "R", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("pdf output: %w", err)
	}
	return buf.Bytes(), nil
}

// formatINR formats a float64 as Indian lakh-formatted currency.
// e.g. 125000 → "₹1,25,000"
func formatINR(amount float64) string {
	rupees := int64(amount)
	paise := int64((amount - float64(rupees)) * 100)

	s := fmt.Sprintf("%d", rupees)
	n := len(s)

	var out strings.Builder
	out.WriteString("\xe2\x82\xb9") // ₹ (UTF-8)

	if n <= 3 {
		out.WriteString(s)
	} else {
		// Last 3 digits, then groups of 2
		last3 := s[n-3:]
		rest := s[:n-3]
		var groups []string
		for len(rest) > 2 {
			groups = append([]string{rest[len(rest)-2:]}, groups...)
			rest = rest[:len(rest)-2]
		}
		if len(rest) > 0 {
			groups = append([]string{rest}, groups...)
		}
		out.WriteString(strings.Join(groups, ","))
		out.WriteString(",")
		out.WriteString(last3)
	}

	if paise > 0 {
		out.WriteString(fmt.Sprintf(".%02d", paise))
	}
	return out.String()
}
