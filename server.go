// Package main
//
// txn2.com
package main

import (
	"encoding/json"
	"net/http"
	"os"
	"text/template"
	"time"

	"bytes"

	"github.com/Masterminds/sprig"
	wk "github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/txn2/service/ginack"
	"go.uber.org/zap"
)

// Page
type Page struct {
	Location string `json:location`
}

type Options struct {
	FooterLeft              string            `json:"footer_left"`
	FooterLeftRight         string            `json:"footer_right"`
	CustomHeaders           map[string]string `json:"custom_headers"`
	CustomHeaderPropagation bool              `json:"custom_header_propagation"`
	PrintMediaType          bool              `json:"print_media_type"`
	TocHeaderText           string            `json:"toc_header_text"`
	TocXslSkip              bool              `json:"toc_xsl_skip"`
	NoBackground            bool              `json:"no_background"`
	JavascriptDelay         uint              `json:"javascript_delay"`
	DisableJavascript       bool              `json:"disable_javascript"`
}

// Cfg
type Cfg struct {
	Cover   Page    `json:"cover"`
	TOC     bool    `json:"toc"`
	Pages   []Page  `json:"pages"`
	Options Options `json:"options"`
}

func main() {
	// Default and consistent environment variables
	// help standardize k8s configs and documentation
	//
	port := getEnv("PORT", "8080")
	debug := getEnv("DEBUG", "false")
	basePath := getEnv("BASE_PATH", "")
	tocXsl := getEnv("TOC_XSL", "")

	gin.SetMode(gin.ReleaseMode)

	if debug == "true" {
		gin.SetMode(gin.DebugMode)
	}

	logger, err := zap.NewProduction()
	if err != nil {
		panic(err.Error())
	}

	if debug == "true" {
		logger, _ = zap.NewDevelopment()
	}

	// router
	r := gin.New()

	// middleware
	//
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))

	// route group for specified base path
	rg := r.Group(basePath)

	// routes
	//
	rg.GET("/",
		func(c *gin.Context) {
			// call external libs for business logic here
			ack := ginack.Ack(c)

			ack.SetPayload(gin.H{"message": "pdf generator"})

			// return
			c.JSON(ack.ServerCode, ack)
			return
		},
	)

	rg.POST("/getPdf",
		func(c *gin.Context) {
			ack := ginack.Ack(c)

			rs, err := c.GetRawData()
			if err != nil {
				logger.Error("PostError: " + err.Error())
				ack.ServerCode = 500
				ack.SetPayload(gin.H{"status": "fail", "error": err.Error()})
				c.JSON(ack.ServerCode, ack)
				return
			}

			// parse raw json
			tmpl, err := template.New("PostFilter").Funcs(sprig.TxtFuncMap()).Parse(string(rs))
			if err != nil {
				logger.Error("PostFilterError: " + err.Error())
				ack.ServerCode = 500
				ack.PayloadType = "PostFilterError"
				ack.SetPayload(err.Error())
				c.JSON(ack.ServerCode, ack)
				return
			}

			var tplReturn bytes.Buffer
			if err := tmpl.Execute(&tplReturn, c.Request); err != nil {
				logger.Error("TemplateError: " + err.Error())
				ack.ServerCode = 500
				ack.PayloadType = "TemplateError"
				ack.SetPayload(err.Error())
				c.JSON(ack.ServerCode, ack)
				return
			}

			cfg := &Cfg{}
			err = json.Unmarshal(tplReturn.Bytes(), cfg)
			if err != nil {
				logger.Error("UnmarshalError: " + err.Error())
				ack.ServerCode = 500
				ack.PayloadType = "UnmarshalError"
				ack.SetPayload(err.Error())
				c.JSON(ack.ServerCode, ack)
				return
			}

			// Create new PDF generator
			pdfg, err := wk.NewPDFGenerator()
			if err != nil {
				logger.Error("PDFGeneratorError: " + err.Error())
				ack.ServerCode = 500
				ack.PayloadType = "UnmarshalError"
				ack.SetPayload(err.Error())
				c.JSON(ack.ServerCode, ack)
				return
			}

			// configure a table of contents
			if cfg.TOC {
				logger.Info("Adding TOC")
				pdfg.TOC.Include = true
				pdfg.TOC.ExcludeFromOutline.Set(true)
				pdfg.TOC.LoadMediaErrorHandling.Set("ignore")
				pdfg.TOC.LoadErrorHandling.Set("ignore")

				if len(cfg.Options.TocHeaderText) > 0 {
					pdfg.TOC.TocHeaderText.Set(cfg.Options.TocHeaderText)
				}

				if tocXsl != "" && cfg.Options.TocXslSkip != true {
					pdfg.TOC.XslStyleSheet.Set(tocXsl)
				}
			}

			// add cover page if one exists
			if len(cfg.Cover.Location) > 1 {
				logger.Info("Adding cover: " + cfg.Cover.Location)
				pdfg.Cover.Input = cfg.Cover.Location
				pdfg.Cover.PrintMediaType.Set(true)
				pdfg.Cover.LoadErrorHandling.Set("ignore")
				pdfg.Cover.LoadMediaErrorHandling.Set("ignore")

				for k, v := range cfg.Options.CustomHeaders {
					logger.Info("Adding custom header to cover: k: " + k + " v: " + v)
					pdfg.Cover.CustomHeader.Set(k, v)
					pdfg.Cover.CustomHeaderPropagation.Set(cfg.Options.CustomHeaderPropagation)
				}
			}

			// loop through page locations
			for _, page := range cfg.Pages {
				logger.Info("Adding page: " + page.Location)
				p := &wk.Page{
					Input:       page.Location,
					PageOptions: wk.NewPageOptions(),
				}

				p.EnableTocBackLinks.Set(true)
				p.LoadErrorHandling.Set("ignore")
				p.LoadMediaErrorHandling.Set("ignore")
				p.FooterLeft.Set(cfg.Options.FooterLeft)
				p.FooterRight.Set(cfg.Options.FooterLeftRight)
				p.PrintMediaType.Set(cfg.Options.PrintMediaType)
				p.JavascriptDelay.Set(cfg.Options.JavascriptDelay)
				p.NoBackground.Set(cfg.Options.NoBackground)
				p.DisableJavascript.Set(cfg.Options.NoBackground)

				for k, v := range cfg.Options.CustomHeaders {
					logger.Info("Adding custom header to page: k: " + k + " v: " + v)
					p.CustomHeader.Set(k, v)
					p.CustomHeaderPropagation.Set(cfg.Options.CustomHeaderPropagation)
				}

				pdfg.AddPage(p)

			}

			// Create PDF document in internal buffer

			err = pdfg.Create()
			if err != nil {
				logger.Warn("RunError: " + err.Error())
				// since wkhtmltopdf is not respecting the ignore on
				// missing content we will suppress the error, log it and
				// assume the pdf is acceptable.

				// TODO: investigate
			}

			c.Header("Content-Type", "application/pdf")
			c.String(http.StatusOK, string(pdfg.Bytes()))
		},
	)

	// for external status check
	r.GET(basePath+"/status",
		func(c *gin.Context) {
			ack := ginack.Ack(c)
			p := gin.H{"message": "alive"}

			if c.Query("noack") == "true" {
				c.JSON(200, p)
				return
			}

			ack.SetPayload(p)
			c.JSON(ack.ServerCode, ack)
		},
	)

	// default no route
	r.NoRoute(func(c *gin.Context) {
		ack := ginack.Ack(c)
		ack.SetPayload(gin.H{"message": "not found"})
		ack.ServerCode = 404
		ack.Success = false

		// return
		c.JSON(ack.ServerCode, ack)
	})

	r.Run(":" + port)
}

// getEnv gets an environment variable or sets a default if
// one does not exist.
func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}

	return value
}
