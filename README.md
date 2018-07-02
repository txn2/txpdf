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

If you want to convert web pages to PDF but do not need a web service you can use the [n2pdf] container directly as a command line tool.

## Docker Use

Run the [txPDF Docker container] on your local workstation for testing. Forward port **8080** or any free port to txPDFs default service port **8080** on the container.

```bash
docker run --rm -p 8080:8080 -e DEBUG=true txn2/txpdf
```

## Curl Test
```bash
curl -d "@examples/days.json" -X POST http://localhost:8080/getPdf --output test.pdf
```

[txPDF] can be configured with the following environment variables:

| Variable | Default | Purpose |
| -------- | ------- | ------- |
| PORT | 8080 | Server listen port |
| DEBUG | false | Verbose logging |
| BASE_PATH |  | Base path for routes. Prepends onto web service routes  **BASE_PATH**/getPdf and **BASE_PATH**/status |
| TOC_XSL | | Path to XSL transformation script for Table of Contents (example **./toc.xsl**. Container holds a default **./toc.xsl** |

## Test

```bash
curl -d "@examples/multi-site.json" -X POST http://localhost:8080/getPdf --output test.pdf
```

[n2pdf]:https://github.com/txn2/n2pdf
[txPDF]:https://github.com/txn2/txpdf
[txn2]:https://github.com/txn2
[txPDF Docker container]:https://hub.docker.com/r/txn2/txpdf/
