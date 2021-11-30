# XSLT Transform

The XSLT Transform component needs libxml2 along with their transient dependencies installed.
We are basing on a debian image to add libxml2 and using it at `.ko.yaml` for XSLT Transform.

```console
docker build . -t gcr.io/triggermesh/debian-libxml:v0.1.0
docker push gcr.io/triggermesh/debian-libxml:v0.1.0
```