![](./assets/mast.jpg)

[![](https://images.microbadger.com/badges/image/txn2/txpdf.svg)](https://microbadger.com/images/txn2/txpdf "n2pdf")

# txPDF

txPDF is an HTML to PDF microservice by [txn2]

## Docker Use

Run the [txPDF Docker container] on your local workstation for testing. Forward port **8080** or any free port to txPDFs default service port **8080** on the container.

```bash
docker run --rm -p 8080:8080 -e DEBUG=true txn2/txpdf
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

[txPDF]:https://github.com/txn2/txpdf
[txn2]:https://github.com/txn2
[txPDF Docker container]:https://hub.docker.com/r/txn2/txpdf/