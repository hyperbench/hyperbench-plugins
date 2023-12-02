# Hyperbench-plugins

hyperbench-plugins is blockchain platforms adaptions for hyperbench to perform stress testing, used as plugin, written by go.

detail for [hyperbench](https://github.com/hyperbench/hyperbench).

## Building
Use Makefile to build

```bash
# suppose you are in the hyperbench-plugin/hyperchain directory
make build
```
This will create a `hyperchain.so` plugin file which can be used by hyperbench in current directory.