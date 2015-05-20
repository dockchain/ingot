# ingot

Dockchain is a set of tools to capture [Docker](https://docker.com) events, sign them,
and then forward the events to a service.

## How?

Ingot is the lightweight client that sits in a container either on each
host across a cluster of machines or an a host that can talk to a
[Docker Swarm](https://docs.docker.com/swarm/) and hooks into the
[Docker events](http://docs.docker.com/reference/api/docker_remote_api_v1.18/#monitor-dockers-events) and captures each event.

For `create` events, Ingot also gets [image information](http://docs.docker.com/reference/api/docker_remote_api_v1.18/#inspect-an-image) and includes that information
in the data captured.

The events and image information are assembled into a payload (JSON)
and the payload is digitally signed with an RSA key and posted
to an external service (for example [DockCha.in](https://dockcha.in) ).

## Why?

For the most part, we're all happy with our logs and various
log aggregation services. But when you're in a regulated
industry, having an [irrefutable](http://dictionary.reference.com/browse/irrefutable) log of the code run on your cluster on any date is
a very useful tool. Logging Docker events and image information
to a central service that then hashes the information and
publishes the hashes makes it much, much harder for an intruder
to break in **and** destroy log history.

## Code

The Ingot code is fairly simple Go code that listens to events,
aggregates them, signs them, and POSTs them to a target service.
By default, the target service is DockChain, but the target can
be changed with a configuration option.

This is [David Pollak](https://github.com/dpp)'s first Go project,
so the code will be both non-idiomatic and probably crapping. Feel
free to open pull requests.

## How/Where to communicate

Right now, open a GitHub ticket and we'll talk that way...
or via [Gitter](https://gitter.im/dockchain/ingot).

[![Join the chat at https://gitter.im/dockchain/ingot](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/dockchain/ingot?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

