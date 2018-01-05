# sibyl - Rapid Agile Estimations

[![Build Status](https://travis-ci.org/synacor/sibyl.svg?branch=master)](https://travis-ci.org/synacor/sibyl)
[![Go Report Card](https://goreportcard.com/badge/github.com/synacor/sibyl)](https://goreportcard.com/report/github.com/synacor/sibyl)

**Sibyl** is an online agile estimation tool that doesn't require sign-ups, entering user stories, or any other time consuming steps that keeps you from doing what matters: estimating on stories.

This repo contains both the back-end and front-end code for running Sibyl.

## Getting started

### Get Sibyl

To get **sibyl**:

```
% go get github.com/synacor/sibyl
```

### Run Sibyl

As long as `$GOPATH/bin` is in your `$PATH`, you can now run `sibyl`.

```
% sibyl
```

### Build Sibyl for Distribution

**Sibyl** uses the [rice.go](https://github.com/GeertJohan/go.rice) package to bundle in the templates and that static directory. When running on your local server, it will automatically pull it from the installed directory. If you
want to distribute your binary to other servers, you'll want to bundle up those assets. First, you'll need to install the `rice` command.

```
% go get github.com/GeertJohan/go.rice/rice
```

Now you can bundle up the assets in the binary.

```
% cd $GOPATH/src/github.com/synacor/sibyl
% go build
% rice append --exec sibyl
```

The binary `./sibyl` can now be distributed.

## Configuration

Sibyl uses [viper](https://github.com/spf13/viper) for configuration. The following environment variales are supported:

* `SIBYL_PORT`: Specify the port to run sibyl on. Defaults to `5000`.
* `SIBYL_TLS_PORT`: Specify the TLS port to run sibyl on. By default, Sibyl does not use TLS.
* `SIBYL_DEBUG`: Outputs additional log details.

Extended configuration can be supplied by created a `config.json` file in either of the following two locations:

* `./config.json`
* `/etc/sibyl/config.json`

Only the first config file found will be used.

The following example JSON file contains all the options and their defaults:

```
{
    "port": 5000,
    "tls_port": 0,
    "force_tls": false,
    "tls_private_key": "",
    "tls_public_key": ""
}
```

* `port`: The port to use for HTTP (non-TLS) traffic.
* `tls_port`: The port to use for HTTPS (TLS) traffic. Will only turn on TLS support if specified. If you use this option, you need to also specify `tls_private_key` and `tls_public_key`.
* `force_tls`: If using TLS, redirect non-TLS traffic to use TLS with a permanent redirect.
* `tls_private_key`: Path to the private key file.
* `tls_public_key`: Path to the public key file.

## Known Issues

When running the server over HTTP (non-TLS), some antivirus applications that buffer http connections, such as Kaspersky, may cause the web socket connection to disconnect. The workaround is to either run the server with HTTPS, or to disable port 80 filtering in your antivirus.

## Contributing

All ideas and contributions are appreciated.

## License

GNU AGPLv3 License, please see [LICENSE](LICENSE) for details.
