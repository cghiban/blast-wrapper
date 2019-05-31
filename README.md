# blast wrappers

Blast wrappers for saving the output of the blast search.

These tools will search for any existing cached output and will return that if it exists. If not, it will pass all the arguments to the blast tools found in `/usr/bin*blast*`.

The cacheoutput in stored into `/tmp/blastCacheStore`

```
$ tree /tmp/blastCacheStore/
└── 459
    └── 3c9
        └── 4593c92f4d385c6bffa7cb40c06a9f663d94ae9c824c0eef989cf5688d1eb775
            ├── errors.blast
            └── output.blast
```

The cache key (in the above example `4593c92f4d385c6bffa7cb40c06a9f663d94ae9c824c0eef989cf5688d1eb775`) is bases on the arguments passed to the blast tool (or the wrapper) and the contents of the file (given by `-query` argument).
