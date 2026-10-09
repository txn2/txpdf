package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	wk "github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var dimensionCases = []struct {
	name        string
	options     string
	decodeError bool
	valid       bool
}{
	{"default", `{}`, false, true},
	{"custom", `{"page_width_mm":147,"page_height_mm":210}`, false, true},
	{"decimal", `{"page_width_mm":147.5,"page_height_mm":210.25}`, false, true},
	{"both null", `{"page_width_mm":null,"page_height_mm":null}`, false, true},
	{"width null", `{"page_width_mm":null}`, false, true},
	{"height null", `{"page_height_mm":null}`, false, true},
	{"missing height", `{"page_width_mm":147}`, false, false},
	{"missing width", `{"page_height_mm":210}`, false, false},
	{"null height", `{"page_width_mm":147,"page_height_mm":null}`, false, false},
	{"null width", `{"page_width_mm":null,"page_height_mm":210}`, false, false},
	{"zero width", `{"page_width_mm":0,"page_height_mm":210}`, false, false},
	{"zero height", `{"page_width_mm":147,"page_height_mm":0}`, false, false},
	{"negative width", `{"page_width_mm":-147,"page_height_mm":210}`, false, false},
	{"negative height", `{"page_width_mm":147,"page_height_mm":-210}`, false, false},
	{"string width", `{"page_width_mm":"147mm","page_height_mm":210}`, true, false},
	{"string height", `{"page_width_mm":147,"page_height_mm":"210"}`, true, false},
	{"boolean width", `{"page_width_mm":true,"page_height_mm":210}`, true, false},
	{"boolean height", `{"page_width_mm":147,"page_height_mm":false}`, true, false},
	{"array width", `{"page_width_mm":[],"page_height_mm":210}`, true, false},
	{"array height", `{"page_width_mm":147,"page_height_mm":[]}`, true, false},
	{"object width", `{"page_width_mm":{},"page_height_mm":210}`, true, false},
	{"object height", `{"page_width_mm":147,"page_height_mm":{}}`, true, false},
	{"overflow width", `{"page_width_mm":1e400,"page_height_mm":210}`, true, false},
	{"overflow height", `{"page_width_mm":147,"page_height_mm":1e400}`, true, false},
}

func TestPageDimensionsValidation(t *testing.T) {
	for _, test := range dimensionCases {
		t.Run(test.name, func(t *testing.T) {
			var options Options
			err := json.Unmarshal([]byte(test.options), &options)
			if (err != nil) != test.decodeError {
				t.Fatalf("decode error=%v, expected error=%v", err, test.decodeError)
			}
			if err == nil && (options.validatePageDimensions() == nil) != test.valid {
				t.Fatalf("unexpected validation result for %s", test.options)
			}
		})
	}
	for _, invalid := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		valid := 210.0
		for _, options := range []Options{
			{PageWidthMM: &invalid, PageHeightMM: &valid},
			{PageWidthMM: &valid, PageHeightMM: &invalid},
		} {
			if options.validatePageDimensions() == nil {
				t.Fatal("nonfinite dimensions must be rejected")
			}
		}
	}
}

func TestPageDimensionsGeneratorArguments(t *testing.T) {
	pdfg := wk.NewPDFPreparer()
	// Preserve existing global and document options when dimensions are omitted.
	pdfg.MarginLeft.Set(10)
	pdfg.Cover.Input = "cover.html"
	pdfg.TOC.Include = true
	page := wk.NewPage("page.html")
	page.FooterRight.Set("Page [page]")
	pdfg.AddPage(page)
	before := pdfg.Args()
	Options{}.applyPageDimensions(pdfg)
	if !reflect.DeepEqual(before, pdfg.Args()) {
		t.Fatal("omitted dimensions changed default arguments")
	}
	for _, test := range []struct {
		width, height         float64
		wantWidth, wantHeight string
	}{
		{147, 210, "147mm", "210mm"},
		{147.5, 210.25, "147.5mm", "210.25mm"},
	} {
		Options{PageWidthMM: &test.width, PageHeightMM: &test.height}.applyPageDimensions(pdfg)
		args := pdfg.Args()
		for flag, expected := range map[string]string{
			"--page-width": test.wantWidth, "--page-height": test.wantHeight,
		} {
			if !hasArgument(args, flag, expected) {
				t.Fatalf("missing %s %s in %v", flag, expected, args)
			}
		}
		if !reflect.DeepEqual(before, withoutDimensions(args)) {
			t.Fatal("custom dimensions changed unrelated arguments")
		}
	}
}

func hasArgument(args []string, flag, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == flag && args[i+1] == value {
			return true
		}
	}
	return false
}

func withoutDimensions(args []string) []string {
	var result []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--page-width" || args[i] == "--page-height" {
			i++
			continue
		}
		result = append(result, args[i])
	}
	return result
}

// A stub checks the HTTP handler and actual process arguments without rendering
// a PDF or requiring wkhtmltopdf. Real PDF dimensions are checked separately.
func stubPDFGenerator(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	capture := filepath.Join(dir, "args")
	t.Setenv("TXPDF_TEST_ARGS", capture)
	binary := filepath.Join(dir, "wkhtmltopdf")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\" > \"$TXPDF_TEST_ARGS\"\nprintf '%%PDF-test'\n"), 0700); err != nil {
		t.Fatal(err)
	}
	previous := wk.GetPath()
	wk.SetPath(binary)
	t.Cleanup(func() { wk.SetPath(previous) })
	return capture
}

func pdfRequest(body string) *httptest.ResponseRecorder {
	router := gin.New()
	router.POST("/getPdf", getPDFHandler(zap.NewNop(), ""))
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/getPdf", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(response, request)
	return response
}

func TestPageDimensionsHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	capture := stubPDFGenerator(t)
	for _, test := range dimensionCases {
		t.Run(test.name, func(t *testing.T) {
			os.Remove(capture)
			response := pdfRequest(`{"options":` + test.options + `,"pages":[{"location":"page.html"}]}`)
			want := http.StatusBadRequest
			if test.valid {
				want = http.StatusOK
			}
			if response.Code != want {
				t.Fatalf("status=%d, want=%d: %s", response.Code, want, response.Body.String())
			}
			if test.valid {
				if response.Header().Get("Content-Type") != "application/pdf" || response.Body.String() != "%PDF-test" {
					t.Fatalf("unexpected PDF response: %v %s", response.Header(), response.Body.String())
				}
				return
			}
			var ack struct {
				Success     bool   `json:"success"`
				PayloadType string `json:"payload_type"`
				Payload     string `json:"payload"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &ack); err != nil {
				t.Fatal(err)
			}
			if ack.Success || ack.PayloadType != "PageDimensionsError" || ack.Payload == "" {
				t.Fatalf("unexpected error response: %s", response.Body.String())
			}
			if _, err := os.Stat(capture); !os.IsNotExist(err) {
				t.Fatal("invalid dimensions invoked the PDF generator")
			}
		})
	}
}

func TestPageDimensionsHTTPCompatibility(t *testing.T) {
	gin.SetMode(gin.TestMode)
	capture := stubPDFGenerator(t)
	body := `{"cover":{"location":"cover.html"},"toc":true,"pages":[{"Location":"{{ printf "page.html" }}"},{"location":"second.html"}],"unknown":true,"options":{"footer_left":"left","footer_right":"right","custom_headers":{"X-Test":"value"},"custom_header_propagation":true,"print_media_type":true,"toc_header_text":"Contents","toc_xsl_skip":true,"javascript_delay":10,"no_background":true,"disable_javascript":true,"unknown":true%s}}`
	var before []string
	for _, extra := range []string{"", `,"page_width_mm":null,"page_height_mm":null`, `,"page_width_mm":147.5,"page_height_mm":210.25`} {
		response := pdfRequest(strings.Replace(body, "%s", extra, 1))
		if response.Code != http.StatusOK {
			t.Fatalf("%d: %s", response.Code, response.Body.String())
		}
		data, err := os.ReadFile(capture)
		if err != nil {
			t.Fatal(err)
		}
		args := strings.Split(strings.TrimSpace(string(data)), "\n")
		for argument, value := range map[string]string{
			"cover":             "cover.html",
			"page":              "page.html",
			"--toc-header-text": "Contents",
			"--custom-header":   "X-Test",
			"X-Test":            "value",
			"--footer-left":     "left",
			"--footer-right":    "right",
		} {
			if !hasArgument(args, argument, value) {
				t.Fatalf("missing existing argument %s %s in %v", argument, value, args)
			}
		}
		for _, flag := range []string{"toc", "--custom-header-propagation", "--print-media-type"} {
			found := false
			for _, argument := range args {
				if argument == flag {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("missing existing argument %s in %v", flag, args)
			}
		}
		if before == nil {
			before = args
		}
		if !reflect.DeepEqual(before, withoutDimensions(args)) {
			t.Fatalf("existing arguments changed: %v versus %v", before, args)
		}
		if extra != "" && !strings.Contains(extra, "null") && (!hasArgument(args, "--page-width", "147.5mm") || !hasArgument(args, "--page-height", "210.25mm")) {
			t.Fatalf("custom dimensions missing: %v", args)
		}
	}
	for _, body := range []string{
		`{"options":{"javascript_delay":"invalid"}}`,
		`{"options":{"print_media_type":"invalid"}}`,
		`{"pages":false}`,
		`{"options":`,
	} {
		response := pdfRequest(body)
		if response.Code != http.StatusInternalServerError || !strings.Contains(response.Body.String(), "UnmarshalError") {
			t.Fatalf("existing error behavior changed: %d %s", response.Code, response.Body.String())
		}
	}
}

func TestPDFHTTPDefaultOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	capture := stubPDFGenerator(t)
	var before []string
	for _, test := range []struct {
		name string
		body string
	}{
		{"empty options", `{"options":{},"pages":[{"location":"page.html"}]}`},
		{"omitted options", `{"pages":[{"location":"page.html"}]}`},
		{"null options", `{"options":null,"pages":[{"location":"page.html"}]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := pdfRequest(test.body)
			if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/pdf" || response.Body.String() != "%PDF-test" {
				t.Fatalf("unexpected PDF response: %d %v %s", response.Code, response.Header(), response.Body.String())
			}
			data, err := os.ReadFile(capture)
			if err != nil {
				t.Fatal(err)
			}
			args := strings.Split(strings.TrimSpace(string(data)), "\n")
			if before == nil {
				before = args
			}
			if !reflect.DeepEqual(before, args) {
				t.Fatalf("default arguments changed: %v versus %v", before, args)
			}
		})
	}
}

func TestPDFHTTPTemplateErrors(t *testing.T) {
	gin.SetMode(gin.TestMode)
	capture := stubPDFGenerator(t)
	for _, test := range []struct {
		name        string
		body        string
		payloadType string
	}{
		{"parse error", `{"pages":[{"location":"{{ missingFunction }}"}]}`, "PostFilterError"},
		{"execution error", `{"pages":[{"location":"{{ .MissingField }}"}]}`, "TemplateError"},
	} {
		t.Run(test.name, func(t *testing.T) {
			response := pdfRequest(test.body)
			if response.Code != http.StatusInternalServerError || !strings.HasPrefix(response.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("unexpected template error response: %d %v %s", response.Code, response.Header(), response.Body.String())
			}
			var ack struct {
				Success     bool   `json:"success"`
				PayloadType string `json:"payload_type"`
				Payload     string `json:"payload"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &ack); err != nil {
				t.Fatal(err)
			}
			// Existing template errors leave the acknowledgement's success flag set.
			if !ack.Success || ack.PayloadType != test.payloadType || ack.Payload == "" {
				t.Fatalf("existing template error behavior changed: %s", response.Body.String())
			}
			if _, err := os.Stat(capture); !os.IsNotExist(err) {
				t.Fatal("template error invoked the PDF generator")
			}
		})
	}
}
