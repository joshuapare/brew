# ML Developer CLI

Run common Mosaic Learning developer operations from a simple CLI tool.


## Commmands

```bash
mldev setup
```

Sets up an existing repository from a yaml spec at the root of the repo. Here is an example spec:

```yaml
# Name of this stack
name: "test"
# Name of the general project this iteration is a part of
project: "Testing"
# Networking options
network:
  domain: "dev.moslrn.net"
  subdomains:
  - address: "192.168.223.223"
    names:
    - testing1
    - testing2
    - testing3
```