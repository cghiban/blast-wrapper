#!/bin/bash


containerid=""
if [[ -n $1 ]]; then
    containerid=$(docker ps -q| grep $1)
fi

for tool in blastn blastp blastx tblastn tblastp; do
    go build -o ./$tool main.go
    strip ./$tool
    ls -l ./$tool
done

#ls -lh blastn blastp blastx tblastn tblastp
if [[ -n $containerid ]]; then
    for tool in blastn blastp blastx tblastn tblastp; do
        docker cp $tool $containerid:/usr/local/bin
    done
fi
