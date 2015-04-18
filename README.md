# Experimental golang-based 6502 machine emulator

This is my experimental [go](https://golang.org)-based emulator for my
[Buri](https://github.com/rjw57/buri) homebrew 6502 machine.

It's probably of little interest to others at the moment.

## Using picocom

```console
$ socat PTY,link=/tmp/a,raw,echo=0 PTY,link=/tmp/b,raw,echo=0
$ picocom --noinit /tmp/b
$ buri --serial /tmp/a $OTHER_OPTS
```
