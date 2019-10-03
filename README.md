# blast wrappers

Blast wrappers for caching the output of blast searches.

These tools will search for any existing cached output and will return that if it exists. If not, it will pass all the arguments to the blast tools found in `/usr/bin/*blast*`.

The output in stored/cached into `/tmp/blastCacheStore`

```
    /tmp/blastCacheStore/
    └── 70d
        └── 35d
            └── 70d35d8c2c52155979a2ad66722ccc8c
                ├── errors.blast
                └── output.blast

```

The cache key (in the above example `70d35d8c2c52155979a2ad66722ccc8c`) is created using md5 on the arguments passed to the blast tool (or, rather, to the wrapper) and the *contents* of the file (given by `-query` argument).

This works only if you add these wrappers to the PATH...

## TODO
    - cache location should be configurable (via env variables)
    - use a store struct
