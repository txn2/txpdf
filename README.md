![](https://raw.githubusercontent.com/txn2/txpdf/master/assets/mast.jpg)

[![](https://images.microbadger.com/badges/image/txn2/txpdf.svg)](https://microbadger.com/images/txn2/txpdf "n2pdf")
[![Docker Container Pulls](https://img.shields.io/docker/pulls/txn2/txpdf.svg)](https://hub.docker.com/r/txn2/txpdf/)


# txPDF

Check out the article [Webpage to PDF Microservice](https://mk.imti.co/webpage-to-pdf-microservice/) for a quick getting started guide.

[txPDF] is an HTML to PDF microservice by [txn2]. [txPDF] is built on top of the [n2pdf] container exposing an API endpoint that returns a PDF document from web based **POST** request.

Example Post Body:
```json
{
  "options": {
    "print_media_type": true
  },
  "pages": [
    {
      "Location": "https://www.example.com"
    }
  ]
}
```

## Custom Page Dimensions

These parameters require a server version that includes custom page dimensions. The official `txn2/txpdf:2.0.2` image does not include this functionality. To use the example below, run a server built from source with this feature or a release that includes it.

To set a custom paper size, provide `page_width_mm` and `page_height_mm` in `options`. Both values are expressed in millimeters and apply to the entire document, including the cover and table of contents.

Example Post Body:
```json
{
  "options": {
    "print_media_type": true,
    "page_width_mm": 147.5,
    "page_height_mm": 210.25
  },
  "pages": [
    {
      "location": "https://www.example.com"
    }
  ]
}
```

Both dimensions must be provided together as positive, finite JSON numbers. Integers and decimals are accepted; strings such as `"147mm"`, booleans, arrays, objects, zero, negative values and numbers outside the supported numeric range are rejected with HTTP **400** and a `PageDimensionsError` JSON response.

Omitted or `null` dimensions are treated as unset. When both are unset, the existing default paper size (A4) is preserved. If only one dimension contains a number, the request is rejected with HTTP **400**. Other PDF options retain their existing behavior.

If you want to convert web pages to PDF but do not need a web service you can use the [n2pdf] container directly as a command line tool.

## Security

[txPDF] is intended as a backend service and should only be accessible by other trusted services. [txPDF] allows the passing of template directives that can access environment or request data. This is useful and make
[txPDF] highly flexible. However please treat [txPDF] as a remote code executor and implement it accordingly. See [#2](https://github.com/txn2/txpdf/issues/2) 
for more on this issue.

## Docker Use

Run the [txPDF Docker container] on your local workstation for testing. Forward port **8080** or any free port to txPDFs default service port **8080** on the container.

```bash
docker run --rm -p 8080:8080 -e IP=0.0.0.0 txn2/txpdf:2.0.2
```

## Curl Test
```bash
curl -d "@examples/days.json" -X POST http://localhost:8080/getPdf --output test.pdf
```

[txPDF] can be configured with the following environment variables:

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| IP | 127.0.0.1 | Bind IP address |
| PORT | 8080 | Server listen port |
| DEBUG | false | Verbose logging |
| TOC_XSL | | Path to XSL transformation script for Table of Contents (example **./toc.xsl**. Container holds a default **./toc.xsl** |

## Test

```bash
curl -d "@examples/multi-site.json" -X POST http://localhost:8080/getPdf --output test.pdf
```


[n2pdf]:https://github.com/txn2/n2pdf
[txPDF]:https://github.com/txn2/txpdf
[txn2]:https://github.com/txn2
[txPDF Docker container]:https://hub.docker.com/r/txn2/txpdf/
