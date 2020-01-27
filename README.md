# Brainfuck Tunnel - Psiphon Pro Go Version

...


Requirements
------------

**Linux**

    golang
    redsocks

**Windows**

    golang
    proxifier


Install
-------

**Brainfuck Psiphon Pro Go**

    $ go get -v -u github.com/aztecrabbit/brainfuck-psiphon-pro-go

    $ cd ~/go/src/github.com/aztecrabbit/brainfuck-psiphon-pro-go
    $ go build -ldflags "-s -w"

**Psiphon Tunnel Core**

    $ go get -v -u github.com/Psiphon-Labs/psiphon-tunnel-core/ConsoleClient

    $ cd ~/go/src/github.com/Psiphon-Labs/psiphon-tunnel-core/ConsoleClient
    $ go build -ldflags "-s -w" -o ~/go/src/github.com/aztecrabbit/brainfuck-psiphon-pro-go/psiphon-tunnel-core


Usage
-----

**Linux**

    $ cd ~/go/src/github.com/aztecrabbit/brainfuck-psiphon-pro-go
    $ sudo ./brainfuck-psiphon-pro-go

or

    $ sudo ~/go/src/github.com/aztecrabbit/brainfuck-psiphon-pro-go/brainfuck-psiphon-pro-go


**Termux**

    $ cd ~/go/src/github.com/aztecrabbit/brainfuck-psiphon-pro-go
    $ ./brainfuck-psiphon-pro-go

or

    $ ~/go/src/github.com/aztecrabbit/brainfuck-psiphon-pro-go/brainfuck-psiphon-pro-go


Configurations
--------------

Run `./brainfuck-psiphon-pro-go` first to export all default settings.


### Pro Version

    ...
    "PsiphonCore": 1,
    "Psiphon": {
        "CoreName": "psiphon-tunnel-core",
        "Tunnel": 3,
        "Region": "SG",
        "Protocols": [
            "FRONTED-MEEK-HTTP-OSSH",
            "FRONTED-MEEK-OSSH"
        ],
        "TunnelWorkers": 8,
        "KuotaDataLimit": 0,
        "Authorizations": [
            "blablabla"
        ]
    }
    ...


### Rules

**Xl Iflix or Axis Gaming (Default)**

    ...
    "Rules": {
        "akamai.net:80": [
            "video.iflix.com",
            "videocdn-2.iflix.com",
            "iflix-videocdn-p1.akamaized.net",
            "iflix-videocdn-p2.akamaized.net",
            "iflix-videocdn-p3.akamaized.net",
            "iflix-videocdn-p6.akamaized.net",
            "iflix-videocdn-p7.akamaized.net",
            "iflix-videocdn-p8.akamaized.net"
        ]
    },
    ...

**Direct**

    ...
    "Rules": {
        "*:*": [
            "*"
        ]
    },
    ...

**Xl King**

    ...
    "Rules": {
        "akamai.net:80": [
            "www.pubgmobile.com"
        ]
    },
    ...

**Telkomsel 0P0K**

    ...
    "Rules": {
        "akamai.net:443": [
            "118.97.159.51:443",
            "118.98.95.106:443"
        ]
    },
    ...
