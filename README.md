# go-salpidae
## Overview
Generate hash signature for input file.

SHA256 is generated per block and collected in output.

Block size can be adjusted in arguments, 1MB by default.
Each block is calculated in separate thread.

## Example

Linux:
```bash
$ go build ./cmd/salpidae/
$ ./salpidae -i /tmp/sal-tst2435975305 -o /tmp/out -b 100
$ cat /tmp/out
2ee54ecf3e38e6a31b33f1a0ab4e7ec651004c44018b07c471916976a25a8764
$ sha256sum /tmp/sal-tst2435975305
2ee54ecf3e38e6a31b33f1a0ab4e7ec651004c44018b07c471916976a25a8764  /tmp/sal-tst2435975305
```
