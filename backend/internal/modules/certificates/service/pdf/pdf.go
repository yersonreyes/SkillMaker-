// Package pdf provides the certificate PDF rendering for the certificates module.
// Uses go-pdf/fpdf to generate an A4 landscape typographic certificate.
// Branding is a VECTOR brandmark drawn with fpdf primitives (no raster asset to
// source/embed): a cyan disc with a white "play" glyph + the "SkillMaker" wordmark.
package pdf

import (
	"bytes"
	"time"

	"github.com/go-pdf/fpdf"
)

// RenderCertificate generates a PDF certificate for the given user/course data.
// Returns the PDF as a byte slice. No file is written to disk.
// The returned bytes are opaque (PDF binary); callers should not interpret the content.
func RenderCertificate(nombre, titulo string, fecha time.Time, codigo string) ([]byte, error) {
	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "L", // Landscape
		UnitStr:        "mm",
		SizeStr:        "A4",
	})
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	// ── Background header band ─────────────────────────────────────────────────
	// Cyanotype midnight ink tone (R:18, G:27, B:44 approximation).
	pdf.SetFillColor(18, 27, 44)
	pdf.Rect(0, 0, 297, 30, "F")

	// ── Brandmark (top-left of the header band) ─────────────────────────────────
	drawBrandmark(pdf)

	// ── Header title ──────────────────────────────────────────────────────────
	pdf.SetTextColor(200, 230, 240) // cyan-ish
	pdf.SetFont("Helvetica", "B", 18)
	pdf.SetXY(0, 8)
	pdf.CellFormat(297, 14, "Certificado de Finalización", "", 0, "C", false, 0, "")

	// ── Body ──────────────────────────────────────────────────────────────────
	pdf.SetTextColor(18, 27, 44) // midnight ink

	// Subtitle — "This certifies that"
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(20, 45)
	pdf.CellFormat(257, 8, "Este certificado acredita que", "", 0, "C", false, 0, "")

	// User nombre (prominent).
	pdf.SetFont("Helvetica", "B", 20)
	pdf.SetXY(20, 57)
	pdf.CellFormat(257, 12, nombre, "", 0, "C", false, 0, "")

	// Divider line.
	pdf.SetDrawColor(0, 172, 193) // cyan accent
	pdf.SetLineWidth(0.5)
	pdf.Line(60, 72, 237, 72)

	// Completion phrase.
	pdf.SetFont("Helvetica", "", 11)
	pdf.SetXY(20, 76)
	pdf.CellFormat(257, 8, "ha completado satisfactoriamente el curso", "", 0, "C", false, 0, "")

	// Course titulo (prominent).
	pdf.SetFont("Helvetica", "B", 16)
	pdf.SetXY(20, 87)
	pdf.CellFormat(257, 10, titulo, "", 0, "C", false, 0, "")

	// Issue date.
	pdf.SetFont("Helvetica", "", 10)
	pdf.SetXY(20, 104)
	pdf.CellFormat(257, 7,
		"Emitido el "+fecha.Format("02 de January de 2006"),
		"", 0, "C", false, 0, "")

	// ── Footer ────────────────────────────────────────────────────────────────
	// Footer band.
	pdf.SetFillColor(240, 245, 250)
	pdf.Rect(0, 178, 297, 12, "F")

	// Verification code (centered).
	pdf.SetTextColor(100, 120, 140)
	pdf.SetFont("Helvetica", "", 8)
	pdf.SetXY(0, 180)
	pdf.CellFormat(297, 7, "Código de verificación: "+codigo, "", 0, "C", false, 0, "")

	// Brand line (footer left) — balances the centered code with the issuer name.
	pdf.SetTextColor(0, 130, 150) // cyan, slightly darker for the light footer band
	pdf.SetFont("Helvetica", "B", 8)
	pdf.SetXY(20, 180)
	pdf.CellFormat(120, 7, "SkillMaker", "", 0, "L", false, 0, "")

	// ── Render to bytes ────────────────────────────────────────────────────────
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// drawBrandmark renders the SkillMaker brandmark at the top-left of the header band:
// a cyan disc with a white "play" triangle (video-LMS motif) + the "SkillMaker"
// wordmark and a trailing cyan dot (mirrors the app's `tk-dot` accent). Pure vector —
// no embedded image asset.
func drawBrandmark(pdf *fpdf.Fpdf) {
	const cyanR, cyanG, cyanB = 0, 172, 193

	// Cyan disc.
	pdf.SetFillColor(cyanR, cyanG, cyanB)
	pdf.Circle(27, 15, 6, "F")

	// White play triangle inside the disc.
	pdf.SetFillColor(255, 255, 255)
	pdf.Polygon([]fpdf.PointType{
		{X: 25, Y: 11.8},
		{X: 25, Y: 18.2},
		{X: 30.8, Y: 15},
	}, "F")

	// Wordmark.
	pdf.SetTextColor(235, 243, 250) // near-white on the midnight band
	pdf.SetFont("Helvetica", "B", 15)
	pdf.SetXY(36, 9.2)
	pdf.CellFormat(60, 12, "SkillMaker", "", 0, "L", false, 0, "")

	// Trailing cyan dot, positioned just after the wordmark.
	w := pdf.GetStringWidth("SkillMaker")
	pdf.SetFillColor(cyanR, cyanG, cyanB)
	pdf.Circle(36+w+2.0, 17.6, 1.2, "F")
}
